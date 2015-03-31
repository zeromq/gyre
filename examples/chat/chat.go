package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
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
	input         = make(chan string)
	name          = flag.String("name", "Gyreman", "Your name or nick name in the chat session")
	gossipBind    = flag.String("gossip-bind", "", "At least one node in the cluster must bind to a well-known gossip endpoint, so other nodes can connect to it")
	gossipConnect endpoints
)

func chat() {
	node, err := gyre.New()
	if err != nil {
		log.Fatalln(err)
	}
	defer node.Stop()

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
	node.Join("CHAT")

	for {
		select {
		case e := <-node.Events():
			switch e.Type() {
			case gyre.EventShout:
				fmt.Printf("%c[2K\r%s%s> ", 27, string(e.Msg()), *name)
			}
		case msg := <-input:
			node.Shout("CHAT", []byte(msg))
		}
	}
}

func main() {

	flag.Var(&gossipConnect, "gossip-connect", "A node may connect to multiple other nodes, for redundancy")
	flag.Parse()

	go chat()

	fmt.Printf("%s> ", *name)

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		input <- fmt.Sprintf("%s: %s\n", *name, scanner.Text())
		fmt.Printf("%s> ", *name)
	}
	if err := scanner.Err(); err != nil {
		log.Fatalln("reading standard input:", err)
	}
}
