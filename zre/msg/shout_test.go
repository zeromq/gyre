package msg

import (
	"testing"

	zmq "github.com/pebbe/zmq4"
)

// Yay! Test function.
func TestShout(t *testing.T) {

	// Create pair of sockets we can send through

	// Output socket
	output, err := zmq.NewSocket(zmq.DEALER)
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()

	routingId := "Shout"
	output.SetIdentity(routingId)
	err = output.Bind("inproc://selftest-shout")
	if err != nil {
		t.Fatal(err)
	}
	defer output.Unbind("inproc://selftest-shout")

	// Input socket
	input, err := zmq.NewSocket(zmq.ROUTER)
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()

	err = input.Connect("inproc://selftest-shout")
	if err != nil {
		t.Fatal(err)
	}
	defer input.Disconnect("inproc://selftest-shout")

	// Create a Shout message and send it through the wire
	shout := NewShout()

	shout.sequence = 123

	shout.Group = "Life is short but Now lasts for ever"

	shout.Content = []byte("Captcha Diem")

	err = shout.Send(output)
	if err != nil {
		t.Fatal(err)
	}
	transit, err := Recv(input)
	if err != nil {
		t.Fatal(err)
	}

	tr := transit.(*Shout)

	if tr.sequence != 123 {
		t.Fatalf("expected %d, got %d", 123, tr.sequence)
	}

	if tr.Group != "Life is short but Now lasts for ever" {
		t.Fatalf("expected %s, got %s", "Life is short but Now lasts for ever", tr.Group)
	}

	if string(tr.Content) != "Captcha Diem" {
		t.Fatalf("expected %s, got %s", "Captcha Diem", tr.Content)
	}

	err = tr.Send(input)
	if err != nil {
		t.Fatal(err)
	}

	transit, err = Recv(output)
	if err != nil {
		t.Fatal(err)
	}

	if routingId != string(tr.RoutingId()) {
		t.Fatalf("expected %s, got %s", routingId, string(tr.RoutingId()))
	}
}
