package main

import (
	"log"
	"net"

	"github.com/andyleap/go-ssb"
	_ "github.com/andyleap/go-ssb/channels"
	"github.com/andyleap/go-ssb/cmd/sbot/rpc"
	_ "github.com/andyleap/go-ssb/git"
	"github.com/andyleap/go-ssb/gossip"
	"github.com/andyleap/go-ssb/graph"
	"github.com/andyleap/go-ssb/social"

	r "net/rpc"
)

var datastore *ssb.DataStore

func main() {
	datastore, _ = ssb.OpenDataStore("feeds.db", "secret.json")

	gossip.Replicate(datastore)

	RegisterWebui()

	//datastore.Rebuild("channels")

	r.Register(&Gossip{datastore})
	r.Register(&Feed{datastore})

	l, _ := net.Listen("tcp", ":9822")

	r.Accept(l)

	select {}
}

type Gossip struct {
	ds *ssb.DataStore
}

func (g *Gossip) AddPub(req rpc.AddPubReq, res *rpc.AddPubRes) error {
	gossip.AddPub(g.ds, gossip.Pub{
		Host: req.Host,
		Port: req.Port,
		Link: ssb.ParseRef(req.PubKey),
	})
	return nil
}

type Feed struct {
	ds *ssb.DataStore
}

func (f *Feed) Post(req rpc.PostReq, res *rpc.PostRes) error {
	if req.Feed == "" {
		req.Feed = f.ds.PrimaryRef.String()
	}
	feed := f.ds.GetFeed(ssb.ParseRef(req.Feed))

	post := &social.Post{}

	post.Text = req.Text
	post.Channel = req.Channel
	post.Branch = ssb.ParseRef(req.Branch)
	post.Root = ssb.ParseRef(req.Root)
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
		req.Feed = f.ds.PrimaryRef.String()
	}
	feed := f.ds.GetFeed(ssb.ParseRef(req.Feed))

	follow := &graph.Contact{}

	following := true
	follow.Following = &following
	follow.Contact = ssb.ParseRef(req.Contact)
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
		req.Feed = f.ds.PrimaryRef.String()
	}
	feed := f.ds.GetFeed(ssb.ParseRef(req.Feed))

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
