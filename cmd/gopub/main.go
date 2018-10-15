package main

import (
	"log"

	"go.cryptoscope.co/secretstream/secrethandshake"
	"github.com/andyleap/go-ssb"
	"github.com/andyleap/go-ssb/gossip"
)

func main() {

	keypair, err := secrethandshake.LoadSSBKeyPair("secret.json")
	if err != nil {
		log.Println(err)
	}

	datastore, _ := ssb.OpenDataStore("feeds.db", keypair)

	gossip.Replicate(datastore)

	select {}
}
