package msg

import (
	zmq "github.com/vaughan0/go-zmq"

	"testing"
)

func TestHello(t *testing.T) {
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
	// Create a Hello message and send it through the wire
	hello := NewHello()
	hello.Sequence = 123
	hello.Ipaddress = "Life is short but Now lasts for ever"
	hello.Mailbox = 123
	hello.Groups = []string{"Name: Brutus", "Age: 43"}
	hello.Status = 123
	hello.Headers = map[string]string{"Name": "Brutus", "Age": "43"}

	err = hello.Send(output)
	if err != nil {
		t.Fatal(err)
	}

	transit, err := Recv(input)
	if err != nil {
		t.Fatal(err)
	}

	msg := transit.(*Hello)
	if msg.Sequence != 123 {
		t.Fatalf("expected %d, got %d", 123, msg.Sequence)
	}
	if msg.Ipaddress != "Life is short but Now lasts for ever" {
		t.Fatalf("expected %s, got %s", "Life is short but Now lasts for ever", msg.Ipaddress)
	}
	if msg.Mailbox != 123 {
		t.Fatalf("expected %d, got %d", 123, msg.Mailbox)
	}
	for idx, str := range []string{"Name: Brutus", "Age: 43"} {
		if msg.Groups[idx] != str {
			t.Fatalf("expected %s, got %s", str, msg.Groups[idx])
		}
	}
	if msg.Status != 123 {
		t.Fatalf("expected %d, got %d", 123, msg.Status)
	}
	for key, val := range map[string]string{"Name": "Brutus", "Age": "43"} {
		if msg.Headers[key] != val {
			t.Fatalf("expected %s, got %s", val, msg.Headers[key])
		}
	}
}
