package msg

import (
	zmq "github.com/vaughan0/go-zmq"

	"bytes"
	"encoding/binary"
	"errors"
)

const (
	ShoutId uint8 = 3
)

// Send a message to a group
type Shout struct {
	address  []byte
	sequence uint16
	Group    string
	Content  []byte
}

// New creates new Shout message
func NewShout() *Shout {
	shout := &Shout{}
	return shout
}

// String returns print friendly name
func (s *Shout) String() string {
	return "SHOUT"
}

// Marshal serializes the message
func (s *Shout) Marshal() ([]byte, error) {
	// Calculate size of serialized data
	bufferSize := 2 + 1 // Signature and message ID

	// Sequence is a 2-byte integer
	bufferSize += 2

	// Group is a string with 1-byte length
	bufferSize++ // Size is one byte
	bufferSize += len(s.Group)

	// Now serialize the message
	b := make([]byte, bufferSize)
	b = b[:0]
	buffer := bytes.NewBuffer(b)
	binary.Write(buffer, binary.BigEndian, Signature)
	binary.Write(buffer, binary.BigEndian, ShoutId)

	// Sequence
	binary.Write(buffer, binary.BigEndian, s.Sequence())

	// Group
	putString(buffer, s.Group)

	return buffer.Bytes(), nil
}

// Unmarshal unserializes the message
func (s *Shout) Unmarshal(frames [][]byte) error {
	frame := frames[0]
	frames = frames[1:]

	buffer := bytes.NewBuffer(frame)

	// Check the signature
	var signature uint16
	binary.Read(buffer, binary.BigEndian, &signature)
	if signature != Signature {
		return errors.New("malformed message")
	}

	var id uint8
	binary.Read(buffer, binary.BigEndian, &id)
	if id != ShoutId {
		return errors.New("malformed message")
	}

	// Sequence
	binary.Read(buffer, binary.BigEndian, &s.sequence)

	// Group
	s.Group = getString(buffer)

	// Content
	s.Content = frames[0]

	return nil
}

// Send sends marshaled data through 0mq socket
func (s *Shout) Send(socket *zmq.Socket) (err error) {
	frame, err := s.Marshal()
	if err != nil {
		return err
	}

	// If we're sending to a ROUTER, we send the address first
	if socket.GetType() == zmq.Router {
		err = socket.SendPart(s.address, true)
		if err != nil {
			return err
		}
	}

	// Now send the data frame
	err = socket.SendPart(frame, true)
	if err != nil {
		return err
	}
	// Now send any frame fields, in order
	err = socket.SendPart(s.Content, false)

	return err
}

// Address returns the address for this message, address should is set
// whenever talking to a ROUTER
func (s *Shout) Address() []byte {
	return s.address
}

// SetAddress sets the address for this message, address should be set
// whenever talking to a ROUTER
func (s *Shout) SetAddress(address []byte) {
	s.address = address
}

// SetSequence sets the sequence
func (s *Shout) SetSequence(sequence uint16) {
	s.sequence = sequence
}

// Sequence returns the sequence
func (s *Shout) Sequence() uint16 {
	return s.sequence
}
