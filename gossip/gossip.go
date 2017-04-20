package gossip

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/boltdb/bolt"

	"github.com/andyleap/go-ssb"
	"github.com/andyleap/go-ssb/graph"
	"github.com/cryptix/secretstream"
	"github.com/go-kit/kit/log"
	"scuttlebot.io/go/muxrpc"
)

type Pub struct {
	Link ssb.Ref `json:"link"`
	Host string  `json:"host"`
	Port int     `json:"port"`
}

type PubAnnounce struct {
	ssb.MessageBody
	Pub Pub `json:"pub"`
}

func AddPub(ds *ssb.DataStore, pb Pub) {
	ds.DB().Update(func(tx *bolt.Tx) error {
		PubBucket, err := tx.CreateBucketIfNotExists([]byte("pubs"))
		if err != nil {
			return err
		}
		buf, _ := json.Marshal(pb)
		PubBucket.Put([]byte(pb.Link), buf)
		return nil
	})
}

func init() {
	ssb.AddMessageHooks["gossip"] = func(m *ssb.SignedMessage, tx *bolt.Tx) error {
		_, mb := m.DecodeMessage()
		if mbp, ok := mb.(*PubAnnounce); ok {
			PubBucket, err := tx.CreateBucketIfNotExists([]byte("pubs"))
			if err != nil {
				return err
			}
			buf, _ := json.Marshal(mbp.Pub)
			PubBucket.Put([]byte(mbp.Pub.Link), buf)
			return nil
		}
		return nil
	}
	ssb.MessageTypes["pub"] = func() interface{} {
		return &PubAnnounce{}
	}
}

type ExtraData struct {
	lock  sync.Mutex
	conns map[ssb.Ref]*muxrpc.Client
}

func Replicate(ds *ssb.DataStore) {
	ed := &ExtraData{conns: map[ssb.Ref]*muxrpc.Client{}}
	ds.SetExtraData("gossip", ed)
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
			fmt.Println("New connection from:", conn.RemoteAddr())
			muxConn := muxrpc.NewClient(log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout)), conn)
			go HandleConn(ds, muxConn)
		}
	}()
	go func() {
		ssc, _ := secretstream.NewClient(*ds.PrimaryKey, sbotAppKey)
		pubList := GetPubs(ds)
		t := time.NewTicker(10 * time.Second)
		for range t.C {
			func() {
				ed.lock.Lock()
				defer ed.lock.Unlock()

				fmt.Println("tick")
				if len(pubList) == 0 {
					pubList = GetPubs(ds)
				}
				if len(pubList) == 0 {
					return
				}
				pub := pubList[0]
				pubList = pubList[1:]

				if _, ok := ed.conns[pub.Link]; ok {
					return
				}

				var pubKey [32]byte
				rawpubKey := pub.Link.Raw()
				copy(pubKey[:], rawpubKey)

				d, err := ssc.NewDialer(pubKey)
				if err != nil {
					return
				}
				fmt.Println("Connecting to ", pub)
				conn, err := d("tcp", fmt.Sprintf("%s:%d", pub.Host, pub.Port))
				if err != nil {
					fmt.Println(err)
					return
				}
				muxConn := muxrpc.NewClient(log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout)), conn)

				ed.conns[pub.Link] = muxConn
				go func() {
					HandleConn(ds, muxConn)
					ed.lock.Lock()
					defer ed.lock.Unlock()
					delete(ed.conns, pub.Link)
				}()
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

func HandleConn(ds *ssb.DataStore, muxConn *muxrpc.Client) {
	/*muxConn.HandleSource("createHistoryStream", func(rm json.RawMessage) chan interface{} {
		params := struct {
			Id   ssb.Ref `json:"id"`
			Seq  int     `json:"seq"`
			Live bool    `json:"live"`
		}{
			"",
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
	})*/

	go func() {
		i := 0
		for feed := range graph.GetFollows(ds, ds.PrimaryRef, 2) {
			go func(feed ssb.Ref, i int) {
				time.Sleep(time.Duration(i) * 50 * time.Millisecond)
				reply := make(chan *ssb.SignedMessage)
				f := ds.GetFeed(feed)
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
			}(feed, i)
			i++
		}
	}()
	muxConn.Handle()
}
