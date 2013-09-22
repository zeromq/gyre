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
	// Create a Hello message and send it through the wire
	hello := NewHello()
	hello.SetSequence(123)
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

	tr := transit.(*Hello)
	if tr.Sequence() != 123 {
		t.Fatalf("expected %d, got %d", 123, tr.Sequence())
	}
	if tr.Ipaddress != "Life is short but Now lasts for ever" {
		t.Fatalf("expected %s, got %s", "Life is short but Now lasts for ever", tr.Ipaddress)
	}
	if tr.Mailbox != 123 {
		t.Fatalf("expected %d, got %d", 123, tr.Mailbox)
	}
	for idx, str := range []string{"Name: Brutus", "Age: 43"} {
		if tr.Groups[idx] != str {
			t.Fatalf("expected %s, got %s", str, tr.Groups[idx])
		}
	}
	if tr.Status != 123 {
		t.Fatalf("expected %d, got %d", 123, tr.Status)
	}
	for key, val := range map[string]string{"Name": "Brutus", "Age": "43"} {
		if tr.Headers[key] != val {
			t.Fatalf("expected %s, got %s", val, tr.Headers[key])
		}
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
