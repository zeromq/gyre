// Gyre is Golang port of Zyre, an open-source framework for proximity-based
// peer-to-peer applications.
// Gyre does local area discovery and clustering. A Gyre node broadcasts
// UDP beacons, and connects to peers that it finds. This class wraps a
// Gyre node with a message-based API.
package gyre

import (
	"fmt"
	"time"
)

// Gyre structure
type Gyre struct {
	cmds   chan *cmd
	events chan *Event // Receives incoming cluster events/traffic
	uuid   string      // Copy of our uuid
	name   string      // Copy of our name
}

type cmd struct {
	cmd     string
	key     string
	payload interface{}
	err     error // Only on the return
}

const (
	cmdName        = "NAME"
	cmdUuid        = "UUID"
	cmdHeader      = "HEADER"
	cmdHeaders     = "HEADERS"
	cmdSetName     = "SET NAME"
	cmdSetHeader   = "SET HEADER"
	cmdSetVerbose  = "SET VERBOSE"
	cmdSetPort     = "SET PORT"
	cmdSetInterval = "SET INTERVAL"
	cmdStart       = "START"
	cmdStop        = "STOP"
	cmdJoin        = "JOIN"
	cmdLeave       = "LEAVE"
	cmdWhisper     = "WHISPER"
	cmdShout       = "SHOUT"
	cmdBind        = "BIND"
	cmdConnect     = "CONNECT"
	cmdDump        = "DUMP"
	cmdTerm        = "$TERM"
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
		events: make(chan *Event, 10000), // Do not block on sending events
		cmds:   make(chan *cmd),          // Shouldn't be a buffered channel because the main select acts as a lock
	}

	n, err := newNode(g.events, g.cmds)
	if err != nil {
		return nil, nil, err
	}

	go n.actor()

	return g, n, nil
}

// Return our node UUID, after successful initialization
func (g *Gyre) Uuid() (uuid string) {
	if g.uuid != "" {
		return g.uuid
	}

	g.cmds <- &cmd{cmd: cmdUuid}
	out := <-g.cmds
	g.uuid = out.payload.(string)

	return g.uuid
}

// Return our node name, after successful initialization.
// By default is taken from the UUID and shortened.
func (g *Gyre) Name() (name string) {
	if g.name != "" {
		return g.name
	}

	g.cmds <- &cmd{cmd: cmdName}
	out := <-g.cmds
	g.name = out.payload.(string)

	return g.name
}

// Return specified header
func (g *Gyre) Header(key string) (header string, ok bool) {
	g.cmds <- &cmd{cmd: cmdHeader, key: key}
	out := <-g.cmds

	if out.err != nil {
		return
	}

	header = out.payload.(string)

	return header, true
}

// Return headers
func (g *Gyre) Headers() map[string]string {
	g.cmds <- &cmd{cmd: cmdHeaders}
	out := <-g.cmds

	return out.payload.(map[string]string)
}

// Set node name; this is provided to other nodes during discovery.
// If you do not set this, the UUID is used as a basis.
func (g *Gyre) SetName(name string) *Gyre {
	g.cmds <- &cmd{
		cmd:     cmdSetName,
		payload: name,
	}

	return g
}

// Set node header; these are provided to other nodes during discovery
// and come in each ENTER message.
func (g *Gyre) SetHeader(name string, format string, args ...interface{}) *Gyre {
	payload := fmt.Sprintf(format, args...)
	g.cmds <- &cmd{
		cmd:     cmdSetHeader,
		key:     name,
		payload: payload,
	}

	return g
}

// Set verbose mode; this tells the node to log all traffic as well
// as all major events.
func (g *Gyre) SetVerbose() *Gyre {
	g.cmds <- &cmd{
		cmd:     cmdSetVerbose,
		payload: true,
	}

	return g
}

// Set ZRE discovery port; defaults to 5670, this call overrides that
// so you can create independent clusters on the same network, for e.g
// development vs production.
func (g *Gyre) SetPort(port uint16) *Gyre {
	g.cmds <- &cmd{
		cmd:     cmdSetPort,
		payload: port,
	}

	return g
}

// Set ZRE discovery interval. Default is instant beacon
// exploration followed by pinging every 1,000 msecs.
func (g *Gyre) SetInterval(interval time.Duration) *Gyre {
	g.cmds <- &cmd{
		cmd:     cmdSetInterval,
		payload: interval,
	}

	return g
}

// Set network interface to use for beacons and interconnects. If you
// do not set this, Gyre will choose an interface for you. On boxes
// with multiple interfaces you really should specify which one you
// want to use, or strange things can happen.
func (g *Gyre) SetInterface(iface string) *Gyre {
	// TODO(armen): Implement SetInterface
	return g
}

// Start node, after setting header values. When you start a node it
// begins discovery and connection. Returns nil if OK, and error if
// it wasn't possible to start the node.
func (g *Gyre) Start() (err error) {
	g.cmds <- &cmd{
		cmd: cmdStart,
	}
	out := <-g.cmds

	if out.err != nil {
		return out.err
	}

	return nil
}

// Stop node; this signals to other peers that this node will go away.
// This is polite; however you can also just destroy the node without
// stopping it.
func (g *Gyre) Stop() {
	g.cmds <- &cmd{
		cmd: cmdStop,
	}
	<-g.cmds
}

// Join a named group; after joining a group you can send messages to
// the group and all Gyre nodes in that group will receive them.
func (g *Gyre) Join(group string) *Gyre {
	g.cmds <- &cmd{
		cmd: cmdJoin,
		key: group,
	}
	return g
}

// Leave a group.
func (g *Gyre) Leave(group string) *Gyre {
	g.cmds <- &cmd{
		cmd: cmdLeave,
		key: group,
	}
	return g
}

// Returns a channel of events. The events may be a control
// event (ENTER, EXIT, JOIN, LEAVE) or data (WHISPER, SHOUT).
func (g *Gyre) Events() chan *Event {
	return g.events
}

// Send message to single peer, specified as a UUID string.
func (g *Gyre) Whisper(peer string, payload []byte) *Gyre {
	g.cmds <- &cmd{
		cmd:     cmdWhisper,
		key:     peer,
		payload: payload,
	}
	return g
}

// Send message to a named group.
func (g *Gyre) Shout(group string, payload []byte) *Gyre {
	g.cmds <- &cmd{
		cmd:     cmdShout,
		key:     group,
		payload: payload,
	}
	return g
}

// Send formatted string to a single peer specified as UUID string.
func (g *Gyre) Whispers(peer string, format string, args ...interface{}) *Gyre {
	payload := fmt.Sprintf(format, args...)
	g.cmds <- &cmd{
		cmd:     cmdWhisper,
		key:     peer,
		payload: []byte(payload),
	}
	return g
}

// Send message to a named group.
func (g *Gyre) Shouts(group string, format string, args ...interface{}) *Gyre {
	payload := fmt.Sprintf(format, args...)
	g.cmds <- &cmd{
		cmd:     cmdShout,
		key:     group,
		payload: []byte(payload),
	}
	return g
}

// Prints Gyre node information.
func (g *Gyre) Dump() *Gyre {
	g.cmds <- &cmd{cmd: cmdDump}
	return g
}
