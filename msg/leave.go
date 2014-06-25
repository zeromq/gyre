package msg

import (
	zmq "github.com/pebbe/zmq4"

	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

// Leave a group
type Leave struct {
	address  []byte
	sequence uint16
	Group    string
	Status   byte
}

// New creates new Leave message.
func NewLeave() *Leave {
	leave := &Leave{}
	return leave
}

// String returns print friendly name.
func (l *Leave) String() string {
	str := "MSG_LEAVE:\n"
	str += fmt.Sprintf("    sequence = %v\n", l.sequence)
	str += fmt.Sprintf("    Group = %v\n", l.Group)
	str += fmt.Sprintf("    Status = %v\n", l.Status)
	return str
}

// Marshal serializes the message.
func (l *Leave) Marshal() ([]byte, error) {
	// Calculate size of serialized data
	bufferSize := 2 + 1 // Signature and message ID

	// Sequence is a 2-byte integer
	bufferSize += 2

	// Group is a string with 1-byte length
	bufferSize++ // Size is one byte
	bufferSize += len(l.Group)

	// Status is a 1-byte integer
	bufferSize += 1

	// Now serialize the message
	b := make([]byte, bufferSize)
	b = b[:0]
	buffer := bytes.NewBuffer(b)
	binary.Write(buffer, binary.BigEndian, Signature)
	binary.Write(buffer, binary.BigEndian, LeaveId)

	// Sequence
	binary.Write(buffer, binary.BigEndian, l.Sequence())

	// Group
	putString(buffer, l.Group)

	// Status
	binary.Write(buffer, binary.BigEndian, l.Status)

	return buffer.Bytes(), nil
}

// Unmarshals the message.
func (l *Leave) Unmarshal(frames ...[]byte) error {
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
	if id != LeaveId {
		return errors.New("malformed Leave message")
	}

	// Sequence
	binary.Read(buffer, binary.BigEndian, &l.sequence)

	// Group
	l.Group = getString(buffer)

	// Status
	binary.Read(buffer, binary.BigEndian, &l.Status)

	return nil
}

// Sends marshaled data through 0mq socket.
func (l *Leave) Send(socket *zmq.Socket) (err error) {
	frame, err := l.Marshal()
	if err != nil {
		return err
	}

	socType, err := socket.GetType()
	if err != nil {
		return err
	}

	// If we're sending to a ROUTER, we send the address first
	if socType == zmq.ROUTER {
		_, err = socket.SendBytes(l.address, zmq.SNDMORE)
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
func (l *Leave) Address() []byte {
	return l.address
}

// SetAddress sets the address for this message, address should be set
// whenever talking to a ROUTER.
func (l *Leave) SetAddress(address []byte) {
	l.address = address
}

// SetSequence sets the sequence.
func (l *Leave) SetSequence(sequence uint16) {
	l.sequence = sequence
}

// Sequence returns the sequence.
func (l *Leave) Sequence() uint16 {
	return l.sequence
}
