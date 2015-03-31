package msg

import (
	"testing"

	zmq "github.com/pebbe/zmq4"
)

// Yay! Test function.
func TestHello(t *testing.T) {

	// Create pair of sockets we can send through

	// Output socket
	output, err := zmq.NewSocket(zmq.DEALER)
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()

	routingID := "Shout"
	output.SetIdentity(routingID)
	err = output.Bind("inproc://selftest-hello")
	if err != nil {
		t.Fatal(err)
	}
	defer output.Unbind("inproc://selftest-hello")

	// Input socket
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
	hello.sequence = 123
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

	// Tests number
	if tr.sequence != 123 {
		t.Fatalf("expected %d, got %d", 123, tr.sequence)
	}
	// Tests string
	if tr.Endpoint != "Life is short but Now lasts for ever" {
		t.Fatalf("expected %s, got %s", "Life is short but Now lasts for ever", tr.Endpoint)
	}
	// Tests strings
	for idx, str := range []string{"Name: Brutus", "Age: 43"} {
		if tr.Groups[idx] != str {
			t.Fatalf("expected %s, got %s", str, tr.Groups[idx])
		}
	}
	// Tests number
	if tr.Status != 123 {
		t.Fatalf("expected %d, got %d", 123, tr.Status)
	}
	// Tests string
	if tr.Name != "Life is short but Now lasts for ever" {
		t.Fatalf("expected %s, got %s", "Life is short but Now lasts for ever", tr.Name)
	}
	// Tests hash
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

	if routingID != string(tr.RoutingID()) {
		t.Fatalf("expected %s, got %s", routingID, string(tr.RoutingID()))
	}
}
