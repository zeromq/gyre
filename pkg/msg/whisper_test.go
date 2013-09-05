package msg

import (
	zmq "github.com/vaughan0/go-zmq"

	"testing"
)

func TestWhisper(t *testing.T) {
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
	// Create a Whisper message and send it through the wire
	whisper := NewWhisper()
	whisper.Sequence = 123
	whisper.Content = []byte("Captcha Diem")

	err = whisper.Send(output)
	if err != nil {
		t.Fatal(err)
	}

	transit, err := Recv(input)
	if err != nil {
		t.Fatal(err)
	}

	msg := transit.(*Whisper)
	if msg.Sequence != 123 {
		t.Fatalf("expected %d, got %d", 123, msg.Sequence)
	}
	if string(msg.Content) != "Captcha Diem" {
		t.Fatalf("expected %s, got %s", "Captcha Diem", msg.Content)
	}
}
