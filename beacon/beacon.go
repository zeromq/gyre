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
	"code.google.com/p/go.net/ipv4"
	"code.google.com/p/go.net/ipv6"

	"bytes"
	"errors"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	beaconMax       = 255
	defaultInterval = 1 * time.Second
)

var (
	ipv4Group = "224.0.0.250"
	ipv6Group = "ff02::fa"
)

type Signal struct {
	Addr     string
	Transmit []byte
}

type Beacon struct {
	signals    chan *Signal
	ipv4Conn   *ipv4.PacketConn // UDP incoming connection for sending/receiving beacons
	ipv6Conn   *ipv6.PacketConn // UDP incoming connection for sending/receiving beacons
	ipv4       bool             // Whether or not connection is in ipv4 mode
	port       int              // UDP port number we work on
	interval   time.Duration    // Beacon broadcast interval
	noecho     bool             // Ignore own (unique) beacons
	terminated bool             // API shut us down
	transmit   []byte           // Beacon transmit data
	filter     []byte           // Beacon filter data
	addr       string           // Our own address
	ticker     <-chan time.Time
	wg         sync.WaitGroup
	inAddr     *net.UDPAddr
	outAddr    *net.UDPAddr
}

// Creates a new beacon on a certain UDP port.
func New(port int) (b *Beacon, err error) {

	b = &Beacon{
		signals:  make(chan *Signal, 50),
		interval: defaultInterval,
		port:     port,
	}

	// TODO(armen): An API should be implemented to set the interface name
	i := os.Getenv("BEACON_INTERFACE")
	if i == "" {
		i = os.Getenv("ZSYS_INTERFACE")
	}

	var ifs []net.Interface

	if i == "" {
		ifs, err = net.Interfaces()
		if err != nil {
			return nil, err
		}

	} else {
		iface, err := net.InterfaceByName(i)
		if err != nil {
			return nil, err
		}
		ifs = append(ifs, *iface)
	}

	conn, err := net.ListenPacket("udp4", net.JoinHostPort("224.0.0.0", strconv.Itoa(b.port)))
	if err == nil {
		b.ipv4Conn = ipv4.NewPacketConn(conn)
		b.ipv4Conn.SetMulticastLoopback(true)
	}

	if !b.ipv4 {
		conn, err := net.ListenPacket("udp6", net.JoinHostPort(net.IPv6linklocalallnodes.String(), strconv.Itoa(b.port)))
		if err != nil {
			return nil, err
		}

		b.ipv6Conn = ipv6.NewPacketConn(conn)
		b.ipv6Conn.SetMulticastLoopback(true)
	}

	for _, iface := range ifs {
		if b.ipv4Conn != nil {
			b.inAddr = &net.UDPAddr{
				IP: net.ParseIP(ipv4Group),
			}
			b.ipv4Conn.JoinGroup(&iface, b.inAddr)

			// Find IP of the interface
			// TODO(armen): Let user set the ipaddress which here can be verified to be valid
			addrs, err := iface.Addrs()
			if err != nil {
				return nil, err
			}
			ip, _, err := net.ParseCIDR(addrs[0].String())
			if err != nil {
				return nil, err
			}
			b.addr = ip.String()

			if iface.Flags&net.FlagLoopback != 0 {
				b.outAddr = &net.UDPAddr{IP: net.IPv4allsys, Port: b.port}
			} else {
				b.outAddr = &net.UDPAddr{IP: net.ParseIP(ipv4Group), Port: b.port}
			}

			break
		} else {
			b.inAddr = &net.UDPAddr{
				IP: net.ParseIP(ipv6Group),
			}
			b.ipv6Conn.JoinGroup(&iface, b.inAddr)

			// Find IP of the interface
			// TODO(armen): Let user set the ipaddress which here can be verified to be valid
			addrs, err := iface.Addrs()
			if err != nil {
				return nil, err
			}
			ip, _, err := net.ParseCIDR(addrs[0].String())
			if err != nil {
				return nil, err
			}
			b.addr = ip.String()

			if iface.Flags&net.FlagLoopback != 0 {
				b.outAddr = &net.UDPAddr{IP: net.IPv6interfacelocalallnodes, Port: b.port}
			} else {
				b.outAddr = &net.UDPAddr{IP: net.ParseIP(ipv6Group), Port: b.port}
			}
			break
		}
	}

	if b.ipv4Conn == nil && b.ipv6Conn == nil {
		return nil, errors.New("no interfaces to bind to")
	}

	go b.listen()
	go b.signal()

	return b, nil
}

// Terminates the beacon.
func (b *Beacon) Close() {
	b.terminated = true
	if b.signals != nil {
		close(b.signals)
	}

	// Send a nil udp data to wake up listen()
	if b.ipv4Conn != nil {
		b.ipv4Conn.WriteTo(nil, nil, b.outAddr)
	} else {
		b.ipv6Conn.WriteTo(nil, nil, b.outAddr)
	}

	b.wg.Wait()

	if b.ipv4Conn != nil {
		b.ipv4Conn.Close()
	} else {
		b.ipv6Conn.Close()
	}
}

// Returns our own IP address as printable string
func (b *Beacon) Addr() string {
	return b.addr
}

// Port returns port number
func (b *Beacon) Port() int {
	return b.port
}

// SetInterval sets broadcast interval.
func (b *Beacon) SetInterval(interval time.Duration) *Beacon {
	b.interval = interval
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
	if b.interval == 0 {
		b.ticker = time.After(defaultInterval)
	} else {
		b.ticker = time.After(b.interval)
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

// Signals returns Signals channel
func (b *Beacon) Signals() chan *Signal {
	return b.signals
}

func (b *Beacon) listen() {
	b.wg.Add(1)
	defer b.wg.Done()

	var (
		n    int
		addr net.Addr
		err  error
	)

	for {
		buff := make([]byte, beaconMax)
		if b.terminated {
			return
		}
		if b.ipv4Conn != nil {
			n, _, addr, err = b.ipv4Conn.ReadFrom(buff)
			if err != nil || n > beaconMax || n == 0 {
				continue
			}
		} else {
			n, _, addr, err = b.ipv6Conn.ReadFrom(buff)
			if err != nil || n > beaconMax || n == 0 {
				continue
			}
		}

		send := bytes.HasPrefix(buff[:n], b.filter)
		if send && b.noecho {
			send = !bytes.Equal(buff[:n], b.transmit)
		}

		if send && !b.terminated {
			parts := strings.SplitN(addr.String(), ":", 2)
			ipaddr := parts[0]
			select {
			case b.signals <- &Signal{ipaddr, buff[:n]}:
			default:
			}
		}
	}
}

func (b *Beacon) signal() {
	b.wg.Add(1)
	defer b.wg.Done()

	for {
		select {
		case <-b.ticker:
			if b.terminated {
				return
			}
			if b.transmit != nil {
				// Signal other beacons
				if b.ipv4Conn != nil {
					b.ipv4Conn.WriteTo(b.transmit, nil, b.outAddr)
				} else {
					b.ipv6Conn.WriteTo(b.transmit, nil, b.outAddr)
				}
			}
			b.ticker = time.After(b.interval)
		}
	}
}
