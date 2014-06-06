package msg

import (
	zmq "github.com/pebbe/zmq4"

	"testing"
)

// Yay! Test function.
func TestLeave(t *testing.T) {

	// Output
	output, err := zmq.NewSocket(zmq.DEALER)
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()

	address := "Shout"
	output.SetIdentity(address)
	err = output.Bind("inproc://selftest-leave")
	if err != nil {
		t.Fatal(err)
	}
	defer output.Unbind("inproc://selftest-leave")

	// Input
	input, err := zmq.NewSocket(zmq.ROUTER)
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()

	err = input.Connect("inproc://selftest-leave")
	if err != nil {
		t.Fatal(err)
	}
	defer input.Disconnect("inproc://selftest-leave")

	// Create a Leave message and send it through the wire
	leave := NewLeave()
	leave.SetSequence(123)
	leave.Group = "Life is short but Now lasts for ever"
	leave.Status = 123

	err = leave.Send(output)
	if err != nil {
		t.Fatal(err)
	}
	transit, err := Recv(input)
	if err != nil {
		t.Fatal(err)
	}

	tr := transit.(*Leave)
	if tr.Sequence() != 123 {
		t.Fatalf("expected %d, got %d", 123, tr.Sequence())
	}
	if tr.Group != "Life is short but Now lasts for ever" {
		t.Fatalf("expected %s, got %s", "Life is short but Now lasts for ever", tr.Group)
	}
	if tr.Status != 123 {
		t.Fatalf("expected %d, got %d", 123, tr.Status)
	}

	err = tr.Send(input)
	if err != nil {
		t.Fatal(err)
	}
	transit, err = Recv(output)
	if err != nil {
		t.Fatal(err)
	}
	if address != tr.Address() {
		t.Fatalf("expected %v, got %v", address, tr.Address())
	}
}
