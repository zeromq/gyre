package main

import (
	"github.com/armen/gyre"

	"flag"
	"log"
	"os"
	"os/signal"
)

var (
	input   = make(chan string)
	group   = flag.String("group", "GLOBAL", "The group we are going to join")
	verbose = flag.Bool("verbose", true, "Set verbose flag")
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
	}
	err = node.Start()
	if err != nil {
		log.Fatalln(err)
	}
	node.Join(*group)

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

			case gyre.EventLeave:
				log.Printf("[%s] peer %q left\n", node.Name(), e.Name())

			case gyre.EventWhisper:
				log.Printf("[%s] received ping (WHISPER) from %q\n", node.Name(), e.Name())

			case gyre.EventShout:
				log.Printf("[%s] (%s) received a ping (SHOUT) from %q\n", node.Name(), e.Group(), e.Name())
			}
		case <-c:
			return
		}
	}
}

func main() {
	flag.Parse()

	ping()
}
