package msg

import (
	zmq "github.com/pebbe/zmq4"

	"testing"
)

// Yay! Test function.
func TestWhisper(t *testing.T) {

	// Output
	output, err := zmq.NewSocket(zmq.DEALER)
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()

	address := "Shout"
	output.SetIdentity(address)
	err = output.Bind("inproc://selftest-whisper")
	if err != nil {
		t.Fatal(err)
	}
	defer output.Unbind("inproc://selftest-whisper")

	// Input
	input, err := zmq.NewSocket(zmq.ROUTER)
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()

	err = input.Connect("inproc://selftest-whisper")
	if err != nil {
		t.Fatal(err)
	}
	defer input.Disconnect("inproc://selftest-whisper")

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
	if address != tr.Address() {
		t.Fatalf("expected %v, got %v", address, tr.Address())
	}
}
