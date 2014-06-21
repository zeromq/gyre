package msg

import (
	zmq "github.com/pebbe/zmq4"

	"testing"
)

// Yay! Test function.
func TestJoin(t *testing.T) {

	// Output
	output, err := zmq.NewSocket(zmq.DEALER)
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()

	address := "Shout"
	output.SetIdentity(address)
	err = output.Bind("inproc://selftest-join")
	if err != nil {
		t.Fatal(err)
	}
	defer output.Unbind("inproc://selftest-join")

	// Input
	input, err := zmq.NewSocket(zmq.ROUTER)
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()

	err = input.Connect("inproc://selftest-join")
	if err != nil {
		t.Fatal(err)
	}
	defer input.Disconnect("inproc://selftest-join")

	// Create a Join message and send it through the wire
	join := NewJoin()
	join.SetSequence(123)
	join.Group = "Life is short but Now lasts for ever"
	join.Status = 123

	err = join.Send(output)
	if err != nil {
		t.Fatal(err)
	}
	transit, err := Recv(input)
	if err != nil {
		t.Fatal(err)
	}

	tr := transit.(*Join)
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
	if address != string(tr.Address()) {
		t.Fatalf("expected %s, got %s", address, string(tr.Address()))
	}
}
