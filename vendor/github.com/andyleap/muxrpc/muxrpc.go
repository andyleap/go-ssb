// muxrpc project muxrpc.go
package muxrpc

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/andyleap/muxrpc/codec"
)

type Conn struct {
	reader    *codec.Reader
	writer    *codec.Writer
	writeLock sync.Mutex

	current map[int32]func(*codec.Packet)
	curLock sync.Mutex

	handlers map[string]func(conn *Conn, req int32, args json.RawMessage)

	curReq int32

	Done chan struct{}
}

func New(c io.ReadWriteCloser, handlers map[string]func(conn *Conn, req int32, args json.RawMessage)) *Conn {
	return &Conn{
		reader: codec.NewReader(c),
		writer: codec.NewWriter(c),

		current: map[int32]func(*codec.Packet){},

		handlers: handlers,

		Done: make(chan struct{}),
	}
}

type Request struct {
	Name []string      `json:"name"`
	Args []interface{} `json:"args"`
	Type string        `json:"type,omitempty"`
}

func (mx *Conn) Send(p *codec.Packet) error {
	mx.writeLock.Lock()
	defer mx.writeLock.Unlock()
	return mx.writer.WritePacket(p)
}

func (mx *Conn) RegisterReturn(req int32, h func(*codec.Packet)) {
	mx.curLock.Lock()
	defer mx.curLock.Unlock()
	mx.current[req] = h
}

func (mx *Conn) DeregisterReturn(req int32) {
	mx.curLock.Lock()
	defer mx.curLock.Unlock()
	delete(mx.current, req)
}

func (mx *Conn) Handle() error {
	defer func() {
		close(mx.Done)
	}()
	for {
		p, err := mx.reader.ReadPacket()
		if err != nil {
			return err
		}
		mx.curLock.Lock()
		h, ok := mx.current[-p.Req]
		mx.curLock.Unlock()
		if ok {
			go h(p)
		} else {
			if p.Type == codec.JSON {
				var req struct {
					Name []string        `json:"name"`
					Args json.RawMessage `json:"args"`
					Type string          `json:"type,omitempty"`
				}
				json.Unmarshal(p.Body, &req)
				method := strings.Join(req.Name, ".")
				h, ok := mx.handlers[method]
				if ok {
					go h(mx, p.Req, req.Args)
				} else {
					go func(p *codec.Packet) {
						mx.Send(&codec.Packet{
							Req:    -p.Req,
							Type:   codec.String,
							EndErr: true,
							Stream: false,
							Body:   []byte("No such method available"),
						})
					}(p)
				}
			}
		}
	}
}

func (mx *Conn) Call(method string, reply interface{}, args ...interface{}) error {
	req := Request{
		Name: strings.Split(method, "."),
		Args: args,
		Type: "JSON",
	}
	reqBody, _ := json.Marshal(req)
	seq := atomic.AddInt32(&mx.curReq, 1)
	p := &codec.Packet{
		Req:  seq,
		Type: codec.JSON,
		Body: reqBody,
	}
	done := make(chan error)
	mx.RegisterReturn(seq, func(p *codec.Packet) {
		if p.EndErr {
			done <- fmt.Errorf("Error while performing call %s: %s", method, string(p.Body))
			return
		}
		json.Unmarshal(p.Body, &reply)
		done <- nil
	})
	defer mx.DeregisterReturn(seq)
	err := mx.Send(p)
	if err != nil {
		return err
	}
	return <-done
}

func (mx *Conn) Source(method string, reply func(p *codec.Packet), args ...interface{}) error {
	req := Request{
		Name: strings.Split(method, "."),
		Args: args,
		Type: "source",
	}
	reqBody, _ := json.Marshal(req)
	seq := atomic.AddInt32(&mx.curReq, 1)
	p := &codec.Packet{
		Req:    seq,
		Type:   codec.JSON,
		Body:   reqBody,
		Stream: true,
	}
	done := make(chan error, 1)
	mx.RegisterReturn(seq, func(p *codec.Packet) {
		if p.EndErr {
			if len(p.Body) == 0 {
				done <- nil
			} else {
				done <- fmt.Errorf("Error while performing source %s: %s", method, string(p.Body))
			}
		} else {
			reply(p)
		}
	})
	defer mx.DeregisterReturn(seq)
	err := mx.Send(p)
	if err != nil {
		return err
	}
	select {
	case e := <-done:
		return e
	case <-mx.Done:
		return fmt.Errorf("Socket Closed")
	}
}
