package main

import (
	"crypto/rand"
	"encoding/base64"
	"io/ioutil"
	"log"
	"net"
	//    "os"

	"github.com/andyleap/go-ssb"
	_ "github.com/andyleap/go-ssb/channels"
	"github.com/andyleap/go-ssb/cmd/sbot/rpc"
	_ "github.com/andyleap/go-ssb/git"
	"github.com/andyleap/go-ssb/gossip"
	"github.com/andyleap/go-ssb/graph"
	"github.com/andyleap/go-ssb/social"

	"cryptoscope.co/go/secretstream/secrethandshake"

	r "net/rpc"
)

var datastore *ssb.DataStore

func main() {

	keypair, err := secrethandshake.LoadSSBKeyPair("secret.json")
	if err != nil {
		keypair, err = secrethandshake.GenEdKeyPair(rand.Reader)
		if err != nil {
			log.Fatal(err)
		}

		ref, _ := ssb.NewRef(ssb.RefFeed, keypair.Public[:], ssb.RefAlgoEd25519)
		sbotKey := struct {
			Curve   string `json:"curve"`
			ID      string `json:"id"`
			Private string `json:"private"`
			Public  string `json:"public"`
		}{
			Curve:   "ed25519",
			ID:      ref.String(),
			Private: base64.StdEncoding.EncodeToString(keypair.Secret[:]) + ".ed25519",
			Public:  base64.StdEncoding.EncodeToString(keypair.Public[:]) + ".ed25519",
		}
		buf, _ := ssb.Encode(sbotKey)
		err := ioutil.WriteFile("secret.json", buf, 0600)
		if err != nil {
			log.Fatal(err)
		}
	}

	datastore, err = ssb.OpenDataStore("feeds.db", keypair)
	if err != nil {
		log.Fatal(err)
	}
	defer datastore.Close()

	gossip.Replicate(datastore)

	RegisterWebui()

	//datastore.Rebuild("channels")

	err = r.Register(&Gossip{datastore})
	if err != nil {
		log.Fatal(err)
	}
	err = r.Register(&Feed{datastore})
	if err != nil {
		log.Fatal(err)
	}

	l, _ := net.Listen("tcp", "localhost:9822")
	if err != nil {
		log.Fatal(err)
	}

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
