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
	"os"
	"strconv"
	"sync"
	"time"

	"code.google.com/p/go.net/ipv4"
	"code.google.com/p/go.net/ipv6"
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
	iface      string
	wg         sync.WaitGroup
	inAddr     *net.UDPAddr
	outAddr    *net.UDPAddr
	sync.Mutex
}

// Creates a new beacon on a certain UDP port.
func New() (b *Beacon) {

	b = &Beacon{
		signals:  make(chan *Signal, 50),
		interval: defaultInterval,
	}

	return b
}

func (b *Beacon) start() (err error) {

	if b.iface == "" {
		b.iface = os.Getenv("BEACON_INTERFACE")
	}
	if b.iface == "" {
		b.iface = os.Getenv("ZSYS_INTERFACE")
	}

	var ifs []net.Interface

	if b.iface == "" {
		ifs, err = net.Interfaces()
		if err != nil {
			return err
		}

	} else {
		iface, err := net.InterfaceByName(b.iface)
		if err != nil {
			return err
		}
		ifs = append(ifs, *iface)
	}

	conn, err := net.ListenPacket("udp4", net.JoinHostPort("224.0.0.0", strconv.Itoa(b.port)))
	if err == nil {
		b.ipv4Conn = ipv4.NewPacketConn(conn)
		b.ipv4Conn.SetMulticastLoopback(true)
		b.ipv4Conn.SetControlMessage(ipv4.FlagSrc, true)
	}

	if !b.ipv4 {
		conn, err := net.ListenPacket("udp6", net.JoinHostPort(net.IPv6linklocalallnodes.String(), strconv.Itoa(b.port)))
		if err != nil {
			return err
		}

		b.ipv6Conn = ipv6.NewPacketConn(conn)
		b.ipv6Conn.SetMulticastLoopback(true)
		b.ipv6Conn.SetControlMessage(ipv6.FlagSrc, true)
	}

	broadcast := os.Getenv("BEACON_BROADCAST") != ""

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
				return err
			}

			if len(addrs) <= 0 {
				return errors.New("no address to bind to")
			}

			ip, ipnet, err := net.ParseCIDR(addrs[0].String())
			if err != nil {
				return err
			}
			b.addr = ip.String()

			switch {
			case broadcast:
				bcast := ipnet.IP
				for i := 0; i < len(ipnet.Mask); i++ {
					bcast[i] |= ^ipnet.Mask[i]
				}
				b.outAddr = &net.UDPAddr{IP: bcast, Port: b.port}

			case iface.Flags&net.FlagLoopback != 0:
				b.outAddr = &net.UDPAddr{IP: net.IPv4allsys, Port: b.port}

			default:
				b.outAddr = &net.UDPAddr{IP: net.ParseIP(ipv4Group), Port: b.port}
			}

			break
		} else if b.ipv6Conn != nil {
			b.inAddr = &net.UDPAddr{
				IP: net.ParseIP(ipv6Group),
			}
			b.ipv6Conn.JoinGroup(&iface, b.inAddr)

			// Find IP of the interface
			// TODO(armen): Let user set the ipaddress which here can be verified to be valid
			addrs, err := iface.Addrs()
			if err != nil {
				return err
			}
			ip, ipnet, err := net.ParseCIDR(addrs[0].String())
			if err != nil {
				return err
			}
			b.addr = ip.String()

			switch {
			case broadcast:
				bcast := ipnet.IP
				for i := 0; i < len(ipnet.Mask); i++ {
					bcast[i] |= ^ipnet.Mask[i]
				}
				b.outAddr = &net.UDPAddr{IP: bcast, Port: b.port}

			case iface.Flags&net.FlagLoopback != 0:
				b.outAddr = &net.UDPAddr{IP: net.IPv6interfacelocalallnodes, Port: b.port}

			default:
				b.outAddr = &net.UDPAddr{IP: net.ParseIP(ipv6Group), Port: b.port}
			}
			break
		}
	}

	if b.ipv4Conn == nil && b.ipv6Conn == nil {
		return errors.New("no interfaces to bind to")
	}

	go b.listen()
	go b.signal()

	return nil
}

// Terminates the beacon.
func (b *Beacon) Close() {
	b.Lock()
	b.terminated = true

	if b.signals != nil {
		close(b.signals)
	}
	b.Unlock()

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

// SetInterface sets interface to bind and listen on.
func (b *Beacon) SetInterface(iface string) *Beacon {
	b.iface = iface
	return b
}

// SetPort sets UDP port.
func (b *Beacon) SetPort(port int) *Beacon {
	b.port = port
	return b
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
func (b *Beacon) Publish(transmit []byte) error {
	b.Lock()
	defer b.Unlock()
	b.transmit = transmit

	err := b.start()

	return err
}

// Silence stops broadcasting beacon.
func (b *Beacon) Silence() *Beacon {
	b.Lock()
	defer b.Unlock()

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
		addr net.IP
		err  error
	)

	for {
		buff := make([]byte, beaconMax)

		b.Lock()
		if b.terminated {
			b.Unlock()
			return
		}
		b.Unlock()

		if b.ipv4Conn != nil {
			var cm *ipv4.ControlMessage
			n, cm, _, err = b.ipv4Conn.ReadFrom(buff)
			if err != nil || n > beaconMax || n == 0 {
				continue
			}
			addr = cm.Src
		} else {
			var cm *ipv6.ControlMessage
			n, cm, _, err = b.ipv6Conn.ReadFrom(buff)
			if err != nil || n > beaconMax || n == 0 {
				continue
			}
			addr = cm.Src
		}

		send := bytes.HasPrefix(buff[:n], b.filter)
		if send && b.noecho {
			send = !bytes.Equal(buff[:n], b.transmit)
		}

		if send && !b.terminated {
			select {
			case b.signals <- &Signal{addr.String(), buff[:n]}:
			default:
			}
		}
	}
}

func (b *Beacon) signal() {
	b.wg.Add(1)
	defer b.wg.Done()

	var ticker <-chan time.Time

	if b.interval == 0 {
		ticker = time.After(defaultInterval)
	} else {
		ticker = time.After(b.interval)
	}

	for {
		select {
		case <-ticker:
			b.Lock()
			if b.terminated {
				b.Unlock()
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
			b.Unlock()

			ticker = time.After(b.interval)
		}
	}
}
