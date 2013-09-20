package msg

import (
	zmq "github.com/armen/go-zmq"

	"os"
	"testing"
)

func TestMsg(t *testing.T) {
	// Prepare our context and sockets
	output, err := zmq.NewSocket(zmq.Dealer)
	defer output.Close()
	if err != nil {
		t.Fatal(err)
	}

	err = output.Bind("ipc://msg_selftest.ipc")
	if err != nil {
		t.Fatal(err)
	}

	input, err := zmq.NewSocket(zmq.Dealer)
	defer input.Close()
	if err != nil {
		t.Fatal(err)
	}

	err = input.Connect("ipc://msg_selftest.ipc")
	if err != nil {
		t.Fatal(err)
	}

	kv := make(map[string]*Msg)

	// Test send and receive of simple message
	msg := New(1)
	msg.SetKey("key")
	msg.SetUuid()
	msg.SetBody([]byte("body"))
	err = msg.Send(output)
	if err != nil {
		t.Error(err)
	}

	msg.Store(kv)
	if _, ok := kv["key"]; !ok {
		t.Error("expected to have a key but it isn't set")
	}

	msg, err = Recv(input)
	if err != nil {
		t.Error(err)
	}
	key, err := msg.Key()
	if err != nil {
		t.Error(err)
	}
	if key != "key" {
		t.Errorf("expected %q, got %q", "key", key)
	}

	// Test send and receive of message with properties
	msg = New(2)
	err = msg.SetProp("prop1", "value1")
	if err != nil {
		t.Error(err)
	}
	msg.SetProp("prop2", "value1")
	msg.SetProp("prop2", "value2")
	msg.SetProp("prop3", "value3")
	msg.SetKey("key")
	msg.SetUuid()
	msg.SetBody([]byte("body"))
	if val, err := msg.Prop("prop2"); err != nil || val != "value2" {
		if err != nil {
			t.Fatal(err)
		}
		t.Errorf("expected %q = %q, got %q", "prop2", "value2", val)
	}
	err = msg.Send(output)

	msg, err = Recv(input)
	if err != nil {
		t.Error(err)
	}
	key, err = msg.Key()
	if err != nil {
		t.Error(err)
	}
	if key != "key" {
		t.Errorf("expected %q, got %q", "key", key)
	}
	prop, err := msg.Prop("prop2")
	if err != nil {
		t.Fatal(err)
	}
	if prop != "value2" {
		t.Errorf("expected property %q, got %q", "value2", prop)
	}

	prop, err = msg.Prop("prop3")
	if err != nil {
		t.Fatal(err)
	}
	if prop != "value3" {
		t.Errorf("expected property %q, got %q", "value3", prop)
	}

	input.Close()
	output.Close()
	os.Remove("msg_selftest.ipc")
}
