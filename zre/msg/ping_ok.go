package msg

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"

	zmq "github.com/pebbe/zmq4"
)

// PingOk struct
// Reply to a peer's ping
type PingOk struct {
	routingID []byte
	version   byte
	sequence  uint16
}

// NewPingOk creates new PingOk message.
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
	bufferSize++

	// sequence is a 2-byte integer
	bufferSize += 2

	// Now serialize the message
	tmpBuf := make([]byte, bufferSize)
	tmpBuf = tmpBuf[:0]
	buffer := bytes.NewBuffer(tmpBuf)
	binary.Write(buffer, binary.BigEndian, Signature)
	binary.Write(buffer, binary.BigEndian, PingOkID)

	// version
	value, _ := strconv.ParseUint("2", 10, 1*8)
	binary.Write(buffer, binary.BigEndian, byte(value))

	// sequence
	binary.Write(buffer, binary.BigEndian, p.sequence)

	return buffer.Bytes(), nil
}

// Unmarshal unmarshals the message.
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
		return fmt.Errorf("invalid signature %X != %X", Signature, signature)
	}

	// Get message id and parse per message type
	var id uint8
	binary.Read(buffer, binary.BigEndian, &id)
	if id != PingOkID {
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

// Send sends marshaled data through 0mq socket.
func (p *PingOk) Send(socket *zmq.Socket) (err error) {
	frame, err := p.Marshal()
	if err != nil {
		return err
	}

	socType, err := socket.GetType()
	if err != nil {
		return err
	}

	// If we're sending to a ROUTER, we send the routingID first
	if socType == zmq.ROUTER {
		_, err = socket.SendBytes(p.routingID, zmq.SNDMORE)
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

// RoutingID returns the routingID for this message, routingID should be set
// whenever talking to a ROUTER.
func (p *PingOk) RoutingID() []byte {
	return p.routingID
}

// SetRoutingID sets the routingID for this message, routingID should be set
// whenever talking to a ROUTER.
func (p *PingOk) SetRoutingID(routingID []byte) {
	p.routingID = routingID
}

// SetVersion sets the version.
func (p *PingOk) SetVersion(version byte) {
	p.version = version
}

// Version returns the version.
func (p *PingOk) Version() byte {
	return p.version
}

// SetSequence sets the sequence.
func (p *PingOk) SetSequence(sequence uint16) {
	p.sequence = sequence
}

// Sequence returns the sequence.
func (p *PingOk) Sequence() uint16 {
	return p.sequence
}
