package msg

import (
	zmq "github.com/armen/go-zmq"

	"bytes"
	"encoding/binary"
	"errors"
)

// Leave a group
type Leave struct {
	address  []byte
	sequence uint16
	Group    string
	Status   byte
}

// New creates new Leave message
func NewLeave() *Leave {
	leave := &Leave{}
	return leave
}

// String returns print friendly name
func (l *Leave) String() string {
	return "LEAVE"
}

// Marshal serializes the message
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

// Unmarshal unserializes the message
func (l *Leave) Unmarshal(frames ...[]byte) error {
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
	if id != LeaveId {
		return errors.New("malformed message")
	}

	// Sequence
	binary.Read(buffer, binary.BigEndian, &l.sequence)

	// Group
	l.Group = getString(buffer)

	// Status
	binary.Read(buffer, binary.BigEndian, &l.Status)

	return nil
}

// Send sends marshaled data through 0mq socket
func (l *Leave) Send(socket *zmq.Socket) (err error) {
	frame, err := l.Marshal()
	if err != nil {
		return err
	}

	// If we're sending to a ROUTER, we send the address first
	if socket.GetType() == zmq.Router {
		err = socket.SendPart(l.address, true)
		if err != nil {
			return err
		}
	}

	// Now send the data frame
	err = socket.SendPart(frame, false)
	if err != nil {
		return err
	}

	return err
}

// Address returns the address for this message, address should is set
// whenever talking to a ROUTER
func (l *Leave) Address() []byte {
	return l.address
}

// SetAddress sets the address for this message, address should be set
// whenever talking to a ROUTER
func (l *Leave) SetAddress(address []byte) {
	l.address = address
}

// SetSequence sets the sequence
func (l *Leave) SetSequence(sequence uint16) {
	l.sequence = sequence
}

// Sequence returns the sequence
func (l *Leave) Sequence() uint16 {
	return l.sequence
}
