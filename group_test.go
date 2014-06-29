package gyre

import (
	zmq "github.com/pebbe/zmq4"
	"github.com/zeromq/gyre/msg"

	"bytes"
	crand "crypto/rand"
	"io"
	"testing"
)

func TestGroup(t *testing.T) {

	mailbox, err := zmq.NewSocket(zmq.DEALER)
	if err != nil {
		t.Fatal(err)
	}
	mailbox.Bind("tcp://127.0.0.1:5552")

	group := newGroup("tlests")

	me := make([]byte, 16)
	io.ReadFull(crand.Reader, me)

	you := make([]byte, 16)
	io.ReadFull(crand.Reader, you)

	peer := newPeer(string(you))
	if peer.connected {
		t.Fatal("Peer shouldn't be connected yet")
	}
	err = peer.connect(me, "tcp://127.0.0.1:5552")
	if err != nil {
		t.Fatal(err)
	}
	if !peer.connected {
		t.Fatal("Peer should be connected")
	}

	group.join(peer)

	m := msg.NewHello()
	m.Endpoint = "tcp://127.0.0.1:5551"
	group.send(m)

	exp, err := m.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	exp[4] = 1 // Sequence now is 1

	got, err := mailbox.Recv(0)
	if err != nil {
		t.Fatal(err)
	}
	gotb := []byte(got)

	if !bytes.Equal(gotb, exp) {
		t.Errorf("Hello message is corrupted")
	}

	peer.destroy()
}
