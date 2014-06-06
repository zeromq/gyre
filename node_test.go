package gyre

import (
	"bytes"
	"testing"
	"time"
)

func TestNode(t *testing.T) {

	node1, err := NewNode()
	if err != nil {
		t.Fatal(err)
	}

	node1.Set("X-HELLO", "World")
	node1.Join("GLOBAL")

	node2, err := NewNode()
	if err != nil {
		t.Fatal(err)
	}

	node2.Join("GLOBAL")

	// Give time for them to interconnect
	time.Sleep(250 * time.Millisecond)

	select {
	case event := <-node1.Chan():
		if event.Type != EventEnter {
			t.Errorf("expected to recieve EventEnter but got %s", event.Type)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("No event has been received from node1")
	}

	select {
	case event := <-node2.Chan():
		if event.Type != EventEnter {
			t.Errorf("expected to recieve EventEnter but got %s", event.Type)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("No event has been received from node2")
	}

	select {
	case event := <-node1.Chan():
		if event.Type != EventJoin {
			t.Errorf("expected to recieve EventEnter but got %s", event.Type)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("No event has been received from node1")
	}

	select {
	case event := <-node2.Chan():
		if event.Type != EventJoin {
			t.Errorf("expected to recieve EventJoin but got %s", event.Type)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("No event has been received from node2")
	}

	node1.Shout("GLOBAL", []byte("Hello, World!"))

	select {
	case event := <-node2.Chan():
		if event.Type != EventShout {
			t.Errorf("expected to recieve EventShout but got %s", event.Type)
		}
		if !bytes.Equal(event.Content, []byte("Hello, World!")) {
			t.Error("expected to recieve 'Hello, World!'")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("No event has been received from node2")
	}
}
