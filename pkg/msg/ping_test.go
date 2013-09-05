package msg

import (
	zmq "github.com/vaughan0/go-zmq"

	"testing"
)

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
	ping.Sequence = 123

	err = ping.Send(output)
	if err != nil {
		t.Fatal(err)
	}

	transit, err := Recv(input)
	if err != nil {
		t.Fatal(err)
	}

	msg := transit.(*Ping)
	if msg.Sequence != 123 {
		t.Fatalf("expected %d, got %d", 123, msg.Sequence)
	}
}
