package main

import (
	"github.com/armen/gyre"

	"flag"
	"fmt"
	"log"
)

var (
	input = make(chan string)
	name  = flag.String("name", "Gyreman", "My name")
)

func chat() {
	node, err := gyre.NewNode()
	if err != nil {
		log.Fatalln(err)
	}
	defer node.Disconnect()

	node.Join("CHAT")

	for {
		select {
		case e := <-node.Chan():
			switch e.Type {
			case gyre.EventShout:
				fmt.Printf("\r%s\n%s> ", string(e.Content), *name)
			}
		case msg := <-input:
			node.Shout("CHAT", []byte(msg))
		}
	}
}

func main() {

	flag.Parse()

	go chat()

	var (
		err error
		n   int
	)

	for ln := ""; err == nil; n, err = fmt.Scanln(&ln) {
		fmt.Printf("%s> ", *name)
		if n > 0 {
			input <- fmt.Sprintf("%s: %s", *name, ln)
		}
	}
}
