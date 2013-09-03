// Package beacon implements a peer-to-peer discovery service for local
// networks. A beacon can broadcast and/or capture service announcements
// using UDP messages on the local area network. This implementation uses
// IPv4 UDP broadcasts. You can define the format of your outgoing beacons,
// and set a filter that validates incoming beacons. Beacons are sent and
// received asynchronously in the background.
//
// This package is an idiomatic go translation of zbeacon class of czmq at
// following address:
//      https://github.com/zeromq/czmq
//
// Instead of ZMQ_PEER socket it uses go channel and also uses go routine
// instead of zthread. To simplify the implementation it doesn't pass API
// calls through the pipe (as zbeacon does) instead it modifies beacon
// struct directly.
//
package beacon

import (
	"bytes"
	"net"
	"time"
)

const (
	beaconMax       = 255
	defaultInterval = 1 * time.Second
)

type Signal struct {
	Addr     string
	Transmit []byte
}

type beacon struct {
	signals    chan *Signal
	ticker     <-chan time.Time
	conn       *net.UDPConn
	interval   time.Duration
	noecho     bool
	terminated bool
	transmit   []byte
	filter     []byte
	addr       string
	port       int
	broadcast  string // TODO: need to figure out the broadcast address
	hostname   string
}

// New creates a new beacon
func New(port int) (*beacon, error) {

	bcast := &net.UDPAddr{Port: port, IP: net.IPv4bcast}
	conn, err := net.ListenUDP("udp", bcast)
	if err != nil {
		return nil, err
	}

	var hostname string
	addr := conn.LocalAddr().String()
	name, err := net.LookupAddr(addr)
	if err == nil {
		hostname = name[0]
	} else {
		hostname = addr
	}

	b := &beacon{
		signals:  make(chan *Signal),
		conn:     conn,
		interval: defaultInterval,
		hostname: hostname,
		addr:     addr,
		port:     port,
	}

	go b.listen()
	go b.signal()

	return b, nil
}

// Close terminates the beacon
func (b *beacon) Close() {
	b.terminated = true
	close(b.signals)
}

// SetInterval sets broadcast interval
func (b *beacon) SetInterval(interval time.Duration) *beacon {
	b.interval = interval
	return b
}

// NoEcho filters out any beacon that looks exactly like ours
func (b *beacon) NoEcho() *beacon {
	b.noecho = true
	return b
}

// Publish starts broadcasting beacon to peers at the specified interval
func (b *beacon) Publish(transmit []byte) *beacon {
	b.transmit = transmit
	if b.interval == 0 {
		b.ticker = time.After(defaultInterval)
	} else {
		b.ticker = time.After(b.interval)
	}
	return b
}

// Silence stops broadcasting beacon
func (b *beacon) Silence() *beacon {
	b.transmit = nil
	return b
}

// Subscribe starts listening to other peers; zero-sized filter means get everything
func (b *beacon) Subscribe(filter []byte) *beacon {
	b.filter = filter
	return b
}

// Unsubscribe stops listening to other peers
func (b *beacon) Unsubscribe() *beacon {
	b.filter = nil
	return b
}

// Hostname returns our own IP address as printable string
func (b *beacon) Hostname() string {
	return b.hostname
}

// Signals returns signals channel
func (b *beacon) Signals() chan *Signal {
	return b.signals
}

func (b *beacon) listen() {
	for {
		buff := make([]byte, beaconMax)
		n, addr, err := b.conn.ReadFrom(buff)
		if err != nil || n > beaconMax {
			continue
		}

		send := bytes.HasPrefix(buff[:n], b.filter)
		if send && b.noecho {
			send = !bytes.Equal(buff[:n], b.transmit)
		}

		if send {
			// Received a signal, send it to the signals channel
			b.signals <- &Signal{addr.String(), buff[:n]}
		}
	}
}

func (b *beacon) signal() {
	for {
		select {
		case <-b.ticker:
			if b.terminated {
				break
			}
			if b.transmit != nil {
				// Signal other beacons
				bcast := &net.UDPAddr{Port: b.port, IP: net.IPv4bcast}
				b.conn.WriteTo(b.transmit, bcast)
			}
			b.ticker = time.After(b.interval)
		}
	}
}
