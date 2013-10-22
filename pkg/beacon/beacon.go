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
// For more information please visit:
//		http://hintjens.com/blog:32
//
package beacon

import (
	"bytes"
	"errors"
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

type Beacon struct {
	Signals    chan *Signal
	conn       *net.UDPConn  // UDP connection for send/recv
	Port       int           // UDP port number we work on
	Interval   time.Duration // Beacon broadcast interval
	noecho     bool          // Ignore own (unique) beacons
	terminated bool          // API shut us down
	transmit   []byte        // Beacon transmit data
	filter     []byte        // Beacon filter data
	Addr       string        // Our own address
	mcast      *net.UDPAddr  // Our broadcast/multicast address
	ticker     <-chan time.Time
}

// Creates a new beacon on a certain UDP port.
func New(port int) (*Beacon, error) {

	var (
		ip    net.IP
		found bool
		mcast *net.UDPAddr
	)

	ifs, err := net.Interfaces()
	for _, iface := range ifs {
		if iface.Flags&net.FlagLoopback == 0 && iface.Flags&net.FlagBroadcast != 0 {
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}

			mcasts, err := iface.MulticastAddrs()
			if err != nil {
				continue
			}
			ip, _, _ = net.ParseCIDR(addrs[0].String())
			mcast = &net.UDPAddr{Port: port, IP: net.ParseIP(mcasts[0].String())}
			found = true
			break
		}
	}

	if !found {
		return nil, errors.New("no interfaces to bind to")
	}

	conn, err := net.ListenUDP("udp", mcast)
	if err != nil {
		return nil, err
	}

	b := &Beacon{
		Signals:  make(chan *Signal),
		Interval: defaultInterval,
		Addr:     ip.String(),
		Port:     port,
		conn:     conn,
		mcast:    mcast,
	}

	go b.listen()
	go b.signal()

	return b, nil
}

// Terminates the beacon.
func (b *Beacon) Close() {
	b.terminated = true
	close(b.Signals)
}

// SetInterval sets broadcast interval.
func (b *Beacon) SetInterval(interval time.Duration) *Beacon {
	b.Interval = interval
	return b
}

// NoEcho filters out any beacon that looks exactly like ours.
func (b *Beacon) NoEcho() *Beacon {
	b.noecho = true
	return b
}

// Publish starts broadcasting beacon to peers at the specified interval.
func (b *Beacon) Publish(transmit []byte) *Beacon {
	b.transmit = transmit
	if b.Interval == 0 {
		b.ticker = time.After(defaultInterval)
	} else {
		b.ticker = time.After(b.Interval)
	}
	return b
}

// Silence stops broadcasting beacon.
func (b *Beacon) Silence() *Beacon {
	b.transmit = nil
	return b
}

// Subscribe starts listening to other peers; zero-sized filter means get everything.
func (b *Beacon) Subscribe(filter []byte) *Beacon {
	b.filter = filter
	return b
}

// Unsubscribe stops listening to other peers.
func (b *Beacon) Unsubscribe() *Beacon {
	b.filter = nil
	return b
}

// Chan returns Signals channel.
func (b *Beacon) Chan() chan *Signal {
	return b.Signals
}

func (b *Beacon) listen() {
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

		if send && !b.terminated {
			// Send the arrived signal to the Signals channel
			select {
			case b.Signals <- &Signal{addr.String(), buff[:n]}:
			default:
			}
		}
	}
}

func (b *Beacon) signal() {
	for {
		select {
		case <-b.ticker:
			if b.terminated {
				break
			}
			if b.transmit != nil {
				// Signal other beacons
				b.conn.WriteToUDP(b.transmit, b.mcast)
			}
			b.ticker = time.After(b.Interval)
		}
	}
}
