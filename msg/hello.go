package msg

import (
	zmq "github.com/pebbe/zmq4"

	"bytes"
	"encoding/binary"
	"errors"
)

// Greet a peer so it can connect back to us
type Hello struct {
	address  string
	sequence uint16
	Endpoint string
	Groups   []string
	Status   byte
	Name     string
	Headers  map[string]string
}

// New creates new Hello message.
func NewHello() *Hello {
	hello := &Hello{}
	hello.Headers = make(map[string]string)
	return hello
}

// String returns print friendly name.
func (h *Hello) String() string {
	return "HELLO"
}

// Marshal serializes the message.
func (h *Hello) Marshal() ([]byte, error) {
	// Calculate size of serialized data
	bufferSize := 2 + 1 // Signature and message ID

	// Sequence is a 2-byte integer
	bufferSize += 2

	// Endpoint is a string with 1-byte length
	bufferSize++ // Size is one byte
	bufferSize += len(h.Endpoint)

	// Groups is an array of strings
	bufferSize++ // Size is one byte
	// Add up size of list contents
	for _, val := range h.Groups {
		bufferSize += 1 + len(val)
	}

	// Status is a 1-byte integer
	bufferSize += 1

	// Name is a string with 1-byte length
	bufferSize++ // Size is one byte
	bufferSize += len(h.Name)

	// Headers is a hash table
	bufferSize += 4 // Size is 4 bytes
	for key, val := range h.Headers {
		bufferSize += 1 + len(key)
		bufferSize += 4 + len(val)
	}

	// Now serialize the message
	b := make([]byte, bufferSize)
	b = b[:0]
	buffer := bytes.NewBuffer(b)
	binary.Write(buffer, binary.BigEndian, Signature)
	binary.Write(buffer, binary.BigEndian, HelloId)

	// Sequence
	binary.Write(buffer, binary.BigEndian, h.Sequence())

	// Endpoint
	putString(buffer, h.Endpoint)

	// Groups
	binary.Write(buffer, binary.BigEndian, byte(len(h.Groups)))
	for _, val := range h.Groups {
		putString(buffer, val)
	}

	// Status
	binary.Write(buffer, binary.BigEndian, h.Status)

	// Name
	putString(buffer, h.Name)

	// Headers
	binary.Write(buffer, binary.BigEndian, uint32(len(h.Headers)))
	for key, val := range h.Headers {
		putString(buffer, key)
		putLongString(buffer, val)
	}

	return buffer.Bytes(), nil
}

// Unmarshals the message.
func (h *Hello) Unmarshal(frames ...[]byte) error {
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
	if id != HelloId {
		return errors.New("malformed Hello message")
	}

	// Sequence
	binary.Read(buffer, binary.BigEndian, &h.sequence)

	// Endpoint
	h.Endpoint = getString(buffer)

	// Groups
	var groupsSize byte
	binary.Read(buffer, binary.BigEndian, &groupsSize)
	for ; groupsSize != 0; groupsSize-- {
		h.Groups = append(h.Groups, getString(buffer))
	}

	// Status
	binary.Read(buffer, binary.BigEndian, &h.Status)

	// Name
	h.Name = getString(buffer)

	// Headers
	var headersSize uint32
	binary.Read(buffer, binary.BigEndian, &headersSize)
	for ; headersSize != 0; headersSize-- {
		key := getString(buffer)
		val := getLongString(buffer)
		h.Headers[key] = val
	}

	return nil
}

// Sends marshaled data through 0mq socket.
func (h *Hello) Send(socket *zmq.Socket) (err error) {
	frame, err := h.Marshal()
	if err != nil {
		return err
	}

	socType, err := socket.GetType()
	if err != nil {
		return err
	}

	// If we're sending to a ROUTER, we send the address first
	if socType == zmq.ROUTER {
		_, err = socket.Send(h.address, zmq.SNDMORE)
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

// Address returns the address for this message, address should is set
// whenever talking to a ROUTER.
func (h *Hello) Address() string {
	return h.address
}

// SetAddress sets the address for this message, address should be set
// whenever talking to a ROUTER.
func (h *Hello) SetAddress(address string) {
	h.address = address
}

// SetSequence sets the sequence.
func (h *Hello) SetSequence(sequence uint16) {
	h.sequence = sequence
}

// Sequence returns the sequence.
func (h *Hello) Sequence() uint16 {
	return h.sequence
}
