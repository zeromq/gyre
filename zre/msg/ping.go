package msg

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"

	zmq "github.com/pebbe/zmq4"
)

// Ping a peer that has gone silent
type Ping struct {
	routingId []byte
	version   byte
	sequence  uint16
}

// New creates new Ping message.
func NewPing() *Ping {
	ping := &Ping{}
	return ping
}

// String returns print friendly name.
func (p *Ping) String() string {
	str := "ZRE_MSG_PING:\n"
	str += fmt.Sprintf("    version = %v\n", p.version)
	str += fmt.Sprintf("    sequence = %v\n", p.sequence)
	return str
}

// Marshal serializes the message.
func (p *Ping) Marshal() ([]byte, error) {
	// Calculate size of serialized data
	bufferSize := 2 + 1 // Signature and message ID

	// version is a 1-byte integer
	bufferSize += 1

	// sequence is a 2-byte integer
	bufferSize += 2

	// Now serialize the message
	tmpBuf := make([]byte, bufferSize)
	tmpBuf = tmpBuf[:0]
	buffer := bytes.NewBuffer(tmpBuf)
	binary.Write(buffer, binary.BigEndian, Signature)
	binary.Write(buffer, binary.BigEndian, PingId)

	// version
	value, _ := strconv.ParseUint("2", 10, 1*8)
	binary.Write(buffer, binary.BigEndian, byte(value))

	// sequence
	binary.Write(buffer, binary.BigEndian, p.sequence)

	return buffer.Bytes(), nil
}

// Unmarshals the message.
func (p *Ping) Unmarshal(frames ...[]byte) error {
	if frames == nil {
		return errors.New("Can't unmarshal empty message")
	}

	frame := frames[0]
	frames = frames[1:]

	buffer := bytes.NewBuffer(frame)

	// Get and check protocol signature
	var signature uint16
	binary.Read(buffer, binary.BigEndian, &signature)
	if signature != Signature {
		return errors.New("invalid signature")
	}

	// Get message id and parse per message type
	var id uint8
	binary.Read(buffer, binary.BigEndian, &id)
	if id != PingId {
		return errors.New("malformed Ping message")
	}
	// version
	binary.Read(buffer, binary.BigEndian, &p.version)
	if p.version != 2 {
		return errors.New("malformed version message")
	}
	// sequence
	binary.Read(buffer, binary.BigEndian, &p.sequence)

	return nil
}

// Sends marshaled data through 0mq socket.
func (p *Ping) Send(socket *zmq.Socket) (err error) {
	frame, err := p.Marshal()
	if err != nil {
		return err
	}

	socType, err := socket.GetType()
	if err != nil {
		return err
	}

	// If we're sending to a ROUTER, we send the routingId first
	if socType == zmq.ROUTER {
		_, err = socket.SendBytes(p.routingId, zmq.SNDMORE)
		if err != nil {
			return err
		}
	}

	// Now send the data frame
	_, err = socket.SendBytes(frame, 0)
	if err != nil {
		return err
	}

	return err
}

// RoutingId returns the routingId for this message, routingId should be set
// whenever talking to a ROUTER.
func (p *Ping) RoutingId() []byte {
	return p.routingId
}

// SetRoutingId sets the routingId for this message, routingId should be set
// whenever talking to a ROUTER.
func (p *Ping) SetRoutingId(routingId []byte) {
	p.routingId = routingId
}

// Setversion sets the version.
func (p *Ping) SetVersion(version byte) {
	p.version = version
}

// version returns the version.
func (p *Ping) Version() byte {
	return p.version
}

// Setsequence sets the sequence.
func (p *Ping) SetSequence(sequence uint16) {
	p.sequence = sequence
}

// sequence returns the sequence.
func (p *Ping) Sequence() uint16 {
	return p.sequence
}
