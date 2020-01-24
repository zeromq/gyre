package gyre

import (
	"bytes"
	crand "crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/armen/goviral/zgossip"
	zmq "github.com/pebbe/zmq4"
	"github.com/zeromq/gyre/beacon"
	"github.com/zeromq/gyre/zre/msg"
)

type node struct {
	reactor       *zmq.Reactor
	terminated    chan interface{}  // API shut us down
	wg            sync.WaitGroup    // wait group is used to wait until actor() is done
	events        chan *Event       // We send all Gyre events to the events channel
	cmds          chan interface{}  // Receive commands from the cmds channel
	replies       chan interface{}  // Send command replies to the replies channel
	verbose       bool              // Log all traffic
	beaconPort    int               // Beacon port number
	interval      time.Duration     // Beacon interval
	beacon        *beacon.Beacon    // Beacon object
	uuid          []byte            // Our UUID
	inbox         *zmq.Socket       // Our inbox socket (ROUTER)
	name          string            // Our public name
	endpoint      string            // Our public endpoint
	port          uint16            // Our inbox port number
	bound         bool              // Did app bind node explicitly?
	status        byte              // Our own change counter
	peers         map[string]*peer  // Hash of known peers, fast lookup
	peerGroups    map[string]*group // Groups that our peers are in
	ownGroups     map[string]*group // Groups that we are in
	headers       map[string]string // Our header values
	gossip        *zgossip.Zgossip  // Gossip discovery service, if any
	gossipBind    string            // Gossip bind endpoint, if any
	gossipConnect string            // Gossip connect endpoint, if any
}

// Beacon frame has this format:
//
// Z R E       3 bytes
// Version     1 byte, %x01
// UUID        16 bytes
// Port        2 bytes in network order
type aBeacon struct {
	Protocol [3]byte
	Version  byte
	UUID     []byte
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
func newNode(events chan *Event, cmds chan interface{}, replies chan interface{}) (n *node, err error) {
	n = &node{
		reactor:    zmq.NewReactor(),
		events:     events,
		cmds:       cmds,
		replies:    replies,
		beaconPort: zreDiscoveryPort,
		peers:      make(map[string]*peer),
		peerGroups: make(map[string]*group),
		ownGroups:  make(map[string]*group),
		headers:    make(map[string]string),
		terminated: make(chan interface{}),
	}

	n.beacon = beacon.New()

	n.inbox, err = zmq.NewSocket(zmq.ROUTER)
	if err != nil {
		return nil, err // Could not create new socket
	}
	err = n.inbox.SetIpv6(true)
	if err != nil {
		return nil, err
	}

	// Generate random uuid
	n.uuid = make([]byte, 16)
	io.ReadFull(crand.Reader, n.uuid)

	// Default name for node is first 6 characters of UUID:
	// the shorter string is more readable in logs
	n.name = fmt.Sprintf("%.6s", fmt.Sprintf("%X", n.uuid))

	n.wg.Add(1) // We're going to wait until actor() is done

	return
}

// If we haven't already set-up the gossip network, do so
func (n *node) gossipStart() (err error) {
	n.beaconPort = 0 // Disable UDP beaconing
	if n.gossip == nil {
		n.gossip, err = zgossip.New(n.identity())
		if err != nil {
			return
		}

		if n.verbose {
			n.gossip.SendCmd("VERBOSE", nil, 100*time.Millisecond)
		}
	}

	return err
}

// Start node, return nil if OK, error if not possible
func (n *node) start() (err error) {

	// If application didn't bind explicitly, we grab an ephemeral port
	// on all available network interfaces. This is orthogonal to
	// beaconing, since we can connect to other peers and they will
	// gossip our endpoint to others.
	if !n.bound {
		_, n.port, err = bind(n.inbox, "tcp://*:0")
		if err != nil {
			return err
		}
		n.bound = true
	}

	// Start UDP beaconing, if the application didn't disable it
	if n.beaconPort > 0 {

		b := &aBeacon{}
		b.Protocol[0] = 'Z'
		b.Protocol[1] = 'R'
		b.Protocol[2] = 'E'
		b.Version = beaconVersion
		b.UUID = n.uuid
		b.Port = n.port

		buffer := new(bytes.Buffer)
		binary.Write(buffer, binary.BigEndian, b.Protocol)
		binary.Write(buffer, binary.BigEndian, b.Version)
		binary.Write(buffer, binary.BigEndian, b.UUID)
		binary.Write(buffer, binary.BigEndian, b.Port)

		if n.interval > 0 {
			n.beacon.SetInterval(n.interval)
		}

		n.beacon.SetPort(n.beaconPort)
		n.beacon.NoEcho()
		n.beacon.Subscribe([]byte("ZRE"))
		err := n.beacon.Publish(buffer.Bytes())
		if err != nil {
			return err
		}

		// Our own host endpoint is provided by the beacon
		if n.endpoint != "" {
			panic("Endpoint is already set")
		}
		ip := net.ParseIP(n.beacon.Addr())
		if ip.To4() == nil {
			n.endpoint = fmt.Sprintf("tcp://[%s]:%d", ip.String(), n.port)
		} else {
			n.endpoint = fmt.Sprintf("tcp://%s:%d", ip.String(), n.port)
		}

		n.reactor.AddChannel(n.beacon.Signals(), 1, func(s interface{}) error {
			n.recvFromBeacon(s.(*beacon.Signal))
			return nil
		})

	} else {

		if n.endpoint == "" {
			hostname, err := os.Hostname()
			if err != nil {
				return err
			}
			n.endpoint = fmt.Sprintf("tcp://%s:%d", hostname, n.port)
		}

		if n.gossip == nil {
			return errors.New("Gossip engine hasn't been started yet, use SetEndpoint, GossipBind or GossipConnect")
		}

		n.gossip.SendCmd("PUBLISH", map[string]string{n.identity(): n.endpoint}, 100*time.Millisecond)

		// Start polling on zgossip
		n.reactor.AddChannel(n.gossip.Resp(), 1, func(r interface{}) error {
			n.recvFromGossip(r)
			return nil
		})
	}

	return
}

// Stop node discovery and interconnection
func (n *node) stop() {

	if n.beacon != nil {
		// Stop broadcast/listen beacon
		b := &aBeacon{}
		b.Protocol[0] = 'Z'
		b.Protocol[1] = 'R'
		b.Protocol[2] = 'E'
		b.Version = beaconVersion
		b.UUID = n.uuid
		b.Port = 0 // Zero means we're stopping

		buffer := new(bytes.Buffer)
		binary.Write(buffer, binary.BigEndian, b.Protocol)
		binary.Write(buffer, binary.BigEndian, b.Version)
		binary.Write(buffer, binary.BigEndian, b.UUID)
		binary.Write(buffer, binary.BigEndian, b.Port)

		n.beacon.Publish(buffer.Bytes())
		time.Sleep(1 * time.Millisecond) // Allow 1 msec for beacon to go out

		n.beacon.Close()
	}
}

// recvFromAPI handles a new command received from front-end
func (n *node) recvFromAPI(c *cmd) {

	if n.verbose {
		log.Printf("[%s] Received a %q command from API", n.name, c.cmd)
	}

	switch c.cmd {
	case cmdUUID:
		n.replies <- &reply{cmd: cmdUUID, payload: n.identity()}

	case cmdName:
		n.replies <- &reply{cmd: cmdName, payload: n.name}

	case cmdSetName:
		n.name = c.payload.(string)

	case cmdSetHeader:
		n.headers[c.key] = c.payload.(string)

	case cmdSetVerbose:
		n.verbose = c.payload.(bool)
		// n.reactor.SetVerbose(n.verbose)

	case cmdSetPort:
		n.beaconPort = c.payload.(int)

	case cmdSetInterval:
		// Set beacon interval
		n.interval = c.payload.(time.Duration)

	case cmdSetIface:
		n.beacon.SetInterface(c.payload.(string))

	case cmdSetEndpoint:
		err := n.gossipStart()
		if err != nil {
			// Signal the caller and send back the error if any
			n.replies <- &reply{cmd: cmdSetEndpoint, err: err}
			break
		}

		endpoint := c.payload.(string)
		n.endpoint, _, err = bind(n.inbox, endpoint)
		if err != nil {
			n.replies <- &reply{cmd: cmdSetEndpoint, err: err}
			break
		}
		n.bound = true
		n.beaconPort = 0

		n.replies <- &reply{cmd: cmdSetEndpoint}

	case cmdGossipBind:
		err := n.gossipStart()
		if err != nil {
			n.replies <- &reply{cmd: cmdGossipBind, err: err}
			break
		}

		endpoint := c.payload.(string)
		err = n.gossip.SendCmd("BIND", endpoint, 5*time.Second)
		n.replies <- &reply{cmd: cmdGossipBind, err: err}

	case cmdGossipPort:
		err := n.gossip.SendCmd("PORT", nil, 5*time.Second)
		if err != nil {
			n.replies <- &reply{cmd: cmdGossipPort, err: err}
			break
		}
		port, err := n.gossip.RecvResp(5 * time.Second)
		if err != nil {
			n.replies <- &reply{cmd: cmdGossipPort, err: err}
			break
		}
		n.replies <- &reply{cmd: cmdGossipPort, payload: strconv.FormatUint(uint64(port.(uint16)), 10)}

	case cmdGossipConnect:
		err := n.gossipStart()
		if err != nil {
			n.replies <- &reply{cmd: cmdGossipConnect, err: err}
			break
		}

		endpoint := c.payload.(string)
		err = n.gossip.SendCmd("CONNECT", endpoint, 5*time.Second)
		n.replies <- &reply{cmd: cmdGossipConnect, err: err}

	case cmdStart:
		// Add the ping ticker just right before start so that it reads the latest
		// value of loopInterval
		n.reactor.AddChannelTime(time.Tick(loopInterval), 1, func(interface{}) error {
			n.ping()
			return nil
		})

		err := n.start()
		// Signal the caller and send back the error if any
		n.replies <- &reply{cmd: cmdStart, err: err}

	case cmdStop, cmdTerm:
		if n.terminated != nil {
			close(n.terminated)
		}

		// Wait and send the signal in a separate go routine
		// because closing terminated channel
		go func() {
			n.wg.Wait()
			// Signal the caller
			n.replies <- &reply{}
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

	case cmdDump:
		// TODO: implement DUMP

	case cmdAddr:
		if n.beaconPort > 0 {
			n.replies <- &reply{cmd: cmdAddr, payload: n.beacon.Addr()}
		} else {
			u, err := url.Parse(n.endpoint)
			if err != nil {
				n.replies <- &reply{cmd: cmdHeader, err: err}
				return
			}
			ip, _, err := net.SplitHostPort(u.Host)
			if err != nil {
				n.replies <- &reply{cmd: cmdHeader, err: err}
				return
			}

			n.replies <- &reply{cmd: cmdAddr, payload: ip}
		}

	case cmdHeader:
		header, ok := n.headers[c.key]

		var err error
		if !ok {
			err = errors.New("Header doesn't exist")
		}

		n.replies <- &reply{cmd: cmdHeader, err: err, payload: header}

	case cmdHeaders:
		n.replies <- &reply{cmd: cmdHeader, payload: n.headers}

	default:
		log.Printf("Invalid command %q %#v", c.cmd, c)
	}
}

func (n *node) identity() string {
	return fmt.Sprintf("%X", n.uuid)
}

// requirePeer finds or creates peer via its UUID string
func (n *node) requirePeer(identity string, endpoint string) (peer *peer, err error) {
	peer, ok := n.peers[identity]
	if !ok {
		// Purge any previous peer on same endpoint
		for _, p := range n.peers {
			if p.endpoint == endpoint {
				p.disconnect()
			}
		}

		peer = newPeer(identity)
		err = peer.connect(n.uuid, endpoint)
		if err != nil {
			return nil, err
		}

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

	return peer, nil
}

// Remove a peer from our data structures.
func (n *node) removePeer(peer *peer) {
	if peer == nil {
		return
	}

	// Tell the calling application the peer has gone
	select {
	case n.events <- &Event{eventType: EventExit, sender: peer.identity, name: peer.name}:
	default:
		if n.verbose {
			log.Printf("[%s] Dropping event: %s", n.name, EventExit)
		}
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

// joinPeerGroup joins the peer to a group
func (n *node) joinPeerGroup(peer *peer, name string) *group {
	group := n.requirePeerGroup(name)
	group.join(peer)

	// Now tell the caller about the peer joined group
	select {
	case n.events <- &Event{eventType: EventJoin, sender: peer.identity, name: peer.name, group: name}:
	default:
		if n.verbose {
			log.Printf("[%s] Dropping event: %s", n.name, EventJoin)
		}
	}

	return group
}

// leavePeerGroup leaves the peer to a group
func (n *node) leavePeerGroup(peer *peer, name string) *group {
	group := n.requirePeerGroup(name)
	group.leave(peer)

	// Now tell the caller about the peer left group
	select {
	case n.events <- &Event{eventType: EventLeave, sender: peer.identity, name: peer.name, group: name}:
	default:
		if n.verbose {
			log.Printf("[%s] Dropping event: %s", n.name, EventLeave)
		}
	}

	return group
}

// recvFromPeer handles messages coming from other peers
func (n *node) recvFromPeer(transit msg.Transit) {
	if transit == nil {
		// Invalid transit
		return
	}

	routingID := transit.RoutingID()
	if len(routingID) < 1 {
		// Invalid routing id, ignore the peer
		return
	}

	// Router socket tells us the identity of this peer
	// Identity must be [1] followed by 16-byte UUID, ignore the [1]
	identity := fmt.Sprintf("%X", routingID[1:])

	peer := n.peers[identity]

	if n.verbose {
		for i, str := range strings.Split(transit.String(), "\n") {
			if len(str) <= 0 {
				continue
			}

			if i == 0 && peer != nil {
				log.Printf("[%s] %s %s", n.name, peer.name, str)
			} else {
				log.Printf("[%s] %s", n.name, str)
			}
		}
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
		var err error
		peer, err = n.requirePeer(identity, m.Endpoint)
		if err == nil {
			peer.ready = true
		} else if n.verbose {
			log.Printf("[%s] %s", n.name, err)
		}
	}

	// Ignore command if peer isn't ready
	if peer == nil || !peer.ready {
		if peer != nil {
			n.removePeer(peer)
		}
		return
	}

	if !peer.checkMessage(transit) {
		log.Printf("[%s] lost messages from %s", n.name, identity)
		return
	}

	// Now process each command
	switch m := transit.(type) {
	case *msg.Hello:
		// Store properties from HELLO command into peer
		peer.name = m.Name

		event := &Event{
			eventType: EventEnter,
			sender:    peer.identity,
			name:      peer.name,
			address:   strings.SplitN(strings.TrimPrefix(m.Endpoint, "tcp://"), ":", 2)[0],
			headers:   make(map[string]string),
		}

		// Store peer headers for future reference
		for key, val := range m.Headers {
			peer.headers[key] = val
			event.headers[key] = val
		}

		select {
		case n.events <- event:
		default:
			if n.verbose {
				log.Printf("[%s] Dropping event: %s", n.name, EventEnter)
			}
		}

		// Join peer to listed groups
		for _, group := range m.Groups {
			n.joinPeerGroup(peer, group)
		}

		// Now take peer's status from HELLO, after joining groups
		peer.status = m.Status

		// TODO(armen): If peer is a ZRE/LOG collector, connect to it

	case *msg.Whisper:
		// Pass up to caller API as WHISPER event
		select {
		case n.events <- &Event{eventType: EventWhisper, sender: identity, name: peer.name, msg: m.Content}:
		default:
			if n.verbose {
				log.Printf("[%s] Dropping event: %s", n.name, EventWhisper)
			}
		}

	case *msg.Shout:
		// Pass up to caller as SHOUT event
		select {
		case n.events <- &Event{eventType: EventShout, sender: identity, name: peer.name, group: m.Group, msg: m.Content}:
		default:
			if n.verbose {
				log.Printf("[%s] Dropping event: %s", n.name, EventShout)
			}
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

	b := &aBeacon{}
	buffer := bytes.NewBuffer(s.Transmit)
	binary.Read(buffer, binary.BigEndian, &b.Protocol)
	binary.Read(buffer, binary.BigEndian, &b.Version)

	uuid := make([]byte, 16)
	binary.Read(buffer, binary.BigEndian, uuid)
	b.UUID = append(b.UUID, uuid...)

	binary.Read(buffer, binary.BigEndian, &b.Port)

	// Ignore anything that isn't a valid beacon
	if b.Version == beaconVersion {
		// Check that the peer, identified by its UUID, exists
		identity := fmt.Sprintf("%X", b.UUID)

		if b.Port != 0 {
			var endpoint string
			// s.Addr is IP address of peer beacon
			ip := net.ParseIP(s.Addr)
			if ip.To4() == nil {
				endpoint = fmt.Sprintf("tcp://[%s]:%d", ip.String(), b.Port)
			} else {
				endpoint = fmt.Sprintf("tcp://%s:%d", ip.String(), b.Port)
			}
			peer, err := n.requirePeer(identity, endpoint)
			if err == nil {
				peer.refresh()
			} else if n.verbose {
				log.Printf("[%s] %s", n.name, err)
			}
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

// recvFromGossip handles a new response received from gossip
func (n *node) recvFromGossip(r interface{}) {

	resp := r.(*zgossip.Resp)

	if n.verbose {
		log.Printf("[%s] recvFromGossip: %#v", n.name, resp.Payload.(map[string]string))
	}

	for identity, endpoint := range resp.Payload.(map[string]string) {
		if endpoint != n.endpoint {
			peer, err := n.requirePeer(identity, endpoint)
			if err == nil {
				peer.refresh()
			} else if n.verbose {
				log.Printf("[%s] %s", n.name, err)
			}
		}
	}
}

// We do this once a second:
// - if peer has gone quiet, send TCP ping
// - if peer has disappeared, expire it
func (n *node) pingPeer(peer *peer) {
	if time.Now().Unix() >= peer.expiredAt.Unix() {
		n.removePeer(peer)
	} else if time.Now().Unix() >= peer.evasiveAt.Unix() {
		// If peer is being evasive, force a TCP ping.
		// TODO(armen): do this only once for a peer in this state;
		// it would be nicer to use a proper state machine
		// for peer management.
		m := msg.NewPing()
		peer.send(m)
	}
}

// Terminate leaves all the groups and the closes all the connections to the peers
func (n *node) terminate() {
	// Disconnect from all peers
	for peerID, peer := range n.peers {
		// It's really important to disconnect from the peer before
		// deleting it, unless we'd end up difficulties to reconnect
		// to the same endpoint
		peer.disconnect()
		delete(n.peers, peerID)
	}
	// Now it's safe to close the socket
	n.inbox.Unbind(fmt.Sprintf("tcp://*:%d", n.port))
	n.inbox.Close()
}

func (n *node) actor() {
	defer func() {
		n.wg.Done()
	}()

	// Handle terminate signal
	n.reactor.AddChannel(n.terminated, 1, func(interface{}) error {
		// Quiting
		n.stop()
		n.terminate()

		return errors.New("terminate")
	})

	// Received a command from the caller/API
	n.reactor.AddChannel(n.cmds, 1, func(c interface{}) error {
		n.recvFromAPI(c.(*cmd))
		return nil
	})

	// Handle the inbox
	n.reactor.AddSocket(n.inbox, zmq.POLLIN, func(s zmq.State) error {
		transit, err := msg.Recv(n.inbox)
		if err != nil {
			if n.verbose {
				log.Printf("[%s] %s", n.name, err)
			}
			return nil
		}
		n.recvFromPeer(transit)

		return nil
	})

	n.reactor.Run(10 * time.Millisecond)
}

func (n *node) ping() {
	if n.verbose && len(n.peers) == 0 {
		log.Printf("[%s] There is no peer to ping", n.name)
	}

	for _, peer := range n.peers {
		n.pingPeer(peer)
	}
}

func bind(sock *zmq.Socket, endpoint string) (string, uint16, error) {

	var port uint16

	e, err := url.Parse(endpoint)
	if err != nil {
		return endpoint, 0, err
	}

	if e.Scheme == "inproc" {
		err = sock.Bind(endpoint)
		return endpoint, 0, err
	}
	ip, p, err := net.SplitHostPort(e.Host)
	if err != nil {
		return endpoint, 0, err
	}

	if p == "0" {
		for i := dynPortFrom; i <= dynPortTo; i++ {
			rand.Seed(time.Now().UTC().UnixNano())
			port = uint16(rand.Intn(int(dynPortTo-dynPortFrom))) + dynPortFrom
			endpoint = fmt.Sprintf("%s://%s:%d", e.Scheme, ip, port)
			err = sock.Bind(endpoint)
			if err == nil {
				break
			} else if err.Error() == "no sock.ch device" {
				port = 0
				err = fmt.Errorf("no sock.ch device: %s", endpoint)
				break
			} else if i-dynPortFrom > 100 {
				err = errors.New("Unable to bind to an ephemeral port")
				break
			}
		}

		return endpoint, port, err
	}

	pp, err := strconv.ParseUint(p, 10, 16)
	if err != nil {
		return endpoint, 0, err
	}
	port = uint16(pp)
	err = sock.Bind(endpoint)

	return endpoint, port, err
}
