package msg

import (
	zmq "github.com/vaughan0/go-zmq"

	"testing"
)

// Yay! Test function.
func TestPing(t *testing.T) {
	context, err := zmq.NewContext()
	if err != nil {
		t.Fatal(err)
	}

	// Output
	output, err := context.Socket(zmq.Dealer)
	if err != nil {
		t.Fatal(err)
	}
	address := []byte("Shout")
	output.SetIdentitiy(address)
	err = output.Bind("inproc://selftest")
	if err != nil {
		t.Fatal(err)
	}

	// Input
	input, err := context.Socket(zmq.Router)
	if err != nil {
		t.Fatal(err)
	}
	err = input.Connect("inproc://selftest")
	if err != nil {
		t.Fatal(err)
	}
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
	if string(address) != string(tr.Address()) {
		t.Fatalf("expected %v, got %v", address, tr.Address())
	}
}
