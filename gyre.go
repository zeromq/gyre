// Package gyre is Golang port of Zyre, an open-source framework for proximity-based
// peer-to-peer applications.
// Gyre does local area discovery and clustering. A Gyre node broadcasts
// UDP beacons, and connects to peers that it finds. This class wraps a
// Gyre node with a message-based API.
package gyre

import (
	"errors"
	"fmt"
	"log"
	"time"
)

const (
	timeout = 5 * time.Second
)

// Gyre structure
type Gyre struct {
	cmds    chan interface{}
	replies chan interface{}
	events  chan *Event       // Receives incoming cluster events/traffic
	uuid    string            // Copy of our uuid
	name    string            // Copy of our name
	addr    string            // Copy of our address
	headers map[string]string // Headres cache
}

type cmd struct {
	cmd     string
	key     string
	payload interface{}
}

type reply struct {
	cmd     string
	payload interface{}
	err     error
}

const (
	cmdUUID          = "UUID"
	cmdName          = "NAME"
	cmdSetName       = "SET NAME"
	cmdSetHeader     = "SET HEADER"
	cmdSetVerbose    = "SET VERBOSE"
	cmdSetPort       = "SET PORT"
	cmdSetInterval   = "SET INTERVAL"
	cmdSetIface      = "SET INTERFACE"
	cmdSetEndpoint   = "SET ENDPOINT"
	cmdGossipBind    = "GOSSIP BIND"
	cmdGossipPort    = "GOSSIP PORT"
	cmdGossipConnect = "GOSSIP CONNECT"
	cmdStart         = "START"
	cmdStop          = "STOP"
	cmdWhisper       = "WHISPER"
	cmdShout         = "SHOUT"
	cmdJoin          = "JOIN"
	cmdLeave         = "LEAVE"
	cmdDump          = "DUMP"
	cmdTerm          = "$TERM"

	// Deprecated
	cmdAddr    = "ADDR"
	cmdHeader  = "HEADER"
	cmdHeaders = "HEADERS"
)

// New creates a new Gyre node. Note that until you start the
// node it is silent and invisible to other nodes on the network.
func New() (g *Gyre, err error) {
	g, _, err = newGyre()
	return
}

// New creates a new Gyre node. This methods returns node object as well which is
// used for testing purposes
func newGyre() (*Gyre, *node, error) {
	g := &Gyre{
		// The following channels are used in nodeActor() method which is heart of the Gyre
		// if something blocks while sending to one of these channels, it'll cause pause in
		// the system which isn't desired.
		events:  make(chan *Event, 10000), // Do not block on sending events
		cmds:    make(chan interface{}),   // Shouldn't be a buffered channel because the main select acts as a lock
		replies: make(chan interface{}),
		headers: make(map[string]string),
	}

	n, err := newNode(g.events, g.cmds, g.replies)
	if err != nil {
		return nil, nil, err
	}

	go n.actor()

	return g, n, nil
}

// UUID returns our node UUID, after successful initialization
func (g *Gyre) UUID() (uuid string) {
	if g.uuid != "" {
		return g.uuid
	}

	select {
	case g.cmds <- &cmd{cmd: cmdUUID}:
	case <-time.After(timeout):
		log.Fatal("Node is not responding")
	}

	select {
	case r := <-g.replies:
		out := r.(*reply)
		if uuid, ok := out.payload.(string); ok {
			g.uuid = uuid
		}
	case <-time.After(timeout):
		log.Fatal("Node is not responding")
	}

	return g.uuid
}

// Name returns our node name, after successful initialization.
// By default is taken from the UUID and shortened.
func (g *Gyre) Name() (name string) {
	if g.name != "" {
		return g.name
	}

	select {
	case g.cmds <- &cmd{cmd: cmdName}:
	case <-time.After(timeout):
		log.Fatal("Node is not responding")
	}

	select {
	case r := <-g.replies:
		out := r.(*reply)
		if name, ok := out.payload.(string); ok {
			g.name = name
		}
	case <-time.After(timeout):
		log.Fatal("Node is not responding")
	}

	return g.name
}

// Addr returns our address. Note that it will return empty string
// if called before Start() method.
func (g *Gyre) Addr() string {
	if g.addr != "" {
		return g.addr
	}

	select {
	case g.cmds <- &cmd{cmd: cmdAddr}:
	case <-time.After(timeout):
		log.Fatal("Node is not responding")
	}

	select {
	case r := <-g.replies:
		out := r.(*reply)
		if addr, ok := out.payload.(string); ok {
			g.addr = addr
		}
	case <-time.After(timeout):
		log.Fatal("Node is not responding")
	}

	return g.addr
}

// Header returns specified header
func (g *Gyre) Header(key string) (header string, ok bool) {

	if header, ok = g.headers[key]; ok {
		return
	}

	select {
	case g.cmds <- &cmd{cmd: cmdHeader, key: key}:
	case <-time.After(timeout):
		log.Fatal("Node is not responding")
	}

	select {
	case r := <-g.replies:
		out := r.(*reply)
		if out.err != nil {
			return
		}

		header, ok = out.payload.(string)
		g.headers[key] = header
	case <-time.After(timeout):
		log.Fatal("Node is not responding")
	}

	return header, ok
}

// Headers returns headers
func (g *Gyre) Headers() map[string]string {

	select {
	case g.cmds <- &cmd{cmd: cmdHeaders}:
	case <-time.After(timeout):
		log.Fatal("Node is not responding")
	}

	select {
	case r := <-g.replies:
		out := r.(*reply)
		if headers, ok := out.payload.(map[string]string); ok {
			return headers
		}
	case <-time.After(timeout):
		log.Fatal("Node is not responding")
	}

	return nil
}

// SetName sets node name; this is provided to other nodes during discovery.
// If you do not set this, the UUID is used as a basis.
func (g *Gyre) SetName(name string) *Gyre {

	select {
	case g.cmds <- &cmd{cmd: cmdSetName, payload: name}:
	case <-time.After(timeout):
		log.Fatal("Node is not responding")
	}

	return g
}

// SetHeader sets node header; these are provided to other nodes during discovery
// and come in each ENTER message.
func (g *Gyre) SetHeader(name string, format string, args ...interface{}) *Gyre {
	payload := fmt.Sprintf(format, args...)

	select {
	case g.cmds <- &cmd{cmd: cmdSetHeader, key: name, payload: payload}:
	case <-time.After(timeout):
		log.Fatal("Node is not responding")
	}

	return g
}

// SetVerbose sets verbose mode; this tells the node to log all traffic as well
// as all major events.
func (g *Gyre) SetVerbose() *Gyre {

	select {
	case g.cmds <- &cmd{cmd: cmdSetVerbose, payload: true}:
	case <-time.After(timeout):
		log.Fatal("Node is not responding")
	}

	return g
}

// SetPort sets ZRE discovery port; defaults to 5670, this call overrides that
// so you can create independent clusters on the same network, for e.g
// development vs production.
func (g *Gyre) SetPort(port int) *Gyre {

	select {
	case g.cmds <- &cmd{cmd: cmdSetPort, payload: port}:
	case <-time.After(timeout):
		log.Fatal("Node is not responding")
	}

	return g
}

// SetInterval sets ZRE discovery interval. Default is instant beacon
// exploration followed by pinging every 1,000 msecs.
func (g *Gyre) SetInterval(interval time.Duration) *Gyre {

	select {
	case g.cmds <- &cmd{cmd: cmdSetInterval, payload: interval}:
	case <-time.After(timeout):
		log.Fatal("Node is not responding")
	}

	return g
}

// SetInterface sets network interface to use for beacons and interconnects. If you
// do not set this, Gyre will choose an interface for you. On boxes
// with multiple interfaces you really should specify which one you
// want to use, or strange things can happen.
func (g *Gyre) SetInterface(iface string) *Gyre {
	select {
	case g.cmds <- &cmd{cmd: cmdSetIface, payload: iface}:
	case <-time.After(timeout):
		log.Fatal("Node is not responding")
	}

	return g
}

// SetEndpoint sets the endpoint. By default, Gyre binds to an ephemeral TCP
// port and broadcasts the local host name using UDP beaconing. When you call
// this method, Gyre will use gossip discovery instead of UDP beaconing. You
// MUST set-up the gossip service separately using GossipBind() and
// GossipConnect(). Note that the endpoint MUST be valid for both bind and
// connect operations. You can use inproc://, ipc://, or tcp:// transports
// (for tcp://, use an IP address that is meaningful to remote as well as
// local nodes).
func (g *Gyre) SetEndpoint(endpoint string) *Gyre {
	select {
	case g.cmds <- &cmd{cmd: cmdSetEndpoint, payload: endpoint}:
	case <-time.After(timeout):
		log.Fatal("Node is not responding")
	}

	select {
	case r := <-g.replies:
		out := r.(*reply)
		if out.err != nil {
			log.Fatal(out.err)
		}
	case <-time.After(timeout):
		log.Fatal("Node is not responding")
	}

	return g
}

// GossipBind Sets up gossip discovery of other nodes. At least one node in
// the cluster must bind to a well-known gossip endpoint, so other nodes
// can connect to it. Note that gossip endpoints are completely distinct
// from Gyre node endpoints, and should not overlap (they can use the same
// transport).
func (g *Gyre) GossipBind(endpoint string) *Gyre {
	select {
	case g.cmds <- &cmd{cmd: cmdGossipBind, payload: endpoint}:
	case <-time.After(timeout):
		log.Fatal("Node is not responding")
	}

	select {
	case r := <-g.replies:
		out := r.(*reply)
		if out.err != nil {
			log.Fatal(out.err)
		}
	case <-time.After(timeout):
		log.Fatal("Node is not responding")
	}

	return g
}

// GossipPort returns the port number that gossip engine is bound to
func (g *Gyre) GossipPort() string {
	select {
	case g.cmds <- &cmd{cmd: cmdGossipPort}:
	case <-time.After(timeout):
		log.Fatal("Node is not responding")
	}

	select {
	case r := <-g.replies:
		out := r.(*reply)
		if out.err != nil {
			log.Fatal(out.err)
		}
		return out.payload.(string)
	case <-time.After(timeout):
		log.Fatal("Node is not responding")
	}

	return ""
}

// GossipConnect Sets up gossip discovery of other nodes. A node may connect
// to multiple other nodes, for redundancy paths.
func (g *Gyre) GossipConnect(endpoint string) *Gyre {
	select {
	case g.cmds <- &cmd{cmd: cmdGossipConnect, payload: endpoint}:
	case <-time.After(timeout):
		log.Fatal("Node is not responding")
	}

	select {
	case r := <-g.replies:
		out := r.(*reply)
		if out.err != nil {
			log.Fatal(out.err)
		}
	case <-time.After(timeout):
		log.Fatal("Node is not responding")
	}

	return g
}

// Start starts a node, after setting header values. When you start a node it
// begins discovery and connection. Returns nil if OK, and error if
// it wasn't possible to start the node.
func (g *Gyre) Start() (err error) {
	select {
	case g.cmds <- &cmd{cmd: cmdStart}:
	case <-time.After(timeout):
		log.Fatal("Node is not responding")
	}

	select {
	case r := <-g.replies:
		out := r.(*reply)
		if out.err != nil {
			return out.err
		}
	case <-time.After(timeout):
		return errors.New("Node is not responding")
	}

	return nil
}

// Stop stops a node; this signals to other peers that this node will go away.
// This is polite; however you can also just destroy the node without
// stopping it.
func (g *Gyre) Stop() {

	select {
	case g.cmds <- &cmd{cmd: cmdStop}:
	case <-time.After(timeout):
		log.Fatal("Node is not responding")
	}

	select {
	case <-g.replies:
	case <-time.After(20 * time.Millisecond):
		// Let it timeout, don't wait forever
	}
}

// Join a named group; after joining a group you can send messages to
// the group and all Gyre nodes in that group will receive them.
func (g *Gyre) Join(group string) *Gyre {
	select {
	case g.cmds <- &cmd{cmd: cmdJoin, key: group}:
	case <-time.After(timeout):
		log.Fatal("Node is not responding")
	}
	return g
}

// Leave a group.
func (g *Gyre) Leave(group string) *Gyre {
	select {
	case g.cmds <- &cmd{cmd: cmdLeave, key: group}:
	case <-time.After(timeout):
		log.Fatal("Node is not responding")
	}
	return g
}

// Events returns a channel of events. The events may be a control
// event (ENTER, EXIT, JOIN, LEAVE) or data (WHISPER, SHOUT).
func (g *Gyre) Events() chan *Event {
	return g.events
}

// Whisper sends a message to single peer, specified as a UUID string.
func (g *Gyre) Whisper(peer string, payload []byte) *Gyre {
	select {
	case g.cmds <- &cmd{cmd: cmdWhisper, key: peer, payload: payload}:
	case <-time.After(timeout):
		log.Fatal("Node is not responding")
	}
	return g
}

// Shout sends a message to a named group.
func (g *Gyre) Shout(group string, payload []byte) *Gyre {
	select {
	case g.cmds <- &cmd{cmd: cmdShout, key: group, payload: payload}:
	case <-time.After(timeout):
		log.Fatal("Node is not responding")
	}
	return g
}

// Whispers sends a formatted string to a single peer specified as UUID string.
func (g *Gyre) Whispers(peer string, format string, args ...interface{}) *Gyre {
	payload := fmt.Sprintf(format, args...)
	select {
	case g.cmds <- &cmd{cmd: cmdWhisper, key: peer, payload: []byte(payload)}:
	case <-time.After(timeout):
		log.Fatal("Node is not responding")
	}
	return g
}

// Shouts sends a message to a named group.
func (g *Gyre) Shouts(group string, format string, args ...interface{}) *Gyre {
	payload := fmt.Sprintf(format, args...)
	select {
	case g.cmds <- &cmd{cmd: cmdShout, key: group, payload: []byte(payload)}:
	case <-time.After(timeout):
		log.Fatal("Node is not responding")
	}
	return g
}

// Dump prints Gyre node information.
func (g *Gyre) Dump() *Gyre {
	select {
	case g.cmds <- &cmd{cmd: cmdDump}:
	case <-time.After(timeout):
		log.Fatal("Node is not responding")
	}
	return g
}
