package main

import (
	"github.com/armen/gyre"

	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
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

	fmt.Printf("%s> ", *name)

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		input <- fmt.Sprintf("%s: %s", *name, scanner.Text())
		fmt.Printf("%s> ", *name)
	}
	if err := scanner.Err(); err != nil {
		log.Fatalln("reading standard input:", err)
	}
}
