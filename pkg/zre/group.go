package zre

import (
	"github.com/armen/go-zre/pkg/msg"
)

type Group struct {
	Name  string           // Group name
	Peers map[string]*Peer // Peers in group
}

// NewGroup creates a new group
func NewGroup(name string) *Group {
	return &Group{
		Name:  name,
		Peers: make(map[string]*Peer),
	}
}

// Join adds peer to group. Ignore duplicate joins
func (g *Group) Join(peer *Peer) {
	g.Peers[peer.Identity] = peer
	peer.Status++
}

// Leave removes peer from group
func (g *Group) Leave(peer *Peer) {
	// It's really important to disconnect from the peer on leave
	peer.Disconnect()
	delete(g.Peers, peer.Identity)
	peer.Status++
}

// Send sends message to all peers in group
func (g *Group) Send(m msg.Transit) {
	for _, peer := range g.Peers {
		cloned := msg.Clone(m)
		peer.Send(cloned)
	}
}
