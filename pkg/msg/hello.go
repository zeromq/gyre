package msg

import (
	zmq "github.com/vaughan0/go-zmq"

	"bytes"
	"encoding/binary"
	"errors"
)

// Greet a peer so it can connect back to us
type Hello struct {
	address   []byte
	sequence  uint16
	Ipaddress string
	Mailbox   uint16
	Groups    []string
	Status    byte
	Headers   map[string]string
}

// New creates new Hello message
func NewHello() *Hello {
	hello := &Hello{}
	hello.Headers = make(map[string]string)
	return hello
}

// String returns print friendly name
func (h *Hello) String() string {
	return "HELLO"
}

// Marshal serializes the message
func (h *Hello) Marshal() ([]byte, error) {
	// Calculate size of serialized data
	bufferSize := 2 + 1 // Signature and message ID

	// Sequence is a 2-byte integer
	bufferSize += 2

	// Ipaddress is a string with 1-byte length
	bufferSize++ // Size is one byte
	bufferSize += len(h.Ipaddress)

	// Mailbox is a 2-byte integer
	bufferSize += 2

	// Groups is an array of strings
	bufferSize++ // Size is one byte
	// Add up size of list contents
	for _, val := range h.Groups {
		bufferSize += 1 + len(val)
	}

	// Status is a 1-byte integer
	bufferSize += 1

	// Headers is an array of key=value strings
	bufferSize++ // Size is one byte
	for _, val := range h.Headers {
		bufferSize += 1 + len(val)
	}

	// Now serialize the message
	b := make([]byte, bufferSize)
	b = b[:0]
	buffer := bytes.NewBuffer(b)
	binary.Write(buffer, binary.BigEndian, Signature)
	binary.Write(buffer, binary.BigEndian, HelloId)

	// Sequence
	binary.Write(buffer, binary.BigEndian, h.Sequence())

	// Ipaddress
	putString(buffer, h.Ipaddress)

	// Mailbox
	binary.Write(buffer, binary.BigEndian, h.Mailbox)

	// Groups
	binary.Write(buffer, binary.BigEndian, byte(len(h.Groups)))
	for _, val := range h.Groups {
		putString(buffer, val)
	}

	// Status
	binary.Write(buffer, binary.BigEndian, h.Status)

	// Headers
	binary.Write(buffer, binary.BigEndian, byte(len(h.Headers)))
	for key, val := range h.Headers {
		putKeyValString(buffer, key, val)
	}

	return buffer.Bytes(), nil
}

// Unmarshal unserializes the message
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

	// Ipaddress
	h.Ipaddress = getString(buffer)

	// Mailbox
	binary.Read(buffer, binary.BigEndian, &h.Mailbox)

	// Groups
	var groupsSize byte
	binary.Read(buffer, binary.BigEndian, &groupsSize)
	for ; groupsSize != 0; groupsSize-- {
		h.Groups = append(h.Groups, getString(buffer))
	}

	// Status
	binary.Read(buffer, binary.BigEndian, &h.Status)

	// Headers
	var headersSize byte
	binary.Read(buffer, binary.BigEndian, &headersSize)
	for ; headersSize != 0; headersSize-- {
		key, val := getKeyValString(buffer)
		h.Headers[key] = val
	}

	return nil
}

// Send sends marshaled data through 0mq socket
func (h *Hello) Send(socket *zmq.Socket) (err error) {
	frame, err := h.Marshal()
	if err != nil {
		return err
	}

	// If we're sending to a ROUTER, we send the address first
	if socket.GetType() == zmq.Router {
		err = socket.SendPart(h.address, true)
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
func (h *Hello) Address() []byte {
	return h.address
}

// SetAddress sets the address for this message, address should be set
// whenever talking to a ROUTER
func (h *Hello) SetAddress(address []byte) {
	h.address = address
}

// SetSequence sets the sequence
func (h *Hello) SetSequence(sequence uint16) {
	h.sequence = sequence
}

// Sequence returns the sequence
func (h *Hello) Sequence() uint16 {
	return h.sequence
}
