package msg

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"

	zmq "github.com/pebbe/zmq4"
)

// Reply to a peer's ping
type PingOk struct {
	routingId []byte
	version   byte
	sequence  uint16
}

// New creates new PingOk message.
func NewPingOk() *PingOk {
	pingok := &PingOk{}
	return pingok
}

// String returns print friendly name.
func (p *PingOk) String() string {
	str := "ZRE_MSG_PINGOK:\n"
	str += fmt.Sprintf("    version = %v\n", p.version)
	str += fmt.Sprintf("    sequence = %v\n", p.sequence)
	return str
}

// Marshal serializes the message.
func (p *PingOk) Marshal() ([]byte, error) {
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
	binary.Write(buffer, binary.BigEndian, PingOkId)

	// version
	value, _ := strconv.ParseUint("2", 10, 1*8)
	binary.Write(buffer, binary.BigEndian, byte(value))

	// sequence
	binary.Write(buffer, binary.BigEndian, p.sequence)

	return buffer.Bytes(), nil
}

// Unmarshals the message.
func (p *PingOk) Unmarshal(frames ...[]byte) error {
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
	if id != PingOkId {
		return errors.New("malformed PingOk message")
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
func (p *PingOk) Send(socket *zmq.Socket) (err error) {
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
func (p *PingOk) RoutingId() []byte {
	return p.routingId
}

// SetRoutingId sets the routingId for this message, routingId should be set
// whenever talking to a ROUTER.
func (p *PingOk) SetRoutingId(routingId []byte) {
	p.routingId = routingId
}

// Setversion sets the version.
func (p *PingOk) SetVersion(version byte) {
	p.version = version
}

// version returns the version.
func (p *PingOk) Version() byte {
	return p.version
}

// Setsequence sets the sequence.
func (p *PingOk) SetSequence(sequence uint16) {
	p.sequence = sequence
}

// sequence returns the sequence.
func (p *PingOk) Sequence() uint16 {
	return p.sequence
}
