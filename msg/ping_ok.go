package msg

import (
	zmq "github.com/pebbe/zmq4"

	"bytes"
	"encoding/binary"
	"errors"
)

// Reply to a peer's ping
type PingOk struct {
	address  []byte
	sequence uint16
}

// New creates new PingOk message.
func NewPingOk() *PingOk {
	pingok := &PingOk{}
	return pingok
}

// String returns print friendly name.
func (p *PingOk) String() string {
	return "PING_OK"
}

// Marshal serializes the message.
func (p *PingOk) Marshal() ([]byte, error) {
	// Calculate size of serialized data
	bufferSize := 2 + 1 // Signature and message ID

	// Sequence is a 2-byte integer
	bufferSize += 2

	// Now serialize the message
	b := make([]byte, bufferSize)
	b = b[:0]
	buffer := bytes.NewBuffer(b)
	binary.Write(buffer, binary.BigEndian, Signature)
	binary.Write(buffer, binary.BigEndian, PingOkId)

	// Sequence
	binary.Write(buffer, binary.BigEndian, p.Sequence())

	return buffer.Bytes(), nil
}

// Unmarshals the message.
func (p *PingOk) Unmarshal(frames ...[]byte) error {
	frame := frames[0]
	frames = frames[1:]

	buffer := bytes.NewBuffer(frame)

	// Check the signature
	var signature uint16
	binary.Read(buffer, binary.BigEndian, &signature)
	if signature != Signature {
		return errors.New("invalid signature")
	}

	var id uint8
	binary.Read(buffer, binary.BigEndian, &id)
	if id != PingOkId {
		return errors.New("malformed PingOk message")
	}

	// Sequence
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

	// If we're sending to a ROUTER, we send the address first
	if socType == zmq.ROUTER {
		_, err = socket.SendBytes(p.address, zmq.SNDMORE)
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

// Address returns the address for this message, address should be set
// whenever talking to a ROUTER.
func (p *PingOk) Address() []byte {
	return p.address
}

// SetAddress sets the address for this message, address should be set
// whenever talking to a ROUTER.
func (p *PingOk) SetAddress(address []byte) {
	p.address = address
}

// SetSequence sets the sequence.
func (p *PingOk) SetSequence(sequence uint16) {
	p.sequence = sequence
}

// Sequence returns the sequence.
func (p *PingOk) Sequence() uint16 {
	return p.sequence
}
