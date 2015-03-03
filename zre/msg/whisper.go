package msg

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"

	zmq "github.com/pebbe/zmq4"
)

// Whisper struct
// Send a multi-part message to a peer
type Whisper struct {
	routingID []byte
	version   byte
	sequence  uint16
	Content   []byte
}

// NewWhisper creates new Whisper message.
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
	bufferSize++

	// sequence is a 2-byte integer
	bufferSize += 2

	// Content is a block of []byte with one byte length
	bufferSize += 1 + len(w.Content)

	// Now serialize the message
	tmpBuf := make([]byte, bufferSize)
	tmpBuf = tmpBuf[:0]
	buffer := bytes.NewBuffer(tmpBuf)
	binary.Write(buffer, binary.BigEndian, Signature)
	binary.Write(buffer, binary.BigEndian, WhisperID)

	// version
	value, _ := strconv.ParseUint("2", 10, 1*8)
	binary.Write(buffer, binary.BigEndian, byte(value))

	// sequence
	binary.Write(buffer, binary.BigEndian, w.sequence)

	putBytes(buffer, w.Content)

	return buffer.Bytes(), nil
}

// Unmarshal unmarshals the message.
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
		return fmt.Errorf("invalid signature %X != %X", Signature, signature)
	}

	// Get message id and parse per message type
	var id uint8
	binary.Read(buffer, binary.BigEndian, &id)
	if id != WhisperID {
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

// Send sends marshaled data through 0mq socket.
func (w *Whisper) Send(socket *zmq.Socket) (err error) {
	frame, err := w.Marshal()
	if err != nil {
		return err
	}

	socType, err := socket.GetType()
	if err != nil {
		return err
	}

	// If we're sending to a ROUTER, we send the routingID first
	if socType == zmq.ROUTER {
		_, err = socket.SendBytes(w.routingID, zmq.SNDMORE)
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

// RoutingID returns the routingID for this message, routingID should be set
// whenever talking to a ROUTER.
func (w *Whisper) RoutingID() []byte {
	return w.routingID
}

// SetRoutingID sets the routingID for this message, routingID should be set
// whenever talking to a ROUTER.
func (w *Whisper) SetRoutingID(routingID []byte) {
	w.routingID = routingID
}

// SetVersion sets the version.
func (w *Whisper) SetVersion(version byte) {
	w.version = version
}

// Version returns the version.
func (w *Whisper) Version() byte {
	return w.version
}

// SetSequence sets the sequence.
func (w *Whisper) SetSequence(sequence uint16) {
	w.sequence = sequence
}

// Sequence returns the sequence.
func (w *Whisper) Sequence() uint16 {
	return w.sequence
}
