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
	name  = flag.String("name", "Gyreman", "Your name or nick name in the chat session")
)

func chat() {
	node, err := gyre.New()
	if err != nil {
		log.Fatalln(err)
	}
	defer node.Stop()

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
