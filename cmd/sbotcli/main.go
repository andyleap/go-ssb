package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/urfave/cli"

	r "net/rpc"

	"github.com/andyleap/go-ssb/cmd/sbot/rpc"
)

func main() {
	app := cli.NewApp()

	client, _ := r.Dial("tcp", "127.0.0.1:9822")

	app.Commands = []cli.Command{
		{
			Name:    "gossip.add",
			Aliases: []string{"g.a"},
			Usage:   "add a peer to the gossip list",
			Action: func(c *cli.Context) error {
				if c.NArg() != 3 {
					return fmt.Errorf("Expected 3 arguments")
				}
				port, err := strconv.Atoi(c.Args().Get(1))
				if err != nil {
					return err
				}
				req := rpc.AddPubReq{
					Host:   c.Args().Get(0),
					Port:   port,
					PubKey: c.Args().Get(2),
				}
				res := rpc.AddPubRes{}
				return client.Call("Gossip.AddPub", req, &res)
			},
		},
		{
			Name:    "feed.post",
			Aliases: []string{"f.p"},
			Usage:   "publish a new post to a feed",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "feed",
					Usage: "Feed to publish to",
				},
				cli.StringFlag{
					Name:  "root",
					Usage: "Root post to respond to",
				},
				cli.StringFlag{
					Name:  "branch",
					Usage: "Branch post to respond to",
				},
				cli.StringFlag{
					Name:  "channel",
					Usage: "Channel to publish to",
				},
			},
			Action: func(c *cli.Context) error {
				if c.NArg() != 1 {
					return fmt.Errorf("Expected 1 argument")
				}
				req := rpc.PostReq{
					Feed:    c.String("feed"),
					Text:    c.Args().Get(0),
					Root:    c.String("root"),
					Branch:  c.String("branch"),
					Channel: c.String("channel"),
				}
				res := rpc.PostRes{}
				return client.Call("Feed.Post", req, &res)
			},
		},
		{
			Name:    "feed.follow",
			Aliases: []string{"f.f"},
			Usage:   "follow a feed",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "feed",
					Usage: "Feed to publish to",
				},
			},
			Action: func(c *cli.Context) error {
				if c.NArg() != 1 {
					return fmt.Errorf("Expected 1 argument")
				}
				req := rpc.FollowReq{
					Feed:    c.String("feed"),
					Contact: c.Args().Get(0),
				}
				res := rpc.FollowRes{}
				return client.Call("Feed.Follow", req, &res)
			},
		},
		{
			Name:    "feed.name",
			Aliases: []string{"f.n"},
			Usage:   "set a name for the feed",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "feed",
					Usage: "Feed to publish to",
				},
			},
			Action: func(c *cli.Context) error {
				if c.NArg() != 1 {
					return fmt.Errorf("Expected 1 argument")
				}
				req := rpc.AboutReq{
					Feed: c.String("feed"),
					Name: c.Args().Get(0),
				}
				res := rpc.AboutRes{}
				return client.Call("Feed.About", req, &res)
			},
		},
	}
	app.Run(os.Args)
}
