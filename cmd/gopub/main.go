package main

import (
	"github.com/andyleap/go-ssb"
	"github.com/andyleap/go-ssb/gossip"
)

func main() {
	datastore, _ := ssb.OpenDataStore("feeds.db", "secret.json")
	gossip.Replicate(datastore)

	select {}
}
