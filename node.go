// Gyre is Golang port of Zyre, an open-source framework for proximity-based
// peer-to-peer applications.
// Gyre does local area discovery and clustering. A Gyre node broadcasts
// UDP beacons, and connects to peers that it finds. This class wraps a
// Gyre node with a message-based API.
package gyre

import (
	"github.com/armen/gyre/beacon"
	"github.com/armen/gyre/msg"
	zmq "github.com/pebbe/zmq4"

	"bytes"
	crand "crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"
)

const (
	// IANA-assigned port for ZRE discovery protocol
	zreDiscoveryPort = 5670

	beaconVersion = 0x1

	// Port range 0xc000~0xffff is defined by IANA for dynamic or private ports
	// We use this when choosing a port for dynamic binding
	dynPortFrom uint16 = 0xc000
	dynPortTo   uint16 = 0xffff
)

const (
	EventEnter   = "ENTER"
	EventExit    = "EXIT"
	EventWhisper = "WHISPER"
	EventShout   = "SHOUT"
	EventJoin    = "JOIN"
	EventLeave   = "LEAVE"
	EventSet     = "SET"
)

type sig struct {
	Protocol [3]byte
	Version  byte
	Uuid     []byte
	Port     uint16
}

type Event struct {
	Type    string
	Peer    string
	Group   string
	Key     string // Only used for EventSet
	Content []byte
}

type Node struct {
	quit chan struct{}  // quit is used to signal handler() about quiting
	wg   sync.WaitGroup // wait group is used to wait until handler() is done

	events     chan *Event
	commands   chan *Event
	inboxChan  chan [][]byte
	beacon     *beacon.Beacon
	uuid       []byte            // Our UUID
	identity   string            // Our UUID as hex string
	inbox      *zmq.Socket       // Our inbox socket (ROUTER)
	host       string            // Our host IP address
	port       uint16            // Our inbox port number
	status     byte              // Our own change counter
	peers      map[string]*peer  // Hash of known peers, fast lookup
	peerGroups map[string]*group // Groups that our peers are in
	ownGroups  map[string]*group // Groups that we are in
	headers    map[string]string // Our header values
}

// NewNode creates a new node.
func NewNode() (node *Node, err error) {
	node = &Node{
		quit: make(chan struct{}),
		// Following three channels are used in handler() method which is heart of the Gyre
		// if something blocks while sending to one of these channels, it'll cause pause in
		// the system which isn't desired.
		events:     make(chan *Event, 10000),   // Do not block on sending events
		commands:   make(chan *Event, 10000),   // Do not block on sending commands
		inboxChan:  make(chan [][]byte, 10000), // Do not block while reading from inbox channel
		peers:      make(map[string]*peer),
		peerGroups: make(map[string]*group),
		ownGroups:  make(map[string]*group),
		headers:    make(map[string]string),
	}
	node.wg.Add(1) // We're going to wait until handler() is done

	node.inbox, err = zmq.NewSocket(zmq.ROUTER)
	if err != nil {
		return nil, err
	}

	for i := dynPortFrom; i <= dynPortTo; i++ {
		rand.Seed(time.Now().UTC().UnixNano())
		port := uint16(rand.Intn(int(dynPortTo-dynPortFrom))) + dynPortFrom
		err = node.inbox.Bind(fmt.Sprintf("tcp://*:%d", port))
		if err == nil {
			node.port = port
			break
		}
	}

	// Generate random uuid
	node.uuid = make([]byte, 16)
	io.ReadFull(crand.Reader, node.uuid)
	node.identity = fmt.Sprintf("%X", node.uuid)

	s := &sig{}
	s.Protocol[0] = 'Z'
	s.Protocol[1] = 'R'
	s.Protocol[2] = 'E'
	s.Version = beaconVersion
	s.Uuid = node.uuid
	s.Port = node.port

	buffer := new(bytes.Buffer)
	binary.Write(buffer, binary.BigEndian, s.Protocol)
	binary.Write(buffer, binary.BigEndian, s.Version)
	binary.Write(buffer, binary.BigEndian, s.Uuid)
	binary.Write(buffer, binary.BigEndian, s.Port)

	// Create a beacon
	node.beacon, err = beacon.New(zreDiscoveryPort)
	if err != nil {
		return nil, err
	}
	node.host = node.beacon.Addr()
	node.beacon.NoEcho()
	node.beacon.Subscribe([]byte("ZRE"))
	node.beacon.Publish(buffer.Bytes())

	go node.inboxHandler()
	go node.handler()

	return
}

// Sends message to single peer. peer ID is first frame in message.
func (n *Node) Whisper(identity string, content []byte) *Node {
	n.commands <- &Event{
		Type:    EventWhisper,
		Peer:    identity,
		Content: content,
	}
	return n
}

// Sends message to a group of peers.
func (n *Node) Shout(group string, content []byte) *Node {
	n.commands <- &Event{
		Type:    EventShout,
		Group:   group,
		Content: content,
	}
	return n
}

// Joins a group.
func (n *Node) Join(group string) *Node {
	n.commands <- &Event{
		Type:  EventJoin,
		Group: group,
	}
	return n
}

func (n *Node) Leave(group string) *Node {
	n.commands <- &Event{
		Type:  EventLeave,
		Group: group,
	}
	return n
}

func (n *Node) Set(key, value string) *Node {
	n.commands <- &Event{
		Type:    EventSet,
		Key:     key,
		Content: []byte(value),
	}
	return n
}

// Returns specified header value
func (n *Node) Header(key string) (header string, ok bool) {
	header, ok = n.headers[key]
	return
}

func (n *Node) whisper(identity string, content []byte) {

	// Get peer to send message to
	peer, ok := n.peers[identity]

	// Send frame on out to peer's mailbox, drop message
	// if peer doesn't exist (may have been destroyed)
	if ok {
		m := msg.NewWhisper()
		m.Content = content
		peer.send(m)
	}
}

func (n *Node) shout(group string, content []byte) {
	// Get group to send message to
	if g, ok := n.peerGroups[group]; ok {
		m := msg.NewShout()
		m.Group = group
		m.Content = content
		g.send(m)
	}
}

func (n *Node) join(group string) {
	if _, ok := n.ownGroups[group]; !ok {

		// Only send if we're not already in group
		n.ownGroups[group] = newGroup(group)
		m := msg.NewJoin()
		m.Group = group

		// Update status before sending command
		n.status++
		m.Status = n.status

		for _, peer := range n.peers {
			cloned := msg.Clone(m)
			peer.send(cloned)
		}
	}
}

func (n *Node) leave(group string) {
	if _, ok := n.ownGroups[group]; ok {
		// Only send if we are actually in group
		m := msg.NewLeave()
		m.Group = group

		// Update status before sending command
		n.status++
		m.Status = n.status

		for _, peer := range n.peers {
			cloned := msg.Clone(m)
			peer.send(cloned)
		}
		delete(n.ownGroups, group)
	}
}

func (n *Node) set(key string, value []byte) {
	n.headers[key] = string(value)
}

// Chan returns events channel
func (n *Node) Chan() chan *Event {
	return n.events
}

func (n *Node) inboxHandler() {
	poller := zmq.NewPoller()
	poller.Add(n.inbox, zmq.POLLIN)

	for {
		sockets, _ := poller.Poll(-1)
		for _, socket := range sockets {
			switch s := socket.Socket; s {
			case n.inbox:
				msg, err := s.RecvMessageBytes(0)
				if err != nil {

				}
				n.inboxChan <- msg
			}
		}
	}
}

func (n *Node) handler() {
	defer func() {
		n.wg.Done()
	}()

	ping := time.After(reapInterval)
	stype, _ := n.inbox.GetType()

	for {
		select {
		case <-n.quit:
			// Quiting, do not send beacons anymore
			n.beacon.Silence().Close()
			return

		case e := <-n.commands:
			// Received a command from the caller/API
			switch e.Type {
			case EventWhisper:
				n.whisper(e.Peer, e.Content)
			case EventShout:
				n.shout(e.Group, e.Content)
			case EventJoin:
				n.join(e.Group)
			case EventLeave:
				n.leave(e.Group)
			case EventSet:
				n.set(e.Key, e.Content)
			}

		case frames := <-n.inboxChan:
			transit, err := msg.Unmarshal(stype, frames...)
			if err != nil {
				continue
			}
			n.recvFromPeer(transit)

		case s := <-n.beacon.Signals():
			n.recvFromBeacon(s)

		case <-ping:
			ping = time.After(reapInterval)
			for _, peer := range n.peers {
				n.pingPeer(peer)
			}
		}
	}
}

// recvFromPeer handles messages coming from other peers
func (n *Node) recvFromPeer(transit msg.Transit) {
	// Router socket tells us the identity of this peer
	// Identity must be [1] followed by 16-byte UUID, ignore the [1]
	identity := string(transit.Address()[1:])

	peer := n.peers[identity]

	switch m := transit.(type) {
	case *msg.Hello:
		// On HELLO we may create the peer if it's unknown
		// On other commands the peer must already exist
		peer = n.requirePeer(identity, m.Ipaddress, m.Mailbox)
		peer.ready = true
	}

	// Ignore command if peer isn't ready
	if peer == nil || !peer.ready {
		log.Printf("W: [%s] peer %s wasn't ready, ignoring a %s message", n.identity, identity, transit)
		return
	}

	if !peer.checkMessage(transit) {
		log.Printf("W: [%s] lost messages from %s", n.identity, identity)
		return
	}

	// Now process each command
	switch m := transit.(type) {
	case *msg.Hello:
		// Store peer headers for future reference
		for key, val := range m.Headers {
			peer.headers[key] = val
		}

		// Join peer to listed groups
		for _, group := range m.Groups {
			n.joinPeerGroup(peer, group)
		}

		// Hello command holds latest status of peer
		peer.status = m.Status

	case *msg.Whisper:
		// Pass up to caller API as WHISPER event
		n.events <- &Event{
			Type:    EventWhisper,
			Peer:    identity,
			Content: m.Content,
		}

	case *msg.Shout:
		// Pass up to caller as SHOUT event
		n.events <- &Event{
			Type:    EventShout,
			Peer:    identity,
			Group:   m.Group,
			Content: m.Content,
		}

	case *msg.Ping:
		ping := msg.NewPingOk()
		peer.send(ping)

	case *msg.Join:
		n.joinPeerGroup(peer, m.Group)
		if m.Status != peer.status {
			log.Printf("W: [%s] message status isn't equal to peer status, %d != %d", n.identity, m.Status, peer.status)
		}

	case *msg.Leave:
		n.leavePeerGroup(peer, m.Group)
		if m.Status != peer.status {
			log.Printf("W: [%s] message status isn't equal to peer status, %d != %d", n.identity, m.Status, peer.status)
		}
	}

	// Activity from peer resets peer timers
	peer.refresh()
}

// recvFromBeacon handles a new signal received from beacon
func (n *Node) recvFromBeacon(b *beacon.Signal) {
	// Get IP address and beacon of peer

	parts := strings.SplitN(b.Addr, ":", 2)
	ipaddress := parts[0]

	s := &sig{}
	buffer := bytes.NewBuffer(b.Transmit)
	binary.Read(buffer, binary.BigEndian, &s.Protocol)
	binary.Read(buffer, binary.BigEndian, &s.Version)

	uuid := make([]byte, 16)
	binary.Read(buffer, binary.BigEndian, uuid)
	s.Uuid = append(s.Uuid, uuid...)

	binary.Read(buffer, binary.BigEndian, &s.Port)

	// Ignore anything that isn't a valid beacon
	if s.Version == beaconVersion {
		// Check that the peer, identified by its UUID, exists
		identity := fmt.Sprintf("%X", s.Uuid)
		peer := n.requirePeer(identity, ipaddress, s.Port)
		peer.refresh()
	}
}

// requirePeer finds or creates peer via its UUID string
func (n *Node) requirePeer(identity, address string, port uint16) (peer *peer) {
	peer, ok := n.peers[identity]
	if !ok {
		// Purge any previous peer on same endpoint
		endpoint := fmt.Sprintf("%s:%d", address, port)
		for _, p := range n.peers {
			if p.endpoint == endpoint {
				p.disconnect()
			}
		}

		peer = newPeer(identity)
		peer.connect(n.identity, endpoint)

		// Handshake discovery by sending HELLO as first message
		m := msg.NewHello()
		m.Ipaddress = n.host
		m.Mailbox = n.port
		m.Status = n.status
		for key := range n.ownGroups {
			m.Groups = append(m.Groups, key)
		}
		for key, header := range n.headers {
			m.Headers[key] = header
		}
		peer.send(m)
		n.peers[identity] = peer

		// Now tell the caller about the peer
		n.events <- &Event{
			Type: EventEnter,
			Peer: identity,
		}
	}

	return peer
}

// requirePeerGroup finds or creates group via its name
func (n *Node) requirePeerGroup(name string) (group *group) {
	group, ok := n.peerGroups[name]
	if !ok {
		group = newGroup(name)
		n.peerGroups[name] = group
	}

	return
}

// joinPeerGroup joins the pear to a group
func (n *Node) joinPeerGroup(peer *peer, name string) {
	group := n.requirePeerGroup(name)
	group.join(peer)

	// Now tell the caller about the peer joined group
	n.events <- &Event{
		Type:  EventJoin,
		Peer:  peer.identity,
		Group: name,
	}
}

// leavePeerGroup leaves the pear to a group
func (n *Node) leavePeerGroup(peer *peer, name string) {
	group := n.requirePeerGroup(name)
	group.leave(peer)

	// Now tell the caller about the peer left group
	n.events <- &Event{
		Type:  EventLeave,
		Peer:  peer.identity,
		Group: name,
	}
}

// We do this once a second:
// - if peer has gone quiet, send TCP ping
// - if peer has disappeared, expire it
func (n *Node) pingPeer(peer *peer) {
	if time.Now().Unix() >= peer.expiredAt.Unix() {
		// If peer has really vanished, expire it
		n.events <- &Event{
			Type: EventExit,
			Peer: peer.identity,
		}
		for _, group := range n.peerGroups {
			group.leave(peer)
		}
		// It's really important to disconnect from the peer before
		// deleting it, unless we'd end up difficulties to reconnect
		// to the same endpoint
		peer.disconnect()
		delete(n.peers, peer.identity)
	} else if time.Now().Unix() >= peer.evasiveAt.Unix() {
		//  If peer is being evasive, force a TCP ping.
		//  TODO: do this only once for a peer in this state;
		//  it would be nicer to use a proper state machine
		//  for peer management.
		m := msg.NewPing()
		peer.send(m)
	}
}

// Disconnect leaves all the groups and the closes all the connections to the peers
func (n *Node) Disconnect() {
	close(n.quit)
	n.wg.Wait()

	// Close sockets on a signal
	for group := range n.ownGroups {
		// Note that n.leave is used not n.Leave because we're already in select
		// and Leave sends communicate to events channel which obviously blocks
		n.leave(group)
	}
	// Disconnect from all peers
	for peerId, peer := range n.peers {
		// It's really important to disconnect from the peer before
		// deleting it, unless we'd end up difficulties to reconnect
		// to the same endpoint
		peer.disconnect()
		delete(n.peers, peerId)
	}
	// Now it's safe to close the socket
	n.inbox.Unbind(fmt.Sprintf("tcp://*:%d", n.port))
	n.inbox.Close()
}

// Identity returns string representation of UUID
func (n *Node) Identity() string {
	return n.identity
}

// Uuid returns UUID of the node
func (n *Node) Uuid() []byte {
	return n.uuid
}
