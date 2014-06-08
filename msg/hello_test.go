package msg

import (
	zmq "github.com/pebbe/zmq4"

	"testing"
)

// Yay! Test function.
func TestHello(t *testing.T) {

	// Output
	output, err := zmq.NewSocket(zmq.DEALER)
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()

	address := "Shout"
	output.SetIdentity(address)
	err = output.Bind("inproc://selftest-hello")
	if err != nil {
		t.Fatal(err)
	}
	defer output.Unbind("inproc://selftest-hello")

	// Input
	input, err := zmq.NewSocket(zmq.ROUTER)
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()

	err = input.Connect("inproc://selftest-hello")
	if err != nil {
		t.Fatal(err)
	}
	defer input.Disconnect("inproc://selftest-hello")

	// Create a Hello message and send it through the wire
	hello := NewHello()
	hello.SetSequence(123)
	hello.Endpoint = "Life is short but Now lasts for ever"
	hello.Groups = []string{"Name: Brutus", "Age: 43"}
	hello.Status = 123
	hello.Name = "Life is short but Now lasts for ever"
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
	if tr.Endpoint != "Life is short but Now lasts for ever" {
		t.Fatalf("expected %s, got %s", "Life is short but Now lasts for ever", tr.Endpoint)
	}
	for idx, str := range []string{"Name: Brutus", "Age: 43"} {
		if tr.Groups[idx] != str {
			t.Fatalf("expected %s, got %s", str, tr.Groups[idx])
		}
	}
	if tr.Status != 123 {
		t.Fatalf("expected %d, got %d", 123, tr.Status)
	}
	if tr.Name != "Life is short but Now lasts for ever" {
		t.Fatalf("expected %s, got %s", "Life is short but Now lasts for ever", tr.Name)
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
	if address != tr.Address() {
		t.Fatalf("expected %v, got %v", address, tr.Address())
	}
}
