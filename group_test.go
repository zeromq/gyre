package gyre

import (
	"github.com/armen/gyre/msg"
	zmq "github.com/vaughan0/go-zmq"

	"bytes"
	crand "crypto/rand"
	"fmt"
	"io"
	"testing"
)

func TestGroup(t *testing.T) {

	mailbox, err := zmq.NewSocket(zmq.Dealer)
	if err != nil {
		t.Fatal(err)
	}
	mailbox.Bind("tcp://127.0.0.1:5552")

	group := newGroup("tests")

	uuid := make([]byte, 16)
	io.ReadFull(crand.Reader, uuid)
	me := fmt.Sprintf("%X", uuid)

	io.ReadFull(crand.Reader, uuid)
	you := fmt.Sprintf("%X", uuid)

	peer := newPeer(you)
	if peer.connected {
		t.Fatal("Peer shouldn't be connected yet")
	}
	err = peer.connect(me, "127.0.0.1:5552")
	if err != nil {
		t.Fatal(err)
	}
	if !peer.connected {
		t.Fatal("Peer should be connected")
	}

	group.join(peer)

	m := msg.NewHello()
	m.Ipaddress = "127.0.0.1"
	m.Mailbox = 5551
	group.send(m)

	exp, err := m.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	exp[4] = 1 // Sequence now is 1

	got, err := mailbox.Recv()
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(got[0], exp) {
		t.Errorf("Hello message is corrupted")
	}

	peer.destroy()
}
