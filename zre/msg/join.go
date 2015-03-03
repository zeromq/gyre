package msg

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"

	zmq "github.com/pebbe/zmq4"
)

// Join struct
// Join a group
type Join struct {
	routingID []byte
	version   byte
	sequence  uint16
	Group     string
	Status    byte
}

// NewJoin creates new Join message.
func NewJoin() *Join {
	join := &Join{}
	return join
}

// String returns print friendly name.
func (j *Join) String() string {
	str := "ZRE_MSG_JOIN:\n"
	str += fmt.Sprintf("    version = %v\n", j.version)
	str += fmt.Sprintf("    sequence = %v\n", j.sequence)
	str += fmt.Sprintf("    Group = %v\n", j.Group)
	str += fmt.Sprintf("    Status = %v\n", j.Status)
	return str
}

// Marshal serializes the message.
func (j *Join) Marshal() ([]byte, error) {
	// Calculate size of serialized data
	bufferSize := 2 + 1 // Signature and message ID

	// version is a 1-byte integer
	bufferSize++

	// sequence is a 2-byte integer
	bufferSize += 2

	// Group is a string with 1-byte length
	bufferSize++ // Size is one byte
	bufferSize += len(j.Group)

	// Status is a 1-byte integer
	bufferSize++

	// Now serialize the message
	tmpBuf := make([]byte, bufferSize)
	tmpBuf = tmpBuf[:0]
	buffer := bytes.NewBuffer(tmpBuf)
	binary.Write(buffer, binary.BigEndian, Signature)
	binary.Write(buffer, binary.BigEndian, JoinID)

	// version
	value, _ := strconv.ParseUint("2", 10, 1*8)
	binary.Write(buffer, binary.BigEndian, byte(value))

	// sequence
	binary.Write(buffer, binary.BigEndian, j.sequence)

	// Group
	putString(buffer, j.Group)

	// Status
	binary.Write(buffer, binary.BigEndian, j.Status)

	return buffer.Bytes(), nil
}

// Unmarshal unmarshals the message.
func (j *Join) Unmarshal(frames ...[]byte) error {
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
	if id != JoinID {
		return errors.New("malformed Join message")
	}
	// version
	binary.Read(buffer, binary.BigEndian, &j.version)
	if j.version != 2 {
		return errors.New("malformed version message")
	}
	// sequence
	binary.Read(buffer, binary.BigEndian, &j.sequence)
	// Group
	j.Group = getString(buffer)
	// Status
	binary.Read(buffer, binary.BigEndian, &j.Status)

	return nil
}

// Send sends marshaled data through 0mq socket.
func (j *Join) Send(socket *zmq.Socket) (err error) {
	frame, err := j.Marshal()
	if err != nil {
		return err
	}

	socType, err := socket.GetType()
	if err != nil {
		return err
	}

	// If we're sending to a ROUTER, we send the routingID first
	if socType == zmq.ROUTER {
		_, err = socket.SendBytes(j.routingID, zmq.SNDMORE)
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
func (j *Join) RoutingID() []byte {
	return j.routingID
}

// SetRoutingID sets the routingID for this message, routingID should be set
// whenever talking to a ROUTER.
func (j *Join) SetRoutingID(routingID []byte) {
	j.routingID = routingID
}

// SetVersion sets the version.
func (j *Join) SetVersion(version byte) {
	j.version = version
}

// Version returns the version.
func (j *Join) Version() byte {
	return j.version
}

// SetSequence sets the sequence.
func (j *Join) SetSequence(sequence uint16) {
	j.sequence = sequence
}

// Sequence returns the sequence.
func (j *Join) Sequence() uint16 {
	return j.sequence
}
