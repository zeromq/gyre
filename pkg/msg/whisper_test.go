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
	// Create a Whisper message and send it through the wire
	whisper := NewWhisper()
	whisper.SetSequence(123)
	whisper.Content = []byte("Captcha Diem")

	err = whisper.Send(output)
	if err != nil {
		t.Fatal(err)
	}
	transit, err := Recv(input)
	if err != nil {
		t.Fatal(err)
	}

	tr := transit.(*Whisper)
	if tr.Sequence() != 123 {
		t.Fatalf("expected %d, got %d", 123, tr.Sequence())
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
	if string(address) != string(tr.Address()) {
		t.Fatalf("expected %v, got %v", address, tr.Address())
	}
}
