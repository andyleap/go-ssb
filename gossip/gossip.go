package gossip

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/boltdb/bolt"

	"github.com/andyleap/go-ssb"
	"github.com/andyleap/go-ssb/graph"
	"github.com/andyleap/go-ssb/muxrpcManager"
	"github.com/andyleap/muxrpc"
	"github.com/andyleap/muxrpc/codec"
	"github.com/cryptix/secretstream"
)

type Pub struct {
	Link ssb.Ref `json:"key"`
	Host string  `json:"host"`
	Port int     `json:"port"`
}

type PubAnnounce struct {
	ssb.MessageBody
	Pub Pub `json:"address"`
}

func AddPub(ds *ssb.DataStore, pb Pub) {
	ds.DB().Update(func(tx *bolt.Tx) error {
		PubBucket, err := tx.CreateBucketIfNotExists([]byte("pubs"))
		if err != nil {
			return err
		}
		buf, _ := json.Marshal(pb)
		PubBucket.Put(pb.Link.DBKey(), buf)
		return nil
	})
}

func init() {
	ssb.RebuildClearHooks["gossip"] = func(tx *bolt.Tx) error {
		tx.DeleteBucket([]byte("pubs"))
		return nil
	}
	ssb.AddMessageHooks["gossip"] = func(m *ssb.SignedMessage, tx *bolt.Tx) error {
		_, mb := m.DecodeMessage()
		if mbp, ok := mb.(*PubAnnounce); ok {
			if mbp.Pub.Link.Type != ssb.RefFeed {
				return nil
			}
			PubBucket, err := tx.CreateBucketIfNotExists([]byte("pubs"))
			if err != nil {
				return err
			}
			buf, _ := json.Marshal(mbp.Pub)
			err = PubBucket.Put(mbp.Pub.Link.DBKey(), buf)
			if err != nil {
				return err
			}
			return nil
		}
		return nil
	}
	ssb.MessageTypes["pub"] = func() interface{} {
		return &PubAnnounce{}
	}

	ssb.RegisterInit(func(ds *ssb.DataStore) {
		handlers, ok := ds.ExtraData("muxrpcHandlers").(map[string]func(conn *muxrpc.Conn, req int32, args json.RawMessage))
		if !ok {
			handlers = map[string]func(conn *muxrpc.Conn, req int32, args json.RawMessage){}
			ds.SetExtraData("muxrpcHandlers", handlers)
		}
		handlers["createHistoryStream"] = func(conn *muxrpc.Conn, req int32, rm json.RawMessage) {
			params := struct {
				Id   ssb.Ref `json:"id"`
				Seq  int     `json:"seq"`
				Live bool    `json:"live"`
			}{
				ssb.Ref{},
				0,
				false,
			}
			args := []interface{}{&params}
			json.Unmarshal(rm, &args)
			f := ds.GetFeed(params.Id)
			go func() {
				err := f.Follow(params.Seq, params.Live, func(m *ssb.SignedMessage) error {
					err := conn.Send(&codec.Packet{
						Req:    -req,
						Type:   codec.JSON,
						Body:   m.Encode(),
						Stream: true,
					})
					return err
				}, conn.Done)
				if err != nil {
					log.Println(err)
					return
				}
				conn.Send(&codec.Packet{
					Req:    -req,
					Type:   codec.JSON,
					Stream: true,
					EndErr: true,
				})
			}()
		}
		onConnects, ok := ds.ExtraData("muxrpcOnConnect").(map[string]func(conn *muxrpc.Conn))
		if !ok {
			onConnects = map[string]func(conn *muxrpc.Conn){}
			ds.SetExtraData("muxrpcOnConnect", onConnects)
		}
		onConnects["replicate"] = func(conn *muxrpc.Conn) {
			i := 0
			for feed := range graph.GetFollows(ds, ds.PrimaryRef, 2) {
				go func(feed ssb.Ref, i int) {
					time.Sleep(time.Duration(i) * 10 * time.Millisecond)
					f := ds.GetFeed(feed)
					if f == nil {
						return
					}
					seq := 0
					if f.Latest() != nil {
						seq = f.Latest().Sequence + 1
					}
					go func() {
						reply := func(p *codec.Packet) {
							if p.Type != codec.JSON {
								return
							}
							var m *ssb.SignedMessage
							err := json.Unmarshal(p.Body, &m)
							if err != nil {
								return
							}
							f.AddMessage(m)
						}
						err := conn.Source("createHistoryStream", reply, map[string]interface{}{"id": f.ID, "seq": seq, "live": true, "keys": false})
						if err != nil {
							log.Println(err)
						}
					}()
				}(feed, i)
				i++
			}
		}
	})

}

func Replicate(ds *ssb.DataStore) {
	sbotAppKey, _ := base64.StdEncoding.DecodeString("1KHLiKZvAvjbY1ziZEHMXawbCEIM6qwjCDm3VYRan/s=")
	go func() {

		sss, _ := secretstream.NewServer(*ds.PrimaryKey, sbotAppKey)
		l, err := sss.Listen("tcp", ":8008")
		if err != nil {
			fmt.Println(err)
			return
		}
		for {
			conn, err := l.Accept()
			if err != nil {
				fmt.Println(err)
				return
			}
			remPubKey := conn.RemoteAddr().(*secretstream.Addr).PubKey()
			remRef, _ := ssb.NewRef(ssb.RefFeed, remPubKey, ssb.RefAlgoEd25519)
			go muxrpcManager.HandleConn(ds, remRef, conn)
		}
	}()
	go func() {
		ed := ds.ExtraData("muxrpcConns").(*muxrpcManager.ExtraData)
		ssc, _ := secretstream.NewClient(*ds.PrimaryKey, sbotAppKey)
		pubList := GetPubs(ds)
		t := time.NewTicker(5 * time.Second)
		for range t.C {
			fmt.Println("tick")
			ed.Lock.Lock()
			connCount := len(ed.Conns)
			ed.Lock.Unlock()
			if connCount >= 3 {
				continue
			}
			if len(pubList) == 0 {
				pubList = GetPubs(ds)
			}
			if len(pubList) == 0 {
				continue
			}
			pub := pubList[0]
			pubList = pubList[1:]

			ed.Lock.Lock()
			_, ok := ed.Conns[pub.Link]
			ed.Lock.Unlock()
			if ok {
				continue
			}

			var pubKey [32]byte
			rawpubKey := pub.Link.Raw()
			copy(pubKey[:], rawpubKey)

			d, err := ssc.NewDialer(pubKey)
			if err != nil {
				continue
			}
			go func() {
				log.Println("Connecting to ", pub)
				conn, err := d("tcp", fmt.Sprintf("%s:%d", pub.Host, pub.Port))
				if err != nil {
					log.Println(err)
					return
				}
				end := time.NewTimer(5 * time.Minute)
				go func() {
					for range end.C {
						conn.Close()
					}
				}()
				muxrpcManager.HandleConn(ds, pub.Link, conn)
				end.Stop()
			}()

		}
	}()
}

func GetPubs(ds *ssb.DataStore) (pds []*Pub) {
	ds.DB().View(func(tx *bolt.Tx) error {
		PubBucket := tx.Bucket([]byte("pubs"))
		if PubBucket == nil {
			return nil
		}
		PubBucket.ForEach(func(k, v []byte) error {
			var pd *Pub
			json.Unmarshal(v, &pd)
			pds = append(pds, pd)
			return nil
		})
		return nil
	})
	return
}

/*func HandleConn(ds *ssb.DataStore, muxConn *muxrpc.Client) {
	muxConn.HandleSource("createHistoryStream", func(rm json.RawMessage) chan interface{} {
		params := struct {
			Id   ssb.Ref `json:"id"`
			Seq  int     `json:"seq"`
			Live bool    `json:"live"`
		}{
			ssb.Ref{},
			0,
			false,
		}
		args := []interface{}{&params}
		json.Unmarshal(rm, &args)
		f := ds.GetFeed(params.Id)
		if f.ID == ds.PrimaryRef {
			fmt.Println(params)
			fmt.Println(string(rm))
		}
		c := make(chan interface{})
		go func() {
			for m := range f.Log(params.Seq, params.Live) {
				fmt.Println("Sending", m.Author, m.Sequence)
				c <- m
			}
			close(c)
		}()
		return c
	})

	go func() {
		i := 0
		for feed := range graph.GetFollows(ds, ds.PrimaryRef, 2) {
			go func(feed ssb.Ref, i int) {
				time.Sleep(time.Duration(i) * 50 * time.Millisecond)
				reply := make(chan *ssb.SignedMessage)
				f := ds.GetFeed(feed)
				if f == nil {
					return
				}
				seq := 0
				if f.Latest() != nil {
					seq = f.Latest().Sequence + 1
				}
				go func() {
					muxConn.Source("createHistoryStream", reply, map[string]interface{}{"id": f.ID, "seq": seq, "live": true, "keys": false})
					close(reply)
				}()
				for m := range reply {
					if m.Sequence == 0 {
						continue
					}
					fmt.Print("*")
					f.AddMessage(m)
				}
			}(feed, i)
			i++
		}
	}()
	muxConn.Handle()
}
*/
