package main

import (
	"github.com/armen/gyre"

	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
)

var (
	input   = make(chan string)
	group   = flag.String("group", "*", "The group we are going to join. By default joins every group in the network. For multiple groups separate groups with comma.")
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
	flag.Parse()

	ping()
}
