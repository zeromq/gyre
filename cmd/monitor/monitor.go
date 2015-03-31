package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strings"

	"github.com/zeromq/gyre"
)

type endpoints []string

func (e *endpoints) String() string {
	return fmt.Sprint(*e)
}

// Set is the method to set the flag value, part of the flag.Value interface.
func (e *endpoints) Set(value string) error {
	if len(*e) > 0 {
		return errors.New("endpoints flag already set")
	}
	for _, rawurl := range strings.Split(value, ",") {
		u, err := url.Parse(rawurl)
		if err != nil {
			return err
		}
		*e = append(*e, u.String())
	}
	return nil
}

var (
	group         = flag.String("group", "*", "The group we are going to join. By default joins every group in the network. For multiple groups separate groups with comma.")
	verbose       = flag.Bool("verbose", true, "Set verbose flag")
	gossipBind    = flag.String("gossip-bind", "", "At least one node in the cluster must bind to a well-known gossip endpoint, so other nodes can connect to it")
	gossipConnect endpoints
)

func ping() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, os.Kill)

	node, err := gyre.New()
	if err != nil {
		log.Fatalln(err)
	}
	defer node.Stop()

	if *verbose {
		node.SetVerbose()
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	if gossipConnect != nil {
		for _, u := range gossipConnect {
			node.GossipConnect(u)
		}
	} else if *gossipBind != "" {
		node.GossipBind(*gossipBind)
	}

	err = node.Start()
	if err != nil {
		log.Fatalln(err)
	}

	if *group != "*" {
		for _, g := range strings.Split(*group, ",") {
			node.Join(strings.TrimSpace(g))
		}
	}

	for {
		select {
		case e := <-node.Events():
			switch e.Type() {
			case gyre.EventEnter:
				log.Printf("[%s] peer %q entered\n", node.Name(), e.Name())

			case gyre.EventExit:
				log.Printf("[%s] peer %q exited\n", node.Name(), e.Name())

			case gyre.EventJoin:
				log.Printf("[%s] peer %q joined to %s\n", node.Name(), e.Name(), e.Group())
				if *group == "*" {
					node.Join(e.Group())
				}

			case gyre.EventLeave:
				log.Printf("[%s] peer %q left\n", node.Name(), e.Name())

			case gyre.EventWhisper:
				log.Printf("[%s] received a WHISPER from %q\n", node.Name(), e.Name())

			case gyre.EventShout:
				log.Printf("[%s] received a SHOUT targeted to %s group from %q\n", node.Name(), e.Group(), e.Name())
			}
		case <-c:
			return
		}
	}
}

func main() {
	flag.Var(&gossipConnect, "gossip-connect", "A node may connect to multiple other nodes, for redundancy")
	flag.Parse()

	ping()
}
