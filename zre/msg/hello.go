package msg

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"

	zmq "github.com/pebbe/zmq4"
)

// Greet a peer so it can connect back to us
type Hello struct {
	routingId []byte
	version   byte
	sequence  uint16
	Endpoint  string
	Groups    []string
	Status    byte
	Name      string
	Headers   map[string]string
}

// New creates new Hello message.
func NewHello() *Hello {
	hello := &Hello{}
	hello.Headers = make(map[string]string)
	return hello
}

// String returns print friendly name.
func (h *Hello) String() string {
	str := "ZRE_MSG_HELLO:\n"
	str += fmt.Sprintf("    version = %v\n", h.version)
	str += fmt.Sprintf("    sequence = %v\n", h.sequence)
	str += fmt.Sprintf("    Endpoint = %v\n", h.Endpoint)
	str += fmt.Sprintf("    Groups = %v\n", h.Groups)
	str += fmt.Sprintf("    Status = %v\n", h.Status)
	str += fmt.Sprintf("    Name = %v\n", h.Name)
	str += fmt.Sprintf("    Headers = %v\n", h.Headers)
	return str
}

// Marshal serializes the message.
func (h *Hello) Marshal() ([]byte, error) {
	// Calculate size of serialized data
	bufferSize := 2 + 1 // Signature and message ID

	// version is a 1-byte integer
	bufferSize += 1

	// sequence is a 2-byte integer
	bufferSize += 2

	// Endpoint is a string with 1-byte length
	bufferSize++ // Size is one byte
	bufferSize += len(h.Endpoint)

	// Groups is an array of strings
	bufferSize += 4 // Size is 4 bytes
	// Add up size of string contents
	for _, val := range h.Groups {
		bufferSize += 4 + len(val)
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
	tmpBuf := make([]byte, bufferSize)
	tmpBuf = tmpBuf[:0]
	buffer := bytes.NewBuffer(tmpBuf)
	binary.Write(buffer, binary.BigEndian, Signature)
	binary.Write(buffer, binary.BigEndian, HelloId)

	// version
	value, _ := strconv.ParseUint("2", 10, 1*8)
	binary.Write(buffer, binary.BigEndian, byte(value))

	// sequence
	binary.Write(buffer, binary.BigEndian, h.sequence)

	// Endpoint
	putString(buffer, h.Endpoint)

	// Groups
	binary.Write(buffer, binary.BigEndian, uint32(len(h.Groups)))
	for _, val := range h.Groups {
		putLongString(buffer, val)
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
		return errors.New("invalid signature")
	}

	// Get message id and parse per message type
	var id uint8
	binary.Read(buffer, binary.BigEndian, &id)
	if id != HelloId {
		return errors.New("malformed Hello message")
	}
	// version
	binary.Read(buffer, binary.BigEndian, &h.version)
	if h.version != 2 {
		return errors.New("malformed version message")
	}
	// sequence
	binary.Read(buffer, binary.BigEndian, &h.sequence)
	// Endpoint
	h.Endpoint = getString(buffer)
	// Groups
	var groupsSize uint32
	binary.Read(buffer, binary.BigEndian, &groupsSize)
	for ; groupsSize != 0; groupsSize-- {
		h.Groups = append(h.Groups, getLongString(buffer))
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

	// If we're sending to a ROUTER, we send the routingId first
	if socType == zmq.ROUTER {
		_, err = socket.SendBytes(h.routingId, zmq.SNDMORE)
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

// RoutingId returns the routingId for this message, routingId should be set
// whenever talking to a ROUTER.
func (h *Hello) RoutingId() []byte {
	return h.routingId
}

// SetRoutingId sets the routingId for this message, routingId should be set
// whenever talking to a ROUTER.
func (h *Hello) SetRoutingId(routingId []byte) {
	h.routingId = routingId
}

// Setversion sets the version.
func (h *Hello) SetVersion(version byte) {
	h.version = version
}

// version returns the version.
func (h *Hello) Version() byte {
	return h.version
}

// Setsequence sets the sequence.
func (h *Hello) SetSequence(sequence uint16) {
	h.sequence = sequence
}

// sequence returns the sequence.
func (h *Hello) Sequence() uint16 {
	return h.sequence
}
