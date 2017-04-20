package main

import (
	"log"
	"net"
	"net/http"

	"github.com/andyleap/go-ssb"
	_ "github.com/andyleap/go-ssb/channels"
	"github.com/andyleap/go-ssb/cmd/sbot/rpc"
	_ "github.com/andyleap/go-ssb/git"
	"github.com/andyleap/go-ssb/gossip"
	"github.com/andyleap/go-ssb/graph"
	"github.com/andyleap/go-ssb/social"

	r "net/rpc"

	"github.com/andyleap/boltinspect"
)

var datastore *ssb.DataStore

func main() {
	datastore, _ = ssb.OpenDataStore("feeds.db", "secret.json")
	gossip.Replicate(datastore)

	bi := boltinspect.New(datastore.DB())

	http.HandleFunc("/bolt", bi.InspectEndpoint)

	http.HandleFunc("/", Index)
	http.HandleFunc("/channel", Channel)

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
	gossip.AddPub(g.ds, gossip.Pub{
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

	post := &social.Post{}

	post.Text = req.Text
	post.Channel = req.Channel
	post.Branch = ssb.Ref(req.Branch)
	post.Root = ssb.Ref(req.Root)
	post.Type = "post"

	err := feed.PublishMessage(post)
	if err != nil {
		log.Println(err)
	} else {
		log.Println("Message ", post, " posted to feed ", feed.ID)
	}

	return nil
}

func (f *Feed) Follow(req rpc.FollowReq, res *rpc.FollowRes) error {
	if req.Feed == "" {
		req.Feed = string(f.ds.PrimaryRef)
	}
	feed := f.ds.GetFeed(ssb.Ref(req.Feed))

	follow := &graph.Contact{}

	following := true
	follow.Following = &following
	follow.Contact = ssb.Ref(req.Contact)
	follow.Type = "contact"

	err := feed.PublishMessage(follow)
	if err != nil {
		log.Println(err)
	} else {
		log.Println("Message ", follow, " posted to feed ", feed.ID)
	}

	return nil
}

func (f *Feed) About(req rpc.AboutReq, res *rpc.AboutRes) error {
	if req.Feed == "" {
		req.Feed = string(f.ds.PrimaryRef)
	}
	feed := f.ds.GetFeed(ssb.Ref(req.Feed))

	about := &social.About{}

	about.Name = req.Name
	about.About = feed.ID
	about.Type = "about"

	err := feed.PublishMessage(about)
	if err != nil {
		log.Println(err)
	} else {
		log.Println("Message ", about, " posted to feed ", feed.ID)
	}

	return nil
}
