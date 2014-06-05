package gyre

import (
	"github.com/armen/gyre/msg"
	zmq "github.com/vaughan0/go-zmq"

	"fmt"
	"time"
)

var (
	peerEvasive  = 5 * time.Second  // Five seconds' silence is evasive
	peerExpired  = 10 * time.Second // Ten seconds' silence is expired
	reapInterval = 1 * time.Second  //  Once per second
)

type Peer struct {
	mailbox      *zmq.Socket // Socket through to peer
	Identity     string
	Endpoint     string            // Endpoint connected to
	EvasiveAt    time.Time         // Peer is being evasive
	ExpiredAt    time.Time         // Peer has expired by now
	Connected    bool              // Peer will send messages
	Ready        bool              // Peer has said Hello to us
	Status       byte              // Our status counter
	SentSequence uint16            // Outgoing message sequence
	WantSequence uint16            // Incoming message sequence
	Headers      map[string]string // Peer headers
}

// NewPeer creates a new peer
func NewPeer(identity string) (peer *Peer) {
	peer = &Peer{
		Identity: identity,
		Headers:  make(map[string]string),
	}
	peer.Refresh()
	return
}

// Connect configures mailbox and connects to peer's router endpoint
func (p *Peer) Connect(replyTo, endpoint string) (err error) {
	// Create new outgoing socket (drop any messages in transit)
	p.mailbox, err = zmq.NewSocket(zmq.Dealer)
	if err != nil {
		return err
	}

	// Set our caller 'From' identity so that receiving node knows
	// who each message came from.
	p.mailbox.SetIdentitiy([]byte(replyTo))

	// Set a high-water mark that allows for reasonable activity
	// TODO(armen): set SetSendHWM correctly
	// p.mailbox.SetSendHWM(uint64(peerExpired * 100000))

	// Send messages immediately or return EAGAIN
	p.mailbox.SetSendTimeout(0)

	// Connect through to peer node
	err = p.mailbox.Connect(fmt.Sprintf("tcp://%s", endpoint))
	if err != nil {
		return err
	}
	p.Endpoint = endpoint
	p.Connected = true
	p.Ready = false

	return nil
}

// Disconnect disconnects peer mailbox. No more messages will be sent to peer until connected again
func (p *Peer) Disconnect() {
	if p.Connected {
		p.Connected = false
		if p.mailbox != nil {
			p.mailbox.Disconnect(p.Endpoint)
			p.mailbox.Close()
			p.mailbox = nil
		}
		p.Endpoint = ""
	}
}

// Send sends message to peer
func (p *Peer) Send(t msg.Transit) {
	if p.Connected {
		p.SentSequence++
		t.SetSequence(p.SentSequence)
		err := t.Send(p.mailbox)
		if err != nil {
			p.Disconnect()
		}
	}
}

// CheckMessage checks peer message sequence
func (p *Peer) CheckMessage(t msg.Transit) bool {
	p.WantSequence++
	valid := p.WantSequence == t.Sequence()
	if !valid {
		p.WantSequence--
	}

	return valid
}

// Refresh refreshes activity at peer
func (p *Peer) Refresh() {
	p.EvasiveAt = time.Now().Add(peerEvasive)
	p.ExpiredAt = time.Now().Add(peerExpired)
}

// SetExpired sets expired.
func SetExpired(expired time.Duration) {
	peerExpired = expired
}

// SetEvasive sets evasive.
func SetEvasive(evasive time.Duration) {
	peerEvasive = evasive
}

// SetPingInterval sets interval of pinging other peers
func SetPingInterval(interval time.Duration) {
	reapInterval = interval
}
