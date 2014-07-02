package beacon

import (
	"bytes"
	"testing"
	"time"
)

func TestBeacon(t *testing.T) {
	transmit := []byte("SAMPLE-BEACON")
	b, err := New(9999)
	if err != nil {
		t.Fatal(err)
	}
	b.SetInterval(50 * time.Millisecond)
	b.Publish(transmit)

	select {
	case <-time.After(1 * time.Second):
		t.Fatalf("expected to receive a signal but got nothing!")
	case signal := <-b.Signals():
		if !bytes.Equal(transmit, signal.Transmit) {
			t.Fatalf("expected % X, got % X", transmit, signal.Transmit)
		}
	}

	b.Silence()

	select {
	case <-time.After(300 * time.Millisecond):
	case signal := <-b.Signals():
		t.Fatalf("expected silence, got %v", signal)
	}

	b.NoEcho()

	select {
	case <-time.After(300 * time.Millisecond):
	case signal := <-b.Signals():
		t.Fatalf("expected to not recive an echo, got %q, %q", signal.Addr, signal.Transmit)
	}

	b.Close()
	select {
	case <-time.After(1 * time.Second):
		t.Fatalf("expected to receive a close signal but got nothing!")
	case _, ok := <-b.Signals():
		if ok {
			t.Fatalf("Signals is not closed!")
		}
	}

	node1, err := New(5670)
	if err != nil {
		t.Fatal(err)
	}
	defer node1.Close()
	node2, err := New(5670)
	if err != nil {
		t.Fatal(err)
	}
	defer node2.Close()
	node3, err := New(5670)
	if err != nil {
		t.Fatal(err)
	}
	defer node3.Close()

	node1.SetInterval(50 * time.Millisecond)
	node2.SetInterval(50 * time.Millisecond)
	node3.SetInterval(50 * time.Millisecond)

	node1.NoEcho()
	node1.Subscribe([]byte("NODE"))
	node2.Subscribe([]byte("NODE/1"))
	node3.Subscribe([]byte("SOMETHING"))

	node1.Publish([]byte("NODE/1"))
	node2.Publish([]byte("NODE/2"))
	node3.Publish([]byte("GARBAGE"))

	for i := 0; i < 10; i++ {
		select {
		case signal := <-node1.Signals():
			expected := []byte("NODE/2")
			if !bytes.Equal(expected, signal.Transmit) {
				t.Fatalf("expected %s, got %s", expected, signal.Transmit)
			}
		case signal := <-node2.Signals():
			expected := []byte("NODE/1")
			if !bytes.Equal(expected, signal.Transmit) {
				t.Fatalf("expected %s, got %s", expected, signal.Transmit)
			}
		case signal := <-node3.Signals():
			t.Fatalf("not expected to recieve any signal from node3, got %q, %q", signal.Addr, signal.Transmit)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
