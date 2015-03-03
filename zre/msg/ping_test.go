package msg

import (
	"testing"

	zmq "github.com/pebbe/zmq4"
)

// Yay! Test function.
func TestPing(t *testing.T) {

	// Create pair of sockets we can send through

	// Output socket
	output, err := zmq.NewSocket(zmq.DEALER)
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()

	routingID := "Shout"
	output.SetIdentity(routingID)
	err = output.Bind("inproc://selftest-ping")
	if err != nil {
		t.Fatal(err)
	}
	defer output.Unbind("inproc://selftest-ping")

	// Input socket
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

	ping.sequence = 123

	err = ping.Send(output)
	if err != nil {
		t.Fatal(err)
	}
	transit, err := Recv(input)
	if err != nil {
		t.Fatal(err)
	}

	tr := transit.(*Ping)

	if tr.sequence != 123 {
		t.Fatalf("expected %d, got %d", 123, tr.sequence)
	}

	err = tr.Send(input)
	if err != nil {
		t.Fatal(err)
	}

	transit, err = Recv(output)
	if err != nil {
		t.Fatal(err)
	}

	if routingID != string(tr.RoutingID()) {
		t.Fatalf("expected %s, got %s", routingID, string(tr.RoutingID()))
	}
}
