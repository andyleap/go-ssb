package ssb

import (
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"testing"

	"github.com/cryptix/go-muxrpc"
	"github.com/cryptix/secretstream"
	"github.com/cryptix/secretstream/secrethandshake"
	"github.com/go-kit/kit/log"
)

func TestSlurp(t *testing.T) {
	sbotAppKey, _ := base64.StdEncoding.DecodeString("1KHLiKZvAvjbY1ziZEHMXawbCEIM6qwjCDm3VYRan/s=")
	u, _ := user.Current()
	localKey, err := secrethandshake.LoadSSBKeyPair(filepath.Join(u.HomeDir, ".ssb", "secret"))
	if err != nil {
		t.Fatal(err)
	}
	var conn net.Conn
	c, err := secretstream.NewClient(*localKey, sbotAppKey)
	if err != nil {
		t.Fatal(err)
	}
	var remotPubKey = localKey.Public
	d, err := c.NewDialer(remotPubKey)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("Connecting to local sbot")
	conn, err = d("tcp", "127.0.0.1:8008")
	fmt.Println("Connected to local sbot")
	if err != nil {
		t.Fatal(err)
	}
	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))

	client := muxrpc.NewClient(logger, conn)

	output := make(chan *SignedMessage)

	fmt.Println("@" + base64.StdEncoding.EncodeToString(localKey.Public[:]) + ".ed25519")

	feedstore, _ := OpenFeedStore("feeds.db")
	f := feedstore.GetFeed(Ref("@" + base64.StdEncoding.EncodeToString(localKey.Public[:]) + ".ed25519"))

	seq := 0

	latest := f.Latest()

	if latest != nil {
		seq = latest.Sequence + 1
	}

	go func() {
		client.Source("createHistoryStream", output, map[string]interface{}{"id": "@" + base64.StdEncoding.EncodeToString(localKey.Public[:]) + ".ed25519", "keys": false, "seq": seq}, 0, false)
		close(output)
	}()

	for m := range output {
		err = f.AddMessage(m)
		if err != nil {
			t.Fatal(err)
		}
	}
}
