package msg

import (
	zmq "github.com/armen/go-zmq"

	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"
	"strings"
)

const (
	keyFrame = iota
	seqFrame
	uuidFrame
	propsFrame
	bodyFrame
)

const totalFrames = 5

var (
	KeyNotSetError   = errors.New("key is not set")
	SeqNotSetError   = errors.New("sequence is not set")
	BodyNotSetError  = errors.New("body is not set")
	UuidNotSetError  = errors.New("uuid is not set")
	PropsNotSetError = errors.New("properties are not set")
	InvalidPropError = errors.New("invalid property")
)

// Msg is formatted on wire as 4 frames:
// frame 0: key (string)
// frame 1: sequence (8 bytes, network order)
// frame 2: uuid (blob, 16 bytes)
// frame 3: properties (string)
// frame 4: body (blob)
type Msg struct {
	frames map[int][]byte // Corresponding 0MQ message frames, if any
	props  []string       // List of properties, as name=value strings
}

// New creates a new msg and sets squence number
func New(seq int64) (msg *Msg) {
	msg = &Msg{
		frames: make(map[int][]byte, totalFrames),
		props:  make([]string, 0),
	}
	msg.SetSequence(seq)
	return
}

// Recv reads a key-value message from socket, and returns a new msg.
func Recv(socket *zmq.Socket) (msg *Msg, err error) {
	msg = &Msg{
		frames: make(map[int][]byte, totalFrames),
	}
	m, err := socket.Recv()
	if err != nil {
		return nil, err
	}

	for i := 0; i < totalFrames && i < len(m); i++ {
		msg.frames[i] = m[i]
	}

	// Decode properties
	msg.props = strings.Split(string(msg.frames[propsFrame]), "\n")
	if l := len(msg.props); l > 0 && msg.props[l-1] == "" {
		msg.props = msg.props[:l-1]
	}

	return
}

// Sends key-value message to socket; any empty frames are sent as such.
func (msg *Msg) Send(socket *zmq.Socket) (err error) {
	for i := 0; i < totalFrames; i++ {
		more := (i != totalFrames-1)
		err = socket.SendPart(msg.frames[i], more)
		if err != nil {
			return
		}
	}
	return
}

// Duplicates a msg and returns the new msg.
func (msg *Msg) Dup() (dup *Msg) {
	dup = &Msg{
		frames: make(map[int][]byte, totalFrames),
		props:  make([]string, len(msg.props)),
	}
	for id, frame := range msg.frames {
		dup.frames[id] = frame
	}
	copy(dup.props, msg.props)

	return
}

// Key returns key from last read message, if any, else returns error
func (msg *Msg) Key() (key string, err error) {
	if key, ok := msg.frames[keyFrame]; ok {
		return string(key), nil
	}
	err = KeyNotSetError
	return
}

// SetKey sets key frame.
func (msg *Msg) SetKey(key string) {
	msg.frames[keyFrame] = []byte(key)
}

// Sequence decodes sequence frame then returns decoded sequence number.
func (msg *Msg) Sequence() (seq int64, err error) {
	if s, ok := msg.frames[seqFrame]; ok {
		seq = int64(binary.BigEndian.Uint64(s))
		return
	}
	err = SeqNotSetError
	return
}

// SetSequence sets sequence by encoding it to bigendian byte order.
func (msg *Msg) SetSequence(seq int64) {
	s := make([]byte, 8)
	binary.BigEndian.PutUint64(s, uint64(seq))
	msg.frames[seqFrame] = s
}

// Body returns body frame.
func (msg *Msg) Body() (body []byte, err error) {
	var ok bool
	if body, ok = msg.frames[bodyFrame]; ok {
		return
	}
	err = BodyNotSetError
	return
}

// SetBody sets body frame.
func (msg *Msg) SetBody(body []byte) {
	msg.frames[bodyFrame] = body
}

// Size returns the body size of the last-read message, if any.
func (msg *Msg) Size() int {
	if body, ok := msg.frames[bodyFrame]; ok {
		return len(body)
	}
	return 0
}

// Uuid returns generated uuid
func (msg *Msg) Uuid() (uuid []byte, err error) {
	var ok bool
	if uuid, ok = msg.frames[uuidFrame]; ok {
		return
	}
	err = UuidNotSetError
	return
}

// SetUuid sets the UUID to a random generated value
func (msg *Msg) SetUuid() {
	msg.frames[uuidFrame] = make([]byte, 16)
	io.ReadFull(rand.Reader, msg.frames[uuidFrame])
}

// Prop returns property and returns err if no such property defined.
func (msg *Msg) Prop(name string) (value string, err error) {
	f := name + "="
	for _, prop := range msg.props {
		if strings.HasPrefix(prop, f) {
			value = prop[len(f):]
			return
		}
	}
	err = PropsNotSetError
	return
}

// SetProp sets message property. Property name cannot contain '='.
func (msg *Msg) SetProp(name, value string) (err error) {
	if strings.Index(name, "=") >= 0 {
		err = InvalidPropError
		return
	}
	p := name + "="
	for i, prop := range msg.props {
		if strings.HasPrefix(prop, p) {
			msg.props = append(msg.props[:i], msg.props[i+1:]...)
			break
		}
	}
	msg.props = append(msg.props, name+"="+value)

	// Encode properties
	msg.frames[propsFrame] = []byte(strings.Join(msg.props, "\n") + "\n")
	return
}

// Stores method stores the key-value message into a hash map, unless
// the key is nil.
func (msg *Msg) Store(kv map[string]*Msg) {
	key, ok := msg.frames[keyFrame]
	if !ok {
		return
	}
	if _, ok := msg.frames[bodyFrame]; ok {
		kv[string(key)] = msg
	} else {
		delete(kv, string(key))
	}
}
