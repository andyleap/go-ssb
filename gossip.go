package ssb

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/boltdb/bolt"

	"github.com/cryptix/go-muxrpc"
	"github.com/cryptix/secretstream"
	"github.com/go-kit/kit/log"
)

func (fs *DataStore) AddPub(pb PubData) {
	fs.db.Update(func(tx *bolt.Tx) error {
		PubBucket, err := tx.CreateBucketIfNotExists([]byte("pubs"))
		if err != nil {
			return err
		}
		buf, _ := json.Marshal(pb)
		PubBucket.Put([]byte(pb.Link), buf)
		return nil
	})
}

func (fs *DataStore) HandlePubs() {
	c := fs.Topic.Register(nil, true)

	go func() {
		for m := range c {
			_, mb := m.DecodeMessage()
			if mbp, ok := mb.(*Pub); ok {
				fs.AddPub(mbp.Pub)
			}
		}
	}()

	go func() {
		sbotAppKey, _ := base64.StdEncoding.DecodeString("1KHLiKZvAvjbY1ziZEHMXawbCEIM6qwjCDm3VYRan/s=")
		ssc, _ := secretstream.NewClient(*fs.PrimaryKey, sbotAppKey)
		pubList := fs.GetPubs()
		t := time.NewTicker(10 * time.Second)
		for range t.C {
			fmt.Println("tick")
			if len(pubList) == 0 {
				pubList = fs.GetPubs()
			}
			if len(pubList) == 0 {
				continue
			}
			pub := pubList[0]
			pubList = pubList[1:]

			if _, ok := fs.conns[pub.Link]; ok {
				continue
			}

			var pubKey [32]byte
			rawpubKey := pub.Link.Raw()
			copy(pubKey[:], rawpubKey)

			d, err := ssc.NewDialer(pubKey)
			if err != nil {
				continue
			}
			fmt.Println("Connecting to ", pub)
			conn, err := d("tcp", fmt.Sprintf("%s:%d", pub.Host, pub.Port))
			if err != nil {
				continue
			}

			muxConn := muxrpc.NewClient(log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout)), conn)

			fs.conns[pub.Link] = muxConn

			muxConn.HandleSource("createHistoryStream", func(rm json.RawMessage) chan interface{} {
				params := struct {
					Id   Ref  `json:"id"`
					Seq  int  `json:"seq"`
					Live bool `json:"live"`
				}{
					"",
					0,
					false,
				}
				args := []interface{}{&params}
				json.Unmarshal(rm, &args)
				f := fs.GetFeed(params.Id)
				if f.ID == fs.PrimaryRef {
					fmt.Println(params)
				}
				c := make(chan interface{})
				go func() {
					for m := range f.Log(params.Seq, params.Live) {
						c <- m
					}
					close(c)
				}()
				return c
			})

			go func() {
				muxConn.Handle()
				delete(fs.conns, pub.Link)
			}()

			for feed := range fs.GetFollows(fs.PrimaryRef, 2) {
				go func(feed Ref) {
					reply := make(chan *SignedMessage)
					f := fs.GetFeed(feed)
					seq := 0
					if f.Latest() != nil {
						seq = f.Latest().Sequence + 1
					}
					fmt.Println("Asking for ", f.ID, seq)
					go func() {
						err := muxConn.Source("createHistoryStream", reply, map[string]interface{}{"id": f.ID, "seq": seq, "live": true, "keys": false})
						if err != nil {
							fmt.Println(err)
						}
						close(reply)
					}()
					for m := range reply {
						f.AddMessage(m)
					}
				}(feed)
			}
		}
	}()
}

func (fs *DataStore) GetPubs() (pds []*PubData) {
	fs.db.View(func(tx *bolt.Tx) error {
		PubBucket := tx.Bucket([]byte("pubs"))
		if PubBucket == nil {
			return nil
		}
		PubBucket.ForEach(func(k, v []byte) error {
			var pd *PubData
			json.Unmarshal(v, &pd)
			pds = append(pds, pd)
			return nil
		})
		return nil
	})
	return
}
