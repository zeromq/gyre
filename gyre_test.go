package gyre

import (
	"bytes"
	"testing"
	"time"
)

func TestNode(t *testing.T) {

	node1, err := New()
	if err != nil {
		t.Fatal(err)
	}
	node1.SetName("node1")
	node1.SetHeader("X-HELLO", "World")
	node1.SetVerbose()
	node1.SetPort(5670)
	err = node1.Start()
	if err != nil {
		t.Fatal(err)
	}
	node1.Join("GLOBAL")

	node2, err := New()
	if err != nil {
		t.Fatal(err)
	}
	node2.SetName("node2")
	node2.SetVerbose()
	node2.SetPort(5670)
	err = node2.Start()
	if err != nil {
		t.Fatal(err)
	}
	node2.Join("GLOBAL")

	// Give time for them to interconnect
	time.Sleep(1500 * time.Millisecond)

	node1.Shout("GLOBAL", []byte("Hello, World!"))

	select {
	case event := <-node2.Events():

		if event.Type() != EventEnter {
			t.Errorf("expected to recieve EventEnter but got %#v", event.Type())
		}
		header, ok := event.Header("X-HELLO")
		if !ok || header != "World" {
			t.Errorf("expected World but got %s", header)
		}
		if event.Name() != "node1" {
			t.Errorf("expected node1 but got %s", event.Name())
		}
	case <-time.After(1 * time.Second):
		t.Fatal("No event has been received from node2")
	}

	select {
	case event := <-node2.Events():
		if event.Type() != EventJoin {
			t.Errorf("expected to recieve EventJoin but got %#v", event.Type())
		}
	case <-time.After(1 * time.Second):
		t.Fatal("No event has been received from node2")
	}

	select {
	case event := <-node2.Events():
		if event.Type() != EventShout {
			t.Errorf("expected to recieve EventShout but got %#v", event.Type())
		}
		if !bytes.Equal(event.Msg(), []byte("Hello, World!")) {
			t.Error("expected to recieve 'Hello, World!'")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("No event has been received from node2")
	}

	node1.Stop()
	node2.Stop()
}
