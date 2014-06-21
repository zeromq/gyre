package msg

import (
	zmq "github.com/pebbe/zmq4"

	"bytes"
	"encoding/binary"
	"errors"
)

// Send a multi-part message to a peer
type Whisper struct {
	address  []byte
	sequence uint16
	Content  []byte
}

// New creates new Whisper message.
func NewWhisper() *Whisper {
	whisper := &Whisper{}
	return whisper
}

// String returns print friendly name.
func (w *Whisper) String() string {
	return "WHISPER"
}

// Marshal serializes the message.
func (w *Whisper) Marshal() ([]byte, error) {
	// Calculate size of serialized data
	bufferSize := 2 + 1 // Signature and message ID

	// Sequence is a 2-byte integer
	bufferSize += 2

	// Now serialize the message
	b := make([]byte, bufferSize)
	b = b[:0]
	buffer := bytes.NewBuffer(b)
	binary.Write(buffer, binary.BigEndian, Signature)
	binary.Write(buffer, binary.BigEndian, WhisperId)

	// Sequence
	binary.Write(buffer, binary.BigEndian, w.Sequence())

	return buffer.Bytes(), nil
}

// Unmarshals the message.
func (w *Whisper) Unmarshal(frames ...[]byte) error {
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
	if id != WhisperId {
		return errors.New("malformed Whisper message")
	}

	// Sequence
	binary.Read(buffer, binary.BigEndian, &w.sequence)

	// Content
	if 0 <= len(frames)-1 {
		w.Content = frames[0]
	}

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

	// If we're sending to a ROUTER, we send the address first
	if socType == zmq.ROUTER {
		_, err = socket.SendBytes(w.address, zmq.SNDMORE)
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

// Address returns the address for this message, address should be set
// whenever talking to a ROUTER.
func (w *Whisper) Address() []byte {
	return w.address
}

// SetAddress sets the address for this message, address should be set
// whenever talking to a ROUTER.
func (w *Whisper) SetAddress(address []byte) {
	w.address = address
}

// SetSequence sets the sequence.
func (w *Whisper) SetSequence(sequence uint16) {
	w.sequence = sequence
}

// Sequence returns the sequence.
func (w *Whisper) Sequence() uint16 {
	return w.sequence
}
