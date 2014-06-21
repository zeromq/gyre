package msg

import (
	zmq "github.com/pebbe/zmq4"

	"testing"
)

// Yay! Test function.
func TestPing(t *testing.T) {

	// Output
	output, err := zmq.NewSocket(zmq.DEALER)
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()

	address := "Shout"
	output.SetIdentity(address)
	err = output.Bind("inproc://selftest-ping")
	if err != nil {
		t.Fatal(err)
	}
	defer output.Unbind("inproc://selftest-ping")

	// Input
	input, err := zmq.NewSocket(zmq.ROUTER)
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()

	err = input.Connect("inproc://selftest-ping")
	if err != nil {
		t.Fatal(err)
	}
	defer input.Disconnect("inproc://selftest-ping")

	// Create a Ping message and send it through the wire
	ping := NewPing()
	ping.SetSequence(123)

	err = ping.Send(output)
	if err != nil {
		t.Fatal(err)
	}
	transit, err := Recv(input)
	if err != nil {
		t.Fatal(err)
	}

	tr := transit.(*Ping)
	if tr.Sequence() != 123 {
		t.Fatalf("expected %d, got %d", 123, tr.Sequence())
	}

	err = tr.Send(input)
	if err != nil {
		t.Fatal(err)
	}
	transit, err = Recv(output)
	if err != nil {
		t.Fatal(err)
	}
	if address != string(tr.Address()) {
		t.Fatalf("expected %s, got %s", address, string(tr.Address()))
	}
}
