package msg

import (
	zmq "github.com/pebbe/zmq4"

	"testing"
)

// Yay! Test function.
func TestShout(t *testing.T) {

	// Output
	output, err := zmq.NewSocket(zmq.DEALER)
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()

	address := "Shout"
	output.SetIdentity(address)
	err = output.Bind("inproc://selftest-shout")
	if err != nil {
		t.Fatal(err)
	}
	defer output.Unbind("inproc://selftest-shout")

	// Input
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
	shout.SetSequence(123)
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
	if tr.Sequence() != 123 {
		t.Fatalf("expected %d, got %d", 123, tr.Sequence())
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
	if address != string(tr.Address()) {
		t.Fatalf("expected %s, got %s", address, string(tr.Address()))
	}
}
