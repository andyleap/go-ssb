package rpc

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"reflect"

	"github.com/andyleap/go-ssb"
)

type Request struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
	ID     interface{}     `json:"id"`
}

type Response struct {
	Result interface{} `json:"result"`
	Error  interface{} `json:"error"`
	ID     interface{} `json:"id"`
}

func ServeConn(datastore *ssb.DataStore, conn io.ReadWriteCloser) {
	reader := json.NewDecoder(conn)
	writer := json.NewEncoder(conn)
	resp := make(chan interface{})
	defer conn.Close()
	go func() {
		for v := range resp {
			writer.Encode(v)
		}
	}()
	for {
		var req Request
		err := reader.Decode(&req)
		if err != nil {
			return
		}
		RPCMethods, ok := datastore.ExtraData("RPCMethods").(map[string]interface{})
		if !ok {
			if req.ID != nil {
				resp <- Response{Result: nil, Error: "No such method", ID: req.ID}
			}
			continue
		}
		method, ok := RPCMethods[req.Method]
		if !ok {
			if req.ID != nil {
				resp <- Response{Result: nil, Error: "No such method", ID: req.ID}
			}
			continue
		}
		go func() {
			defer func() {
				if r := recover(); r != nil {
					if req.ID != nil {
						resp <- Response{Result: nil, Error: fmt.Sprintf("Panic while running method: %s", r), ID: req.ID}
					}
				}
			}()
			rval := reflect.ValueOf(method)
			if rval.Kind() != reflect.Func {
				if req.ID != nil {
					resp <- Response{Result: nil, Error: "No such method", ID: req.ID}
				}
				return
			}
			params := []reflect.Value{}
			decodeparams := []interface{}{}
			rtype := rval.Type()
			for l1 := 0; l1 < rtype.NumIn(); l1++ {
				pval := reflect.New(rtype.In(l1))
				params = append(params, pval.Elem())
				decodeparams = append(decodeparams, pval.Interface())
			}
			err := json.Unmarshal(req.Params, &decodeparams)
			if err != nil {
				if req.ID != nil {
					resp <- Response{Result: nil, Error: fmt.Sprintf("Error decoding method parameters: %s", err), ID: req.ID}
				}
			}
			ret := rval.Call(params)
			if req.ID != nil {
				if rtype.NumOut() == 2 {
					if ret[1].Interface() != nil {
						resp <- Response{Result: nil, Error: ret[1].Interface().(error).Error(), ID: req.ID}
					} else {
						resp <- Response{Result: ret[0], Error: nil, ID: req.ID}
					}
				} else {
					if rtype.Out(0).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
						if ret[0].Interface() != nil {
							resp <- Response{Result: nil, Error: ret[0].Interface().(error).Error(), ID: req.ID}
						} else {
							resp <- Response{Result: true, Error: nil, ID: req.ID}
						}
					} else {
						resp <- Response{Result: ret[0], Error: nil, ID: req.ID}
					}
				}
			}
		}()
	}
}

func ListenAndServe(datastore *ssb.DataStore, n string, a string) error {
	l, err := net.Listen(n, a)
	defer l.Close()
	if err != nil {
		return err
	}
	for {
		c, err := l.Accept()
		if err != nil {
			return err
		}
		go func() {
			ServeConn(datastore, c)
		}()
	}
}
