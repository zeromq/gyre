package gyre

import (
	"github.com/armen/gyre/beacon"
	"github.com/armen/gyre/msg"
	zmq "github.com/pebbe/zmq4"

	"bytes"
	crand "crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"
)

type node struct {
	terminated chan struct{}       // API shut us down
	wg         sync.WaitGroup      // wait group is used to wait until handler() is done
	events     chan *Event         // We send all Gyre events to the events channel
	cmds       chan *cmd           // We receive commands from and send command replies and signals to the cmds channel
	verbose    bool                // Log all traffic
	beaconPort int                 // Beacon port number
	interval   time.Duration       // Beacon interval
	inboxChan  chan [][]byte       // Channel of inbox messages
	beacon     *beacon.Beacon      // Beacon object
	beacons    chan *beacon.Signal // Beacons
	uuid       []byte              // Our UUID
	inbox      *zmq.Socket         // Our inbox socket (ROUTER)
	name       string              // Our public name
	endpoint   string              // Our public endpoint
	port       uint16              // Our inbox port number
	bound      bool                // Did app bind node explicitly?
	status     byte                // Our own change counter
	peers      map[string]*peer    // Hash of known peers, fast lookup
	peerGroups map[string]*group   // Groups that our peers are in
	ownGroups  map[string]*group   // Groups that we are in
	headers    map[string]string   // Our header values
}

// Beacon frame has this format:
//
// Z R E       3 bytes
// Version     1 byte, %x01
// Uuid        16 bytes
// Port        2 bytes in network order
type aBeacon struct {
	Protocol [3]byte
	Version  byte
	Uuid     []byte
	Port     uint16
}

const (
	beaconVersion = 0x1

	// IANA-assigned port for ZRE discovery protocol
	zreDiscoveryPort = 5670

	// Port range 0xc000~0xffff is defined by IANA for dynamic or private ports
	// We use this when choosing a port for dynamic binding
	dynPortFrom uint16 = 0xc000
	dynPortTo   uint16 = 0xffff
)

// newNode creates a new node.
func newNode(events chan *Event, cmds chan *cmd) (n *node, err error) {
	n = &node{
		events:     events,
		cmds:       cmds,
		inboxChan:  make(chan [][]byte), // Shouldn't be a buffered channel because the main select acts as a lock
		beaconPort: zreDiscoveryPort,
		peers:      make(map[string]*peer),
		peerGroups: make(map[string]*group),
		ownGroups:  make(map[string]*group),
		headers:    make(map[string]string),
	}

	n.inbox, err = zmq.NewSocket(zmq.ROUTER)
	if err != nil {
		return nil, err // Could not create new socket
	}

	// Generate random uuid
	n.uuid = make([]byte, 16)
	io.ReadFull(crand.Reader, n.uuid)

	// Default name for node is first 6 characters of UUID:
	// the shorter string is more readable in logs
	n.name = fmt.Sprintf("%.6s", fmt.Sprintf("%X", n.uuid))

	n.wg.Add(1) // We're going to wait until handler() is done

	return
}

// Bind node to endpoint.
func (n *node) bind(endpoint string) (err error) {

	err = n.inbox.Bind(endpoint)
	if err != nil {
		return
	}

	n.bound = true
	n.beaconPort = 0

	return
}

// Connect node to endpoint explicitly.
func (n *node) connect(endpoint string) (err error) {
	return
}

// Start node, return nil if OK, error if not possible
func (n *node) start() (err error) {

	// If application didn't bind explicitly, we grab an ephemeral port
	// on all available network interfaces. This is orthogonal to
	// beaconing, since we can connect to other peers and they will
	// gossip our endpoint to others.

	if !n.bound {
		for i := dynPortFrom; i <= dynPortTo; i++ {
			rand.Seed(time.Now().UTC().UnixNano())
			port := uint16(rand.Intn(int(dynPortTo-dynPortFrom))) + dynPortFrom
			err = n.inbox.Bind(fmt.Sprintf("tcp://*:%d", port))
			if err == nil {
				n.port = port
				n.bound = true
				break
			} else {
				return err
			}
		}
	}
	// Start UDP beaconing, if the application didn't disable it
	if n.beaconPort > 0 {

		b := &aBeacon{}
		b.Protocol[0] = 'Z'
		b.Protocol[1] = 'R'
		b.Protocol[2] = 'E'
		b.Version = beaconVersion
		b.Uuid = n.uuid
		b.Port = n.port

		buffer := new(bytes.Buffer)
		binary.Write(buffer, binary.BigEndian, b.Protocol)
		binary.Write(buffer, binary.BigEndian, b.Version)
		binary.Write(buffer, binary.BigEndian, b.Uuid)
		binary.Write(buffer, binary.BigEndian, b.Port)

		// Create a beacon
		n.beacon, err = beacon.New(n.beaconPort)
		if err != nil {
			return err
		}
		n.beacons = n.beacon.Signals()

		if n.interval > 0 {
			n.beacon.SetInterval(n.interval)
		}

		n.beacon.NoEcho()
		n.beacon.Subscribe([]byte("ZRE"))
		n.beacon.Publish(buffer.Bytes())

		// Our own host endpoint is provided by the beacon
		if n.endpoint != "" {
			panic("Endpoint is already set")
		}

		n.endpoint = fmt.Sprintf("tcp://%s:%d", n.beacon.Addr(), n.port)
	} else if n.endpoint == "" {

		hostname, err := os.Hostname()
		if err != nil {
			return err
		}
		n.endpoint = fmt.Sprintf("tcp://%s:%d", hostname, n.port)
	}

	// Start polling on inbox
	go n.pollInbox()

	return
}

// Stop node discovery and interconnection
func (n *node) stop() {

	if n.beacon != nil {
		//  Stop broadcast/listen beacon
		b := &aBeacon{}
		b.Protocol[0] = 'Z'
		b.Protocol[1] = 'R'
		b.Protocol[2] = 'E'
		b.Version = beaconVersion
		b.Uuid = n.uuid
		b.Port = 0 // Zero means we're stopping

		buffer := new(bytes.Buffer)
		binary.Write(buffer, binary.BigEndian, b.Protocol)
		binary.Write(buffer, binary.BigEndian, b.Version)
		binary.Write(buffer, binary.BigEndian, b.Uuid)
		binary.Write(buffer, binary.BigEndian, b.Port)

		n.beacon.Publish(buffer.Bytes())
		time.Sleep(1 * time.Millisecond) // Allow 1 msec for beacon to go out

		n.beacon.Close()
	}
}

// recvFromApi handles a new command received from front-end
func (n *node) recvFromApi(c *cmd) {

	if n.verbose {
		log.Printf("[%s] Received a %q command from API", n.name, c.cmd)
	}

	switch c.cmd {
	case cmdSetName:
		n.name = c.payload.(string)

	case cmdSetHeader:
		n.headers[c.key] = c.payload.(string)

	case cmdSetVerbose:
		n.verbose = c.payload.(bool)

	case cmdSetPort:
		n.port = c.payload.(uint16)

	case cmdSetInterval:
		n.interval = c.payload.(time.Duration)

	case cmdUuid:
		uuid := fmt.Sprintf("%X", n.uuid)
		n.cmds <- &cmd{payload: uuid}

	case cmdName:
		n.cmds <- &cmd{payload: n.name}

	case cmdHeader:
		header, ok := n.headers[c.key]

		var err error
		if !ok {
			err = errors.New("Header doesn't exist")
		}

		n.cmds <- &cmd{err: err, payload: header}

	case cmdHeaders:
		n.cmds <- &cmd{payload: n.headers}

	case cmdBind:
		endpoint := c.payload.(string)
		err := n.bind(endpoint)
		// Signal the caller and send back the error if any
		n.cmds <- &cmd{payload: err}

	case cmdConnect:
		endpoint := c.payload.(string)
		err := n.connect(endpoint)
		// Signal the caller and send back the error if any
		n.cmds <- &cmd{payload: err}

	case cmdStart:
		err := n.start()
		// Signal the caller and send back the error if any
		n.cmds <- &cmd{err: err}

	case cmdStop, cmdTerm:
		close(n.terminated)

		// Wait and send the signal in a separate go routine
		// because closing terminated channel
		go func() {
			n.wg.Wait()
			// Signal the caller
			n.cmds <- &cmd{}
		}()

	case cmdWhisper:
		// Get peer to send message to
		peer, ok := n.peers[c.key]

		// Send frame on out to peer's mailbox, drop message
		// if peer doesn't exist (may have been destroyed)
		if ok {
			m := msg.NewWhisper()
			m.Content = c.payload.([]byte)
			peer.send(m)
		}

	case cmdShout:
		group := c.key
		// Get group to send message to
		if g, ok := n.peerGroups[group]; ok {
			m := msg.NewShout()
			m.Group = group
			m.Content = c.payload.([]byte)
			g.send(m)
		}

	case cmdJoin:
		group := c.key
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

	case cmdLeave:
		group := c.key
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

	default:
		panic(fmt.Sprintf("Invalid command %q %#v", c.cmd, c))
	}
}

// requirePeer finds or creates peer via its UUID string
func (n *node) requirePeer(identity string, endpoint string) (peer *peer) {
	peer, ok := n.peers[identity]
	if !ok {
		// Purge any previous peer on same endpoint
		for _, p := range n.peers {
			if p.endpoint == endpoint {
				p.disconnect()
			}
		}

		peer = newPeer(identity)
		uuid := fmt.Sprintf("%X", n.uuid)
		peer.connect(uuid, endpoint)

		// Handshake discovery by sending HELLO as first message
		m := msg.NewHello()
		m.Endpoint = n.endpoint
		m.Status = n.status
		m.Name = n.name
		for key := range n.ownGroups {
			m.Groups = append(m.Groups, key)
		}
		for key, header := range n.headers {
			m.Headers[key] = header
		}
		peer.send(m)
		n.peers[identity] = peer

		// TODO(armen): Send new peer event to logger, if any
	}

	return peer
}

// Remove a peer from our data structures.
func (n *node) removePeer(peer *peer) {
	// Tell the calling application the peer has gone
	n.events <- &Event{
		eventType: EventExit,
		sender:    peer.identity,
		name:      peer.name,
	}
	// TODO(armen): Send a log event

	// Remove peer from any groups we've got it in
	for _, group := range n.peerGroups {
		group.leave(peer)
	}

	// It's really important to disconnect from the peer before
	// deleting it, unless we'd end up difficulties to reconnect
	// to the same endpoint
	peer.disconnect()
	delete(n.peers, peer.identity)
}

// requirePeerGroup finds or creates group via its name
func (n *node) requirePeerGroup(name string) *group {
	group, ok := n.peerGroups[name]
	if !ok {
		group = newGroup(name)
		n.peerGroups[name] = group
	}

	return group
}

// joinPeerGroup joins the pear to a group
func (n *node) joinPeerGroup(peer *peer, name string) *group {
	group := n.requirePeerGroup(name)
	group.join(peer)

	// Now tell the caller about the peer joined group
	n.events <- &Event{
		eventType: EventJoin,
		sender:    peer.identity,
		name:      peer.name,
		group:     name,
	}

	return group
}

// leavePeerGroup leaves the pear to a group
func (n *node) leavePeerGroup(peer *peer, name string) *group {
	group := n.requirePeerGroup(name)
	group.leave(peer)

	// Now tell the caller about the peer left group
	n.events <- &Event{
		eventType: EventLeave,
		sender:    peer.identity,
		name:      peer.name,
		group:     name,
	}

	return group
}

// recvFromPeer handles messages coming from other peers
func (n *node) recvFromPeer(transit msg.Transit) {
	// Router socket tells us the identity of this peer
	// Identity must be [1] followed by 16-byte UUID, ignore the [1]
	identity := transit.Address()[1:]

	peer := n.peers[identity]

	if n.verbose {
		log.Printf("[%s] Received a %s message from %q peer", n.name, transit.String(), identity)
	}

	switch m := transit.(type) {
	case *msg.Hello:
		// On HELLO we may create the peer if it's unknown
		// On other cmds the peer must already exist
		if peer != nil {
			// Remove fake peers
			if peer.ready {
				n.removePeer(peer)
			} else if n.endpoint == peer.endpoint {
				// We ignore HELLO, if peer has same endpoint as current node
				return
			}
		}
		peer = n.requirePeer(identity, m.Endpoint)
		peer.ready = true
	}

	// Ignore command if peer isn't ready
	if peer == nil || !peer.ready {
		if peer != nil {
			n.removePeer(peer)
		}
		return
	}

	if !peer.checkMessage(transit) {
		log.Printf("W: [%s] lost messages from %s", n.name, identity)
		return
	}

	// Now process each command
	switch m := transit.(type) {
	case *msg.Hello:
		// Store properties from HELLO command into peer
		peer.name = m.Name
		peer.status = m.Status

		event := &Event{
			eventType: EventEnter,
			sender:    peer.identity,
			name:      peer.name,
			headers:   make(map[string]string),
		}

		// Store peer headers for future reference
		for key, val := range m.Headers {
			peer.headers[key] = val
			event.headers[key] = val
		}

		n.events <- event

		// Join peer to listed groups
		for _, group := range m.Groups {
			n.joinPeerGroup(peer, group)
		}

		// TODO(armen): If peer is a ZRE/LOG collector, connect to it

	case *msg.Whisper:
		// Pass up to caller API as WHISPER event
		n.events <- &Event{
			eventType: EventWhisper,
			sender:    identity,
			name:      peer.name,
			msg:       m.Content,
		}

	case *msg.Shout:
		// Pass up to caller as SHOUT event
		n.events <- &Event{
			eventType: EventShout,
			sender:    identity,
			name:      peer.name,
			group:     m.Group,
			msg:       m.Content,
		}

	case *msg.Ping:
		ping := msg.NewPingOk()
		peer.send(ping)

	case *msg.Join:
		n.joinPeerGroup(peer, m.Group)
		if m.Status != peer.status {
			panic(fmt.Sprintf("[%X] message status isn't equal to peer status, %d != %d", n.uuid, m.Status, peer.status))
		}

	case *msg.Leave:
		n.leavePeerGroup(peer, m.Group)
		if m.Status != peer.status {
			panic(fmt.Sprintf("[%X] message status isn't equal to peer status, %d != %d", n.uuid, m.Status, peer.status))
		}
	}

	// Activity from peer resets peer timers
	peer.refresh()
}

// recvFromBeacon handles a new signal received from beacon
func (n *node) recvFromBeacon(s *beacon.Signal) {

	// Get IP address and beacon of peer
	ipaddress := s.Addr

	b := &aBeacon{}
	buffer := bytes.NewBuffer(s.Transmit)
	binary.Read(buffer, binary.BigEndian, &b.Protocol)
	binary.Read(buffer, binary.BigEndian, &b.Version)

	uuid := make([]byte, 16)
	binary.Read(buffer, binary.BigEndian, uuid)
	b.Uuid = append(b.Uuid, uuid...)

	binary.Read(buffer, binary.BigEndian, &b.Port)

	// Ignore anything that isn't a valid beacon
	if b.Version == beaconVersion {
		// Check that the peer, identified by its UUID, exists
		identity := fmt.Sprintf("%X", b.Uuid)

		if n.verbose {
			log.Printf("[%s] Received a beacon from %q peer (tcp://%s:%d)", n.name, identity, ipaddress, b.Port)
		}

		if b.Port != 0 {
			endpoint := fmt.Sprintf("tcp://%s:%d", ipaddress, b.Port)
			peer := n.requirePeer(identity, endpoint)
			peer.refresh()
		} else {
			// Zero port means peer is going away; remove it if
			// we had any knowledge of it already
			peer := n.peers[identity]
			n.removePeer(peer)
		}
	} else if n.verbose {
		log.Printf("[%s] Received a beacon with invalid version number %d", n.name, b.Version)
	}
}

// We do this once a second:
// - if peer has gone quiet, send TCP ping
// - if peer has disappeared, expire it
func (n *node) pingPeer(peer *peer) {
	if time.Now().Unix() >= peer.expiredAt.Unix() {
		n.removePeer(peer)
	} else if time.Now().Unix() >= peer.evasiveAt.Unix() {
		//  If peer is being evasive, force a TCP ping.
		//  TODO(armen): do this only once for a peer in this state;
		//  it would be nicer to use a proper state machine
		//  for peer management.
		m := msg.NewPing()
		peer.send(m)
	}
}

// Terminate leaves all the groups and the closes all the connections to the peers
func (n *node) terminate() {
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

func nodeActor(events chan *Event, cmds chan *cmd) {
	node, err := newNode(events, cmds)
	node.terminated = make(chan struct{})

	if err != nil {
		msg := fmt.Sprintf("%s", err)
		node.events <- &Event{msg: []byte(msg)}
	} else {
		node.events <- &Event{}
	}

	go node.handler()
}

func (n *node) handler() {
	defer func() {
		n.wg.Done()
	}()

	ping := time.After(reapInterval)
	stype, _ := n.inbox.GetType()

	for {
		select {
		case <-n.terminated:
			// Quiting
			n.stop()
			n.terminate()
			return

		case c := <-n.cmds:
			// Received a command from the caller/API
			n.recvFromApi(c)

		case frames := <-n.inboxChan:
			transit, err := msg.Unmarshal(stype, frames...)
			if err != nil {
				continue
			}
			n.recvFromPeer(transit)

		case s := <-n.beacons:
			n.recvFromBeacon(s)

		case <-ping:
			if n.verbose {
				log.Printf("[%s] Pings every other peer", n.name)
			}
			ping = time.After(reapInterval)
			for _, peer := range n.peers {
				n.pingPeer(peer)
			}
		}
	}
}

// Poll from inbox and proxy it into a channel so that we can read
// everything in one unified select
func (n *node) pollInbox() {
	poller := zmq.NewPoller()
	poller.Add(n.inbox, zmq.POLLIN)

	terminated := false

	// Wait for termination signal on a go routine
	go func() {
		select {
		case <-n.terminated:
			terminated = true
		}
	}()

	for {
		sockets, _ := poller.Poll(250 * time.Millisecond)
		for _, socket := range sockets {
			switch s := socket.Socket; s {
			case n.inbox:
				msg, err := s.RecvMessageBytes(0)
				if err != nil {

				}
				n.inboxChan <- msg
			}
		}
		if terminated {
			// Received a termination signal kill the go routine
			return
		}
	}
}
