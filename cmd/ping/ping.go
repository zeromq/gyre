package main

import (
	"github.com/armen/go-zre/pkg/zre"

	"log"
)

func main() {
	node, err := zre.NewNode()
	if err != nil {
		log.Println(err)
	}
	node.Join("GLOBAL")
	log.Printf("I: [%s] started\n", node.Identity)

	for {
		select {
		case e := <-node.Events:
			switch e.Type {
			case zre.EventEnter:
				log.Printf("I: [%s] peer entered\n", e.Peer)
				node.Whisper(e.Peer, []byte("Hello"))

			case zre.EventExit:
				log.Printf("I: [%s] peer exited\n", e.Peer)

			case zre.EventWhisper:
				log.Printf("I: [%s] received ping (WHISPER)\n", e.Peer)
				node.Shout("GLOBAL", []byte("Hello"))

			case zre.EventShout:
				log.Printf("I: [%s] (%s) received ping (SHOUT)\n", e.Peer, e.Group)
			}
		}
	}
}
