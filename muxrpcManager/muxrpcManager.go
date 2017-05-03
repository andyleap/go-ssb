package muxrpcManager

import (
	"encoding/json"
	"io"
	"sync"

	"github.com/andyleap/go-ssb"
	"github.com/andyleap/muxrpc"
)

type ExtraData struct {
	Lock  sync.Mutex
	Conns map[ssb.Ref]*muxrpc.Conn
}

func init() {
	ssb.RegisterInit(func(ds *ssb.DataStore) {
		ed := &ExtraData{Conns: map[ssb.Ref]*muxrpc.Conn{}}
		ds.SetExtraData("muxrpcConns", ed)
	})
}

func HandleConn(ds *ssb.DataStore, ref ssb.Ref, conn io.ReadWriteCloser) {
	ed := ds.ExtraData("muxrpcConns").(*ExtraData)

	handlers := ds.ExtraData("muxrpcHandlers").(map[string]func(conn *muxrpc.Conn, req int32, args json.RawMessage))

	muxConn := muxrpc.New(conn, handlers)

	ed.Lock.Lock()
	ed.Conns[ref] = muxConn
	ed.Lock.Unlock()

	onConnect, onConnectOK := ds.ExtraData("muxrpcOnConnect").(func(conn *muxrpc.Conn))

	if onConnectOK {
		go onConnect(muxConn)
	}

	go func() {
		muxConn.Handle()
		ed.Lock.Lock()
		delete(ed.Conns, ref)
		ed.Lock.Unlock()
	}()
}
