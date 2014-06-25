package gyre

import (
	"github.com/zeromq/gyre/msg"
)

type group struct {
	name  string           // Group name
	peers map[string]*peer // Peers in group
}

// newGroup creates a new group
func newGroup(name string) *group {
	return &group{
		name:  name,
		peers: make(map[string]*peer),
	}
}

// Join adds peer to group. Ignore duplicate joins
func (g *group) join(peer *peer) {
	g.peers[peer.identity] = peer
	peer.status++
}

// Leave removes peer from group
func (g *group) leave(peer *peer) {
	delete(g.peers, peer.identity)
	peer.status++
}

// Send sends message to all peers in group
func (g *group) send(m msg.Transit) {
	for _, peer := range g.peers {
		cloned := msg.Clone(m)
		peer.send(cloned)
	}
}
