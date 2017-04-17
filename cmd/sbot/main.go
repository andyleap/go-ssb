package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/andyleap/go-ssb"
	"github.com/andyleap/go-ssb/cmd/sbot/rpc"

	r "net/rpc"

	"github.com/andyleap/boltinspect"
)

var datastore *ssb.DataStore

func main() {
	datastore, _ = ssb.OpenDataStore("feeds.db", "secret.json")

	bi := boltinspect.New(datastore.DB())

	http.HandleFunc("/bolt", bi.InspectEndpoint)

	http.HandleFunc("/", Index)

	go http.ListenAndServe(":9823", nil)

	r.Register(&Gossip{datastore})
	r.Register(&Feed{datastore})

	l, _ := net.Listen("tcp", ":9822")

	r.Accept(l)

	for {
	}
}

type Gossip struct {
	ds *ssb.DataStore
}

func (g *Gossip) AddPub(req rpc.AddPubReq, res *rpc.AddPubRes) error {
	g.ds.AddPub(ssb.PubData{
		Host: req.Host,
		Port: req.Port,
		Link: ssb.Ref(req.PubKey),
	})
	return nil
}

type Feed struct {
	ds *ssb.DataStore
}

func (f *Feed) Post(req rpc.PostReq, res *rpc.PostRes) error {
	if req.Feed == "" {
		req.Feed = string(f.ds.PrimaryRef)
	}
	feed := f.ds.GetFeed(ssb.Ref(req.Feed))

	post := &ssb.Post{}

	post.Text = req.Text
	post.Channel = req.Channel
	post.Branch = ssb.Ref(req.Branch)
	post.Root = ssb.Ref(req.Root)
	post.Type = "post"

	content, _ := json.Marshal(post)

	m := &ssb.Message{
		Author:    feed.ID,
		Timestamp: float64(time.Now().UnixNano() / int64(time.Millisecond)),
		Hash:      "sha256",
		Content:   content,
		Sequence:  1,
	}

	if l := feed.Latest(); l != nil {
		key := l.Key()
		m.Previous = &key
		m.Sequence = l.Sequence + 1
		for m.Timestamp <= l.Timestamp {
			m.Timestamp += 0.01
		}
	}

	signer := f.ds.Keys[feed.ID]
	if signer == nil {
		return nil
	}
	sm := m.Sign(signer)

	err := feed.AddMessage(sm)
	if err != nil {
		log.Println(err)
	} else {
		log.Println("Message ", sm, " posted to feed ", feed.ID)
	}

	return nil
}

func (f *Feed) Follow(req rpc.FollowReq, res *rpc.FollowRes) error {
	if req.Feed == "" {
		req.Feed = string(f.ds.PrimaryRef)
	}
	feed := f.ds.GetFeed(ssb.Ref(req.Feed))

	follow := &ssb.Contact{}

	following := true
	follow.Following = &following
	follow.Contact = ssb.Ref(req.Contact)
	follow.Type = "contact"

	content, _ := json.Marshal(follow)

	m := &ssb.Message{
		Author:    feed.ID,
		Timestamp: float64(time.Now().UnixNano() / int64(time.Millisecond)),
		Hash:      "sha256",
		Content:   content,
		Sequence:  1,
	}

	if l := feed.Latest(); l != nil {
		key := l.Key()
		m.Previous = &key
		m.Sequence = l.Sequence + 1
		for m.Timestamp <= l.Timestamp {
			m.Timestamp += 0.01
		}
	}

	signer := f.ds.Keys[feed.ID]
	if signer == nil {
		return nil
	}
	sm := m.Sign(signer)

	err := feed.AddMessage(sm)
	if err != nil {
		log.Println(err)
	} else {
		log.Println("Message ", sm, " posted to feed ", feed.ID)
	}

	return nil
}
