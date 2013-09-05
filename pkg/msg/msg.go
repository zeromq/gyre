package msg

import (
	zmq "github.com/vaughan0/go-zmq"

	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
)

const (
	Signature uint16 = 0xAAA0 | 1
	StringMax        = 255
	Version          = 1
)

type Transit interface {
	Marshal() ([]byte, error)
	Unmarshal([][]byte) error
	String() string
	SetAddress([]byte)
	Send(*zmq.Socket) error
	SetSequence(uint16)
}

// Recv receives marshaled data from 0mq socket
func Recv(socket *zmq.Socket) (t Transit, err error) {
	var (
		buffer  *bytes.Buffer
		address []byte
		frames  [][]byte
	)

	// Read valid message frame from socket; we loop over any
	// garbage data we might receive from badly-connected peers
	for {
		// Read all frames
		frames, err = socket.Recv()
		if err != nil {
			return nil, err
		}
		// If we're reading from a ROUTER socket, get address
		if socket.GetType() == zmq.Router {
			if len(frames) <= 1 {
				return nil, errors.New("malformed message")
			}
			address = frames[0]
			frames = frames[1:]
		}
		// Check the signature
		var signature uint16
		buffer = bytes.NewBuffer(frames[0])
		binary.Read(buffer, binary.BigEndian, &signature)
		if signature == Signature {
			// Valid signature
			break
		}
	}

	// Get message id and parse per message type
	var id uint8
	binary.Read(buffer, binary.BigEndian, &id)

	switch id {
	case HelloId:
		t = NewHello()
	case WhisperId:
		t = NewWhisper()
	case ShoutId:
		t = NewShout()
	case JoinId:
		t = NewJoin()
	case LeaveId:
		t = NewLeave()
	case PingId:
		t = NewPing()
	case PingOkId:
		t = NewPingOk()
	}
	t.SetAddress(address)
	err = t.Unmarshal(frames)

	return t, err
}

// putString marshals a string into the buffer
func putString(buffer *bytes.Buffer, str string) {
	size := len(str)
	if size > StringMax {
		size = StringMax
	}
	sz := fmt.Sprintf("%d", size)
	str = fmt.Sprintf("%"+sz+"s", str)
	binary.Write(buffer, binary.BigEndian, byte(size))
	binary.Write(buffer, binary.BigEndian, []byte(str))
}

// putKeyValString marshals a key=val pair into the buffer
func putKeyValString(buffer *bytes.Buffer, key, val string) {
	str := fmt.Sprintf("%s=%s", key, val)
	putString(buffer, str)
}

// getString unmarshals a string from the buffer
func getString(buffer *bytes.Buffer) string {
	var size byte
	binary.Read(buffer, binary.BigEndian, &size)
	str := make([]byte, size)
	binary.Read(buffer, binary.BigEndian, &str)
	return string(str)
}

// getKeyValString unmarshals a key=val pair from the buffer
func getKeyValString(buffer *bytes.Buffer) (key, val string) {
	str := getString(buffer)
	strs := strings.SplitN(str, "=", 2)

	if len(strs) == 2 {
		return strs[0], strs[1]
	}

	return "", ""
}
