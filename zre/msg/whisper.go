package msg

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"

	zmq "github.com/pebbe/zmq4"
)

// Send a multi-part message to a peer
type Whisper struct {
	routingId []byte
	version   byte
	sequence  uint16
	Content   []byte
}

// New creates new Whisper message.
func NewWhisper() *Whisper {
	whisper := &Whisper{}
	return whisper
}

// String returns print friendly name.
func (w *Whisper) String() string {
	str := "ZRE_MSG_WHISPER:\n"
	str += fmt.Sprintf("    version = %v\n", w.version)
	str += fmt.Sprintf("    sequence = %v\n", w.sequence)
	str += fmt.Sprintf("    Content = %v\n", w.Content)
	return str
}

// Marshal serializes the message.
func (w *Whisper) Marshal() ([]byte, error) {
	// Calculate size of serialized data
	bufferSize := 2 + 1 // Signature and message ID

	// version is a 1-byte integer
	bufferSize += 1

	// sequence is a 2-byte integer
	bufferSize += 2

	// Content is a block of []byte with one byte length
	bufferSize += 1 + len(w.Content)

	// Now serialize the message
	tmpBuf := make([]byte, bufferSize)
	tmpBuf = tmpBuf[:0]
	buffer := bytes.NewBuffer(tmpBuf)
	binary.Write(buffer, binary.BigEndian, Signature)
	binary.Write(buffer, binary.BigEndian, WhisperId)

	// version
	value, _ := strconv.ParseUint("2", 10, 1*8)
	binary.Write(buffer, binary.BigEndian, byte(value))

	// sequence
	binary.Write(buffer, binary.BigEndian, w.sequence)

	putBytes(buffer, w.Content)

	return buffer.Bytes(), nil
}

// Unmarshals the message.
func (w *Whisper) Unmarshal(frames ...[]byte) error {
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
	if id != WhisperId {
		return errors.New("malformed Whisper message")
	}
	// version
	binary.Read(buffer, binary.BigEndian, &w.version)
	if w.version != 2 {
		return errors.New("malformed version message")
	}
	// sequence
	binary.Read(buffer, binary.BigEndian, &w.sequence)
	// Content

	w.Content = getBytes(buffer)

	return nil
}

// Sends marshaled data through 0mq socket.
func (w *Whisper) Send(socket *zmq.Socket) (err error) {
	frame, err := w.Marshal()
	if err != nil {
		return err
	}

	socType, err := socket.GetType()
	if err != nil {
		return err
	}

	// If we're sending to a ROUTER, we send the routingId first
	if socType == zmq.ROUTER {
		_, err = socket.SendBytes(w.routingId, zmq.SNDMORE)
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
	_, err = socket.SendBytes(w.Content, 0)

	return err
}

// RoutingId returns the routingId for this message, routingId should be set
// whenever talking to a ROUTER.
func (w *Whisper) RoutingId() []byte {
	return w.routingId
}

// SetRoutingId sets the routingId for this message, routingId should be set
// whenever talking to a ROUTER.
func (w *Whisper) SetRoutingId(routingId []byte) {
	w.routingId = routingId
}

// Setversion sets the version.
func (w *Whisper) SetVersion(version byte) {
	w.version = version
}

// version returns the version.
func (w *Whisper) Version() byte {
	return w.version
}

// Setsequence sets the sequence.
func (w *Whisper) SetSequence(sequence uint16) {
	w.sequence = sequence
}

// sequence returns the sequence.
func (w *Whisper) Sequence() uint16 {
	return w.sequence
}
