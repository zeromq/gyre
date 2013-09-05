package msg

import (
	zmq "github.com/vaughan0/go-zmq"

	"testing"
)

func TestShout(t *testing.T) {
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
	// Create a Shout message and send it through the wire
	shout := NewShout()
	shout.Sequence = 123
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

	msg := transit.(*Shout)
	if msg.Sequence != 123 {
		t.Fatalf("expected %d, got %d", 123, msg.Sequence)
	}
	if msg.Group != "Life is short but Now lasts for ever" {
		t.Fatalf("expected %s, got %s", "Life is short but Now lasts for ever", msg.Group)
	}
	if string(msg.Content) != "Captcha Diem" {
		t.Fatalf("expected %s, got %s", "Captcha Diem", msg.Content)
	}
}
