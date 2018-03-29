/*
This file is part of go-muxrpc.

go-muxrpc is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

go-muxrpc is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with go-muxrpc.  If not, see <http://www.gnu.org/licenses/>.
*/

/* WIP/Ripoff off net/rpc

TODO: source streams
Endgame: codegen over muxrpc manifest
*/

package muxrpc

import (
	"encoding/json"
	"io"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/cryptix/go-muxrpc/codec"
	"github.com/go-kit/kit/log"
	"github.com/pkg/errors"
)

// ServerError represents an error that has been returned from
// the remote side of the RPC connection.
type ServerError string

func (e ServerError) Error() string {
	return string(e)
}

var ErrShutdown = errors.New("connection is shut down")

type Client struct {
	r *codec.Reader
	w *codec.Writer
	c io.ReadWriteCloser

	sendqueue chan *queuePacket

	mutex    sync.Mutex // protects following
	seq      int32
	pending  map[int32]*Call
	closing  bool // user has called Close
	shutdown bool // server has told us to stop

	log log.Logger // logging utility for unhandled calls etc

	handlers atomic.Value
}

type queuePacket struct {
	p    *codec.Packet
	sent chan error
}

type CallHandler func(json.RawMessage) interface{}

type SourceHandler func(json.RawMessage) chan interface{}

func NewClient(l log.Logger, rwc io.ReadWriteCloser) *Client {
	// TODO: pass in ctx
	c := Client{
		r:         codec.NewReader(rwc),
		w:         codec.NewWriter(rwc),
		c:         rwc,
		sendqueue: make(chan *queuePacket, 100),
		seq:       1,
		pending:   make(map[int32]*Call),

		log: log.With(l, "unit", "muxrpc"),
	}
	c.handlers.Store(map[string]interface{}{})

	return &c
}

func (c *Client) Handle() {
	go c.send()
	c.read()
}

func (client *Client) send() {
	for pktQ := range client.sendqueue {
		client.mutex.Lock()
		if client.shutdown || client.closing {
			client.mutex.Unlock()
			if pktQ.sent != nil {
				pktQ.sent <- ErrShutdown
			}
			return
		}
		client.mutex.Unlock()

		if err := client.w.WritePacket(pktQ.p); err != nil {
			if pktQ.sent != nil {
				pktQ.sent <- errors.Wrap(err, "muxrpc/call: WritePacket() failed")
			}
		} else {
			if pktQ.sent != nil {
				pktQ.sent <- nil
			}
		}
	}
}

func (client *Client) handleCall(pkt *codec.Packet) {
	var req struct {
		Name []string        `json:"name"`
		Args json.RawMessage `json:"args"`
		Type string          `json:"type,omitempty"`
	}
	if pkt.Type != codec.JSON {
		client.log.Log("event", "warning", "msg", "Non JSON call request!", "pkt", pkt)
		return
	}
	json.Unmarshal(pkt.Body, &req)
	method := strings.Join(req.Name, ".")
	handlers := client.handlers.Load().(map[string]interface{})

	if handler, ok := handlers[method]; ok {
		switch h := handler.(type) {
		case CallHandler:
			var retPacket codec.Packet
			retPacket.Req = -pkt.Req

			ret := h(req.Args)
			switch ret := ret.(type) {
			case string:
				retPacket.Body = []byte(ret)
				retPacket.Type = codec.String
			case []byte:
				retPacket.Body = ret
				retPacket.Type = codec.Buffer
			case error:
				if ret != nil {
					retPacket.Body = []byte(ret.Error())
					retPacket.EndErr = true
					retPacket.Type = codec.String
				}
			default:
				var err error
				retPacket.Body, err = json.Marshal(ret)
				retPacket.Type = codec.JSON
				if err != nil {
					retPacket.Body = []byte(err.Error())
					retPacket.EndErr = true
					retPacket.Type = codec.String
				}
			}
			client.sendqueue <- &queuePacket{p: &retPacket}
		case SourceHandler:
			ret := h(req.Args)
			for val := range ret {
				var retPacket codec.Packet
				retPacket.Req = -pkt.Req
				retPacket.Stream = true

				switch val := val.(type) {
				case string:
					retPacket.Body = []byte(val)
					retPacket.Type = codec.String
				case []byte:
					retPacket.Body = val
					retPacket.Type = codec.Buffer
				default:
					if encoder, ok := val.(interface {
						Encode() []byte
					}); ok {
						retPacket.Body = encoder.Encode()
					} else {
						retPacket.Body, _ = json.Marshal(val)
					}
					retPacket.Type = codec.JSON
				}
				client.sendqueue <- &queuePacket{p: &retPacket}
			}
			var retPacket codec.Packet
			retPacket.Req = -pkt.Req
			retPacket.EndErr = true
			retPacket.Stream = true
			client.sendqueue <- &queuePacket{p: &retPacket}
		}
	} else {
		var retPacket codec.Packet
		retPacket.Req = -pkt.Req
		retPacket.EndErr = true
		retPacket.Stream = false
		retPacket.Body = []byte("Unimplemented rpc call")
		retPacket.Type = codec.String
		client.sendqueue <- &queuePacket{p: &retPacket}
		client.log.Log("event", "warning", "msg", "Unimplemented rpc call", "method", method)
	}

}

func (client *Client) read() {
	var err error
	var pkt *codec.Packet
	for err == nil {
		pkt, err = client.r.ReadPacket()
		if err != nil {
			break
		}
		seq := -pkt.Req
		client.mutex.Lock()
		// TODO: this is... p2p! no srsly we might get called
		call, ok := client.pending[seq]
		client.mutex.Unlock()
		if ok {
			if call.handleResp(pkt) {
				client.mutex.Lock()
				delete(client.pending, seq)
				client.mutex.Unlock()
			}
		} else {
			go client.handleCall(pkt)
		}
	}
	// Terminate pending calls.
	client.mutex.Lock()
	client.shutdown = true
	closing := client.closing
	if err == io.EOF {
		if closing {
			err = ErrShutdown
		} else {
			err = io.ErrUnexpectedEOF
		}
	}
	for _, call := range client.pending {
		call.Error = err
		call.done()
	}
	client.mutex.Unlock()
	if err != io.EOF && !closing {
		client.log.Log("error", errors.Wrap(err, "rpc: client protocol error."))
	}

}

// Request is the Body value for rpc calls
// TODO: might fit into Call cleaner
type Request struct {
	Name []string      `json:"name"`
	Args []interface{} `json:"args"`
	Type string        `json:"type,omitempty"`
}

type Call struct {
	Method string // The name of the service and method to call.
	Type   codec.PacketType
	Args   []interface{} // The argument to the function (*struct).
	Reply  interface{}   // The reply from the function (*struct).
	Error  error         // After completion, the error status.
	Done   chan struct{} // Closes when call is complete.

	stream bool

	log log.Logger
}

func (call *Call) done() {
	close(call.Done)
}

func (call *Call) handleResp(pkt *codec.Packet) (seqDone bool) {
	var err error
	if !call.stream || (pkt.Stream && pkt.EndErr) {
		seqDone = true
	}

	switch {
	case pkt.EndErr:
		// TODO: difference between End and Error?
		if pkt.Stream {
			if len(pkt.Body) > 0 {
				call.Error = ServerError(string(pkt.Body))
			}
		} else {
			// We've got an error response. Give this to the request;
			// any subsequent requests will get the ReadResponseBody
			// error if there is one.
			call.Error = ServerError(string(pkt.Body))
		}
		call.done()
	default:
		switch pkt.Type {
		case codec.JSON:
			// todo there sure is a nicer way to structure this
			if call.stream {
				replyVal := reflect.ValueOf(call.Reply)
				if replyVal.Kind() != reflect.Chan {
					call.Error = errors.Wrap(err, "muxrpc: unmarshall error")
					call.done()
					return true
				}
				elemVal := reflect.New(replyVal.Type().Elem())
				elem := elemVal.Interface()

				if err := json.Unmarshal(pkt.Body, elem); err != nil {
					call.Error = errors.Wrap(err, "muxrpc: unmarshall error")
					call.done()
					return true
				}

				replyVal.Send(elemVal.Elem())
			} else {
				if err := json.Unmarshal(pkt.Body, call.Reply); err != nil {
					call.Error = errors.Wrap(err, "muxrpc: unmarshall error")
					call.done()
					return true
				}
				call.done()
				return true
			}

		case codec.String:
			if call.stream {
				strChan, ok := call.Reply.(chan string)
				if !ok {
					call.Error = errors.New("muxrpc: illegal reply argument. wanted (chan string)")
					call.done()
					return true
				}
				strChan <- string(pkt.Body)
			} else {
				sptr, ok := call.Reply.(*string)
				if !ok {
					call.Error = errors.New("muxrpc: illegal reply argument. wanted (*string)")
					call.done()
					return true
				}
				*sptr = string(pkt.Body)
				call.done()
				return true
			}

		default:
			call.Error = errors.Errorf("muxrpc: unhandled pkt.Type %s", pkt)
			call.done()
			return true
		}
	}
	return
}

// Go invokes the function asynchronously.  It returns the Call structure representing
// the invocation.  The done channel will signal when the call is complete by returning
// the same Call object.  If done is nil, Go will allocate a new channel.
func (client *Client) Go(call *Call, done chan struct{}) *Call {
	if done == nil {
		done = make(chan struct{}, 0) // unbuffered.
	}
	call.Done = done

	client.mutex.Lock()
	if client.shutdown || client.closing {
		call.Error = ErrShutdown
		client.mutex.Unlock()
		call.done()
		return call
	}
	seq := client.seq
	client.seq++
	client.pending[seq] = call
	client.mutex.Unlock()

	// Encode and send the request.
	var pkt codec.Packet
	pkt.Req = seq
	pkt.Type = call.Type // TODO: non JSON request

	var req Request

	req.Name = strings.Split(call.Method, ".")

	req.Args = call.Args
	if call.stream {
		req.Type = "source"
		pkt.Stream = true
	}

	var err error
	if pkt.Body, err = json.Marshal(req); err != nil {
		client.mutex.Lock()
		delete(client.pending, seq)
		client.mutex.Unlock()
		call.Error = errors.Wrap(err, "muxrpc/call: body json.Marshal() failed")
		call.done()
		return call
	}
	sent := make(chan error)
	go func() {
		err := <-sent
		if err != nil {
			call.Error = err
			call.done()
		}
	}()
	client.sendqueue <- &queuePacket{p: &pkt, sent: sent}
	return call
}

func (client *Client) addHandler(method string, handler interface{}) {
	handlers := client.handlers.Load().(map[string]interface{})
	newHandlers := map[string]interface{}{}
	for k, v := range handlers {
		newHandlers[k] = v
	}
	newHandlers[method] = handler
	client.handlers.Store(newHandlers)
}

func (client *Client) HandleCall(method string, handler CallHandler) {
	client.addHandler(method, handler)
}

func (client *Client) HandleSource(method string, handler SourceHandler) {
	client.addHandler(method, handler)
}

// Call invokes the named function, waits for it to complete, and returns its error status.
func (client *Client) Call(method string, reply interface{}, args ...interface{}) error {
	var c Call
	c.log = log.With(client.log, "unit", "muxrpc/call", "method", method)
	c.Method = method
	c.Args = args
	c.Reply = reply
	c.Type = codec.JSON // TODO: find other example
	client.Go(&c, nil)
	<-c.Done
	return c.Error
}

func (client *Client) Source(method string, reply interface{}, args ...interface{}) error {
	var c Call
	c.log = log.With(client.log, "unit", "muxrpc/source", "method", method)
	c.Method = method
	c.Args = args
	replyVal := reflect.ValueOf(reply)
	if replyVal.Kind() != reflect.Chan {
		return errors.Errorf("reply not a channel: %T", reply)
	}
	c.Reply = reply
	c.Type = codec.JSON
	c.stream = true
	client.Go(&c, nil)
	<-c.Done
	return c.Error
}

func (c *Client) Close() error {
	c.closing = true
	return c.w.Close() // also closes the underlying con
}
