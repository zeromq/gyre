package gyre

import (
	zmq "github.com/vaughan0/go-zmq"
	"github.com/zeromq/gyre/msg"

	"bytes"
	crand "crypto/rand"
	"fmt"
	"io"
	"testing"
)

func TestPeer(t *testing.T) {

	mailbox, err := zmq.NewSocket(zmq.Dealer)
	if err != nil {
		t.Fatal(err)
	}
	mailbox.Bind("tcp://127.0.0.1:5551")

	uuid := make([]byte, 16)
	io.ReadFull(crand.Reader, uuid)
	me := fmt.Sprintf("%X", uuid)

	io.ReadFull(crand.Reader, uuid)
	you := fmt.Sprintf("%X", uuid)

	peer := newPeer(you)
	if peer.connected {
		t.Fatal("Peer shouldn't be connected yet")
	}
	err = peer.connect(me, "tcp://127.0.0.1:5551")
	if err != nil {
		t.Fatal(err)
	}
	if !peer.connected {
		t.Fatal("Peer should be connected")
	}

	m := msg.NewHello()
	m.Endpoint = "tcp://127.0.0.1:5551"
	peer.send(m)

	exp, err := m.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	got, err := mailbox.Recv()
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(got[0], exp) {
		t.Error("Hello message was corrupted")
	}

	peer.destroy()
}
