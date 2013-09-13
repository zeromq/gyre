package zre

import (
	zmq "github.com/armen/go-zmq"
	"github.com/armen/go-zre/pkg/beacon"
	"github.com/armen/go-zre/pkg/msg"

	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"
)

const (
	// IANA-assigned port for ZRE discovery protocol
	zreDiscoveryPort = 5670

	beaconVersion = 0x1
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
	quit chan struct{}  // quit is used to signal handler about quiting
	wg   sync.WaitGroup // wait group is used to wait until handler() is done

	Events     chan *Event
	Beacon     *beacon.Beacon
	Uuid       []byte            // Our UUID
	Identity   string            // Our UUID as hex string
	context    *zmq.Context      // zmq context
	inbox      *zmq.Socket       // Our inbox socket (ROUTER)
	Host       string            // Our host IP address
	Port       uint16            // Our inbox port number
	Status     byte              // Our own change counter
	Peers      map[string]*Peer  // Hash of known peers, fast lookup
	PeerGroups map[string]*Group // Groups that our peers are in
	OwnGroups  map[string]*Group // Groups that we are in
	Headers    map[string]string // Our header values
}

// NewNode creates a new node
func NewNode() (node *Node, err error) {
	node = &Node{
		quit:       make(chan struct{}),
		Events:     make(chan *Event),
		Peers:      make(map[string]*Peer),
		PeerGroups: make(map[string]*Group),
		OwnGroups:  make(map[string]*Group),
		Headers:    make(map[string]string),
	}
	node.wg.Add(1) // We're going to wait until handler() is done

	context, err := zmq.NewContext()
	if err != nil {
		return nil, err
	}
	node.context = context

	node.inbox, err = context.Socket(zmq.Router)
	if err != nil {
		return nil, err
	}

	err = node.inbox.Bind("tcp://*:*")
	if err != nil {
		return nil, err
	}
	node.Port = node.inbox.Port()

	// Generate random uuid
	node.Uuid = make([]byte, 16)
	io.ReadFull(rand.Reader, node.Uuid)
	node.Identity = fmt.Sprintf("%X", node.Uuid)

	s := &sig{}
	s.Protocol[0] = 'Z'
	s.Protocol[1] = 'R'
	s.Protocol[2] = 'E'
	s.Version = beaconVersion
	s.Uuid = node.Uuid
	s.Port = node.Port

	buffer := new(bytes.Buffer)
	binary.Write(buffer, binary.BigEndian, s.Protocol)
	binary.Write(buffer, binary.BigEndian, s.Version)
	binary.Write(buffer, binary.BigEndian, s.Uuid)
	binary.Write(buffer, binary.BigEndian, s.Port)

	// Create a beacon
	node.Beacon, err = beacon.New(zreDiscoveryPort)
	if err != nil {
		return nil, err
	}
	node.Host = node.Beacon.Hostname
	node.Beacon.NoEcho()
	node.Beacon.Subscribe([]byte("ZRE"))
	node.Beacon.Publish(buffer.Bytes())

	go node.handle()

	return
}

func (n *Node) Whisper(identity string, content []byte) *Node {
	n.Events <- &Event{
		Type:    EventWhisper,
		Peer:    identity,
		Content: content,
	}
	return n
}

func (n *Node) Shout(group string, content []byte) *Node {
	n.Events <- &Event{
		Type:    EventShout,
		Group:   group,
		Content: content,
	}
	return n
}

func (n *Node) Join(group string) *Node {
	n.Events <- &Event{
		Type:  EventJoin,
		Group: group,
	}
	return n
}

func (n *Node) Leave(group string) *Node {
	n.Events <- &Event{
		Type:  EventLeave,
		Group: group,
	}
	return n
}

func (n *Node) Set(key, value string) *Node {
	n.Events <- &Event{
		Type:    EventSet,
		Key:     key,
		Content: []byte(value),
	}
	return n
}

func (n *Node) Get(key string) (header string) {
	return n.Headers[key]
}

func (n *Node) whisper(identity string, content []byte) {

	// Get peer to send message to
	peer, ok := n.Peers[identity]

	// Send frame on out to peer's mailbox, drop message
	// if peer doesn't exist (may have been destroyed)
	if ok {
		m := msg.NewWhisper()
		m.Content = content
		peer.Send(m)
	}
}

func (n *Node) shout(group string, content []byte) {
	// Get group to send message to
	if g, ok := n.PeerGroups[group]; ok {
		m := msg.NewShout()
		m.Group = group
		m.Content = content
		g.Send(m)
	}
}

func (n *Node) join(group string) {
	if _, ok := n.OwnGroups[group]; !ok {

		// Only send if we're not already in group
		n.OwnGroups[group] = NewGroup(group)
		m := msg.NewJoin()
		m.Group = group

		// Update status before sending command
		n.Status++
		m.Status = n.Status

		for _, peer := range n.Peers {
			cloned := msg.Clone(m)
			peer.Send(cloned)
		}
	}
}

func (n *Node) leave(group string) {
	if _, ok := n.PeerGroups[group]; ok {
		// Only send if we are actually in group
		m := msg.NewLeave()
		m.Group = group

		// Update status before sending command
		n.Status++
		m.Status = n.Status

		for _, peer := range n.Peers {
			cloned := msg.Clone(m)
			peer.Send(cloned)
		}
		delete(n.OwnGroups, group)
	}
}

func (n *Node) set(key string, value []byte) {
	n.Headers[key] = string(value)
}

// Chan returns Events channel
func (n *Node) Chan() chan *Event {
	return n.Events
}

func (n *Node) handle() {
	defer func() {
		n.wg.Done()
	}()

	chans := n.inbox.Channels()
	ping := time.After(reapInterval)
	stype := n.inbox.GetType()

	for {
		select {
		case <-n.quit:
			return

		case e := <-n.Events:
			// Received a command/event from the caller/API
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

		case frames := <-chans.In():
			transit, err := msg.RecvRaw(frames, stype)
			if err != nil {
				continue
			}
			n.recvFromPeer(transit)

		case s := <-n.Beacon.Chan():
			n.recvFromBeacon(s)

		case err := <-chans.Errors():
			log.Println(err)

		case <-ping:
			ping = time.After(reapInterval)
			for _, peer := range n.Peers {
				n.pingPeer(peer)
			}
		}
	}
}

// recvFromPeer handles messages coming from other peers
func (n *Node) recvFromPeer(transit msg.Transit) {
	// Router socket tells us the identity of this peer
	identity := string(transit.Address())

	peer := n.Peers[identity]

	switch m := transit.(type) {
	case *msg.Hello:
		// On HELLO we may create the peer if it's unknown
		// On other commands the peer must already exist
		peer = n.requirePeer(identity, m.Ipaddress, m.Mailbox)
		peer.Ready = true
	}

	// Ignore command if peer isn't ready
	if peer == nil || !peer.Ready {
		log.Printf("W: [%s] peer %s wasn't ready, ignoring a %s message", n.Identity, identity, transit)
		return
	}

	if !peer.CheckMessage(transit) {
		log.Printf("W: [%s] lost messages from %s", n.Identity, identity)
		return
	}

	// Now process each command
	switch m := transit.(type) {
	case *msg.Hello:
		// Hello command holds latest status of peer
		peer.Status = m.Status

		// Store peer headers for future reference
		for key, val := range m.Headers {
			peer.Headers[key] = val
		}

		// Join peer to listed groups
		for _, group := range m.Groups {
			n.joinPeerGroup(peer, group)
		}

	case *msg.Whisper:
		// Pass up to caller API as WHISPER event
		n.Events <- &Event{
			Type:    EventWhisper,
			Peer:    identity,
			Content: m.Content,
		}

	case *msg.Shout:
		// Pass up to caller as SHOUT event
		n.Events <- &Event{
			Type:    EventShout,
			Peer:    identity,
			Group:   m.Group,
			Content: m.Content,
		}

	case *msg.Ping:
		ping := msg.NewPingOk()
		peer.Send(ping)

	case *msg.Join:
		n.joinPeerGroup(peer, m.Group)
		if m.Status != peer.Status {
			log.Printf("W: [%s] message status isn't equal to peer status, %d != %d", n.Identity, m.Status, peer.Status)
		}

	case *msg.Leave:
		n.leavePeerGroup(peer, m.Group)
		if m.Status != peer.Status {
			log.Printf("W: [%s] message status isn't equal to peer status, %d != %d", n.Identity, m.Status, peer.Status)
		}
	}

	// Activity from peer resets peer timers
	peer.Refresh()
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
		peer.Refresh()
	}
}

// requirePeer finds or creates peer via its UUID string
func (n *Node) requirePeer(identity, address string, port uint16) (peer *Peer) {
	peer, ok := n.Peers[identity]
	if !ok {
		// Purge any previous peer on same endpoint
		endpoint := fmt.Sprintf("%s:%d", address, port)
		for _, p := range n.Peers {
			if p.Endpoint == endpoint {
				p.Disconnect()
			}
		}

		peer = NewPeer(identity, n.context)
		peer.Connect(n.Identity, endpoint)

		// Handshake discovery by sending HELLO as first message
		m := msg.NewHello()
		m.Ipaddress = n.Host
		m.Mailbox = n.Port
		m.Status = n.Status
		for key := range n.OwnGroups {
			m.Groups = append(m.Groups, key)
		}
		for key, header := range n.Headers {
			m.Headers[key] = header
		}
		peer.Send(m)
		n.Peers[identity] = peer

		// Now tell the caller about the peer
		n.Events <- &Event{
			Type: EventEnter,
			Peer: identity,
		}
	}

	return peer
}

// requirePeerGroup finds or creates group via its name
func (n *Node) requirePeerGroup(name string) (group *Group) {
	group, ok := n.PeerGroups[name]
	if !ok {
		group = NewGroup(name)
		n.PeerGroups[name] = group
	}

	return
}

// joinPeerGroup joins the pear to a group
func (n *Node) joinPeerGroup(peer *Peer, name string) {
	group := n.requirePeerGroup(name)
	group.Join(peer)

	// Now tell the caller about the peer joined group
	n.Events <- &Event{
		Type:  EventJoin,
		Peer:  peer.Identity,
		Group: name,
	}
}

// leavePeerGroup leaves the pear to a group
func (n *Node) leavePeerGroup(peer *Peer, name string) {
	group := n.requirePeerGroup(name)
	group.Leave(peer)

	// Now tell the caller about the peer left group
	n.Events <- &Event{
		Type:  EventLeave,
		Peer:  peer.Identity,
		Group: name,
	}
}

// We do this once a second:
// - if peer has gone quiet, send TCP ping
// - if peer has disappeared, expire it
func (n *Node) pingPeer(peer *Peer) {
	if time.Now().Unix() >= peer.ExpiredAt.Unix() {
		// If peer has really vanished, expire it
		n.Events <- &Event{
			Type: EventExit,
			Peer: peer.Identity,
		}
		for _, group := range n.PeerGroups {
			group.Leave(peer)
		}
		delete(n.Peers, peer.Identity)
	} else if time.Now().Unix() >= peer.EvasiveAt.Unix() {
		//  If peer is being evasive, force a TCP ping.
		//  TODO: do this only once for a peer in this state;
		//  it would be nicer to use a proper state machine
		//  for peer management.
		m := msg.NewPing()
		peer.Send(m)
	}
}

// Disconnect leaves all the groups and the closes all the connections to the peers
func (n *Node) Disconnect() {
	close(n.quit)
	n.wg.Wait()

	// Close sockets on a signal
	for group := range n.OwnGroups {
		// Note that n.leave is used not n.Leave because we're already in select
		// and Leave sends communicate to Events channel which obviously blocks
		n.leave(group)
	}
	// Disconnect from all peers
	for _, p := range n.Peers {
		p.Disconnect()
	}
	// Now it's safe to close the connection
	n.inbox.Close()
}
