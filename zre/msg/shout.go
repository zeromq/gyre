package msg

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"

	zmq "github.com/pebbe/zmq4"
)

// Shout struct
// Send a multi-part message to a group
type Shout struct {
	routingID []byte
	version   byte
	sequence  uint16
	Group     string
	Content   []byte
}

// NewShout creates new Shout message.
func NewShout() *Shout {
	shout := &Shout{}
	return shout
}

// String returns print friendly name.
func (s *Shout) String() string {
	str := "ZRE_MSG_SHOUT:\n"
	str += fmt.Sprintf("    version = %v\n", s.version)
	str += fmt.Sprintf("    sequence = %v\n", s.sequence)
	str += fmt.Sprintf("    Group = %v\n", s.Group)
	str += fmt.Sprintf("    Content = %v\n", s.Content)
	return str
}

// Marshal serializes the message.
func (s *Shout) Marshal() ([]byte, error) {
	// Calculate size of serialized data
	bufferSize := 2 + 1 // Signature and message ID

	// version is a 1-byte integer
	bufferSize++

	// sequence is a 2-byte integer
	bufferSize += 2

	// Group is a string with 1-byte length
	bufferSize++ // Size is one byte
	bufferSize += len(s.Group)

	// Content is a block of []byte with one byte length
	bufferSize += 1 + len(s.Content)

	// Now serialize the message
	tmpBuf := make([]byte, bufferSize)
	tmpBuf = tmpBuf[:0]
	buffer := bytes.NewBuffer(tmpBuf)
	binary.Write(buffer, binary.BigEndian, Signature)
	binary.Write(buffer, binary.BigEndian, ShoutID)

	// version
	value, _ := strconv.ParseUint("2", 10, 1*8)
	binary.Write(buffer, binary.BigEndian, byte(value))

	// sequence
	binary.Write(buffer, binary.BigEndian, s.sequence)

	// Group
	putString(buffer, s.Group)

	putBytes(buffer, s.Content)

	return buffer.Bytes(), nil
}

// Unmarshal unmarshals the message.
func (s *Shout) Unmarshal(frames ...[]byte) error {
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
	if id != ShoutID {
		return errors.New("malformed Shout message")
	}
	// version
	binary.Read(buffer, binary.BigEndian, &s.version)
	if s.version != 2 {
		return errors.New("malformed version message")
	}
	// sequence
	binary.Read(buffer, binary.BigEndian, &s.sequence)
	// Group
	s.Group = getString(buffer)
	// Content

	s.Content = getBytes(buffer)

	return nil
}

// Send sends marshaled data through 0mq socket.
func (s *Shout) Send(socket *zmq.Socket) (err error) {
	frame, err := s.Marshal()
	if err != nil {
		return err
	}

	socType, err := socket.GetType()
	if err != nil {
		return err
	}

	// If we're sending to a ROUTER, we send the routingID first
	if socType == zmq.ROUTER {
		_, err = socket.SendBytes(s.routingID, zmq.SNDMORE)
		if err != nil {
			return err
		}
	}

	// Now send the data frame
	_, err = socket.SendBytes(frame, zmq.SNDMORE)
	if err != nil {
		return err
	}
	// Now send any frame fields, in order
	_, err = socket.SendBytes(s.Content, 0)

	return err
}

// RoutingID returns the routingID for this message, routingID should be set
// whenever talking to a ROUTER.
func (s *Shout) RoutingID() []byte {
	return s.routingID
}

// SetRoutingID sets the routingID for this message, routingID should be set
// whenever talking to a ROUTER.
func (s *Shout) SetRoutingID(routingID []byte) {
	s.routingID = routingID
}

// SetVersion sets the version.
func (s *Shout) SetVersion(version byte) {
	s.version = version
}

// Version returns the version.
func (s *Shout) Version() byte {
	return s.version
}

// SetSequence sets the sequence.
func (s *Shout) SetSequence(sequence uint16) {
	s.sequence = sequence
}

// Sequence returns the sequence.
func (s *Shout) Sequence() uint16 {
	return s.sequence
}
