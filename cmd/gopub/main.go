package main

import (
    "log"

	"github.com/andyleap/go-ssb"
	"github.com/andyleap/go-ssb/gossip"
	"github.com/cryptix/secretstream/secrethandshake"
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
