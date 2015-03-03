package msg

import (
	"testing"

	zmq "github.com/pebbe/zmq4"
)

// Yay! Test function.
func TestPingOk(t *testing.T) {

	// Create pair of sockets we can send through

	// Output socket
	output, err := zmq.NewSocket(zmq.DEALER)
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()

	routingID := "Shout"
	output.SetIdentity(routingID)
	err = output.Bind("inproc://selftest-pingok")
	if err != nil {
		t.Fatal(err)
	}
	defer output.Unbind("inproc://selftest-pingok")

	// Input socket
	input, err := zmq.NewSocket(zmq.ROUTER)
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()

	err = input.Connect("inproc://selftest-pingok")
	if err != nil {
		t.Fatal(err)
	}
	defer input.Disconnect("inproc://selftest-pingok")

	// Create a Pingok message and send it through the wire
	pingok := NewPingOk()

	pingok.sequence = 123

	err = pingok.Send(output)
	if err != nil {
		t.Fatal(err)
	}
	transit, err := Recv(input)
	if err != nil {
		t.Fatal(err)
	}

	tr := transit.(*PingOk)

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
