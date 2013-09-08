package msg

import (
	zmq "github.com/armen/go-zmq"

	"bytes"
	"encoding/binary"
	"errors"
)

const (
	JoinId uint8 = 4
)

// Join a group
type Join struct {
	address  []byte
	sequence uint16
	Group    string
	Status   byte
}

// New creates new Join message
func NewJoin() *Join {
	join := &Join{}
	return join
}

// String returns print friendly name
func (j *Join) String() string {
	return "JOIN"
}

// Marshal serializes the message
func (j *Join) Marshal() ([]byte, error) {
	// Calculate size of serialized data
	bufferSize := 2 + 1 // Signature and message ID

	// Sequence is a 2-byte integer
	bufferSize += 2

	// Group is a string with 1-byte length
	bufferSize++ // Size is one byte
	bufferSize += len(j.Group)

	// Status is a 1-byte integer
	bufferSize += 1

	// Now serialize the message
	b := make([]byte, bufferSize)
	b = b[:0]
	buffer := bytes.NewBuffer(b)
	binary.Write(buffer, binary.BigEndian, Signature)
	binary.Write(buffer, binary.BigEndian, JoinId)

	// Sequence
	binary.Write(buffer, binary.BigEndian, j.Sequence())

	// Group
	putString(buffer, j.Group)

	// Status
	binary.Write(buffer, binary.BigEndian, j.Status)

	return buffer.Bytes(), nil
}

// Unmarshal unserializes the message
func (j *Join) Unmarshal(frames [][]byte) error {
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
	if id != JoinId {
		return errors.New("malformed message")
	}

	// Sequence
	binary.Read(buffer, binary.BigEndian, &j.sequence)

	// Group
	j.Group = getString(buffer)

	// Status
	binary.Read(buffer, binary.BigEndian, &j.Status)

	return nil
}

// Send sends marshaled data through 0mq socket
func (j *Join) Send(socket *zmq.Socket) (err error) {
	frame, err := j.Marshal()
	if err != nil {
		return err
	}

	// If we're sending to a ROUTER, we send the address first
	if socket.GetType() == zmq.Router {
		err = socket.SendPart(j.address, true)
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
func (j *Join) Address() []byte {
	return j.address
}

// SetAddress sets the address for this message, address should be set
// whenever talking to a ROUTER
func (j *Join) SetAddress(address []byte) {
	j.address = address
}

// SetSequence sets the sequence
func (j *Join) SetSequence(sequence uint16) {
	j.sequence = sequence
}

// Sequence returns the sequence
func (j *Join) Sequence() uint16 {
	return j.sequence
}
