package blobs

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/andyleap/go-ssb"
	"github.com/andyleap/go-ssb/muxrpcManager"
	"github.com/andyleap/muxrpc"
	"github.com/andyleap/muxrpc/codec"
)

func New(root string, ds *ssb.DataStore) *BlobStore {
	return &BlobStore{
		ds:   ds,
		Root: root,
		wait: sync.NewCond(&sync.Mutex{}),
		want: map[ssb.Ref]*want{},
	}
}

type want struct {
	amount int
	resp   chan *muxrpc.Conn
	done   chan struct{}
}

type BlobStore struct {
	ds   *ssb.DataStore
	Root string
	wait *sync.Cond

	want     map[ssb.Ref]*want
	wantLock sync.Mutex
}

func (bs *BlobStore) Add(data []byte) ssb.Ref {
	hash := sha256.Sum256(data)

	hexhash := hex.EncodeToString(hash[:])
	pre, hexhash := hexhash[:2], hexhash[2:]
	os.MkdirAll(filepath.Join(bs.Root, pre), 0777)
	ioutil.WriteFile(filepath.Join(bs.Root, pre, hexhash+".tmp"), data, 0777)
	os.Rename(filepath.Join(bs.Root, pre, hexhash+".tmp"), filepath.Join(bs.Root, pre, hexhash))

	bs.wait.Broadcast()
	r, _ := ssb.NewRef(ssb.RefBlob, hash[:], ssb.RefAlgoSha256)
	return r
}

func (bs *BlobStore) Has(r ssb.Ref) bool {
	hexhash := hex.EncodeToString(r.Raw())

	if len(hexhash) < 2 {
		return false
	}
	pre, hexhash := hexhash[:2], hexhash[2:]
	if _, err := os.Stat(filepath.Join(bs.Root, pre, hexhash)); !os.IsNotExist(err) {
		return true
	}
	return false
}

func (bs *BlobStore) Size(r ssb.Ref) int64 {
	hexhash := hex.EncodeToString(r.Raw())

	if len(hexhash) < 2 {
		return -1
	}
	pre, hexhash := hexhash[:2], hexhash[2:]
	if s, err := os.Stat(filepath.Join(bs.Root, pre, hexhash)); !os.IsNotExist(err) {
		return s.Size()
	}
	return -1
}

func (bs *BlobStore) WaitFor(r ssb.Ref) {
	if bs.Has(r) {
		return
	}
	bs.wait.L.Lock()
	defer bs.wait.L.Unlock()
	for !bs.Has(r) {
		bs.wait.Wait()
	}
}

func (bs *BlobStore) Get(r ssb.Ref) io.ReadCloser {
	if !bs.Has(r) {
		return nil
	}
	hexhash := hex.EncodeToString(r.Raw())
	pre, hexhash := hexhash[:2], hexhash[2:]
	f, _ := os.Open(filepath.Join(bs.Root, pre, hexhash))
	return f
}

func (bs *BlobStore) Want(r ssb.Ref) {
	bs.wantLock.Lock()
	defer bs.wantLock.Unlock()
	if bs.Has(r) {
		return
	}
	if _, ok := bs.want[r]; ok {
		return
	}
	w := &want{
		resp: make(chan *muxrpc.Conn),
		done: make(chan struct{}),
	}
	bs.want[r] = w
	go func() {
		for c := range w.resp {
			data := []byte{}
			err := c.Source("blobs.get", func(p *codec.Packet) {
				data = append(data, p.Body...)
			}, r)
			if err != nil {
				newR := bs.Add(data)
				if newR == r {
					break
				}
			}
		}
		close(w.done)
		bs.wantLock.Lock()
		defer bs.wantLock.Unlock()
		delete(bs.want, r)
	}()
	conns, ok := bs.ds.ExtraData("muxrpcConns").(*muxrpcManager.ExtraData)
	if ok {
		conns.Lock.Lock()
		defer conns.Lock.Unlock()
		for _, conn := range conns.Conns {
			go func(conn *muxrpc.Conn) {
				has := false
				err := conn.Call("blobs.has", &has, r)
				if err != nil {
					return
				}
				if has {
					w.resp <- conn
				}
			}(conn)
		}
	}
}

func init() {
	ssb.RegisterInit(func(ds *ssb.DataStore) {
		bs := New("blobs", ds)
		ds.SetExtraData("blobStore", bs)

		handlers, ok := ds.ExtraData("muxrpcHandlers").(map[string]func(conn *muxrpc.Conn, req int32, args json.RawMessage))
		if !ok {
			handlers = map[string]func(conn *muxrpc.Conn, req int32, args json.RawMessage){}
			ds.SetExtraData("muxrpcHandlers", handlers)
		}
		handlers["blobs.has"] = func(conn *muxrpc.Conn, req int32, rm json.RawMessage) {
			var r ssb.Ref
			args := []interface{}{&r}
			json.Unmarshal(rm, &args)
			buf, _ := json.Marshal(bs.Has(r))
			conn.Send(&codec.Packet{
				Req:  -req,
				Type: codec.JSON,
				Body: buf,
			})
		}
		handlers["blobs.get"] = func(conn *muxrpc.Conn, req int32, rm json.RawMessage) {
			var arg1 json.RawMessage
			args := []interface{}{&arg1}
			json.Unmarshal(rm, &args)
			var r ssb.Ref
			err := json.Unmarshal(arg1, &r)
			if err != nil {
				var robj struct {
					Hash ssb.Ref `json:"hash"`
					Key  ssb.Ref `json:"key"`
				}
				json.Unmarshal(arg1, &robj)
				if robj.Key.Type != ssb.RefInvalid {
					r = robj.Key
				} else if robj.Hash.Type != ssb.RefInvalid {
					r = robj.Hash
				}
			}

			log.Println("Peer asking for ", r.String())
			if !bs.Has(r) {
				conn.Send(&codec.Packet{
					Req:    -req,
					Type:   codec.String,
					Stream: true,
					EndErr: true,
					Body:   []byte("Blob does not exist"),
				})
				return
			}
			rc := bs.Get(r)
			defer rc.Close()
			buf := make([]byte, 1024)
			log.Println("Sending", r.String())
			for {
				n, err := rc.Read(buf[:cap(buf)])
				buf = buf[:n]
				if n == 0 {
					if err == nil {
						continue
					}
					if err == io.EOF {
						break
					}
					log.Fatal(err)
				}

				conn.Send(&codec.Packet{
					Req:    -req,
					Type:   codec.Buffer,
					Stream: true,
					Body:   buf,
				})
				if err != nil && err != io.EOF {
					log.Fatal(err)
				}
			}
			log.Println("Sent", r.String())
			conn.Send(&codec.Packet{
				Req:    -req,
				Stream: true,
				Type:   codec.JSON,
				Body:   []byte("true"),
				EndErr: true,
			})
		}
		handlers["blobs.changes"] = func(conn *muxrpc.Conn, req int32, rm json.RawMessage) {
		}

		peerWants := map[*muxrpc.Conn]chan struct {
			id  ssb.Ref
			val int
		}{}
		var peerWantsLock sync.Mutex

		handlers["blobs.createWants"] = func(conn *muxrpc.Conn, req int32, rm json.RawMessage) {
			peerWantsLock.Lock()
			c := peerWants[conn]
			peerWantsLock.Unlock()

			for ref := range c {
				msg := map[string]int{}
				msg[ref.id.String()] = ref.val
				buf, _ := ssb.Encode(msg)
				log.Println("Telling peer I have ", string(buf))
				conn.Send(&codec.Packet{
					Req:    -req,
					Type:   codec.JSON,
					Stream: true,
					Body:   buf,
				})
			}
		}
		onConnects, ok := ds.ExtraData("muxrpcOnConnect").(map[string]func(conn *muxrpc.Conn))
		if !ok {
			onConnects = map[string]func(conn *muxrpc.Conn){}
			ds.SetExtraData("muxrpcOnConnect", onConnects)
		}
		onConnects["blob"] = func(conn *muxrpc.Conn) {
			bs.wantLock.Lock()
			defer bs.wantLock.Unlock()
			peerWantsLock.Lock()
			c := make(chan struct {
				id  ssb.Ref
				val int
			}, 10)
			peerWants[conn] = c
			peerWantsLock.Unlock()

			go func() {
				conn.Source("blobs.createWants", func(p *codec.Packet) {
					var want map[ssb.Ref]int
					json.Unmarshal(p.Body, &want)
					for id, hopsize := range want {
						if bs.Has(id) && hopsize < 0 {
							log.Println("I do have ", id, "asked at", hopsize)
							c <- struct {
								id  ssb.Ref
								val int
							}{id, int(bs.Size(id))}
						}
					}
				})
				peerWantsLock.Lock()
				delete(peerWants, conn)
				peerWantsLock.Unlock()
			}()

			for r, w := range bs.want {
				go func(r ssb.Ref, w *want) {
					has := false
					err := conn.Call("blobs.has", &has, r)
					if err != nil {
						return
					}
					if has {
						w.resp <- conn
					}
				}(r, w)
			}
		}

	})
}
