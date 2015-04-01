package beacon

import (
	"bytes"
	"testing"
	"time"
)

func TestBeacon(t *testing.T) {
	transmit := []byte("SAMPLE-BEACON")
	b := New()
	b.SetPort(9999).SetInterval(50 * time.Millisecond)
	err := b.Publish(transmit)
	if err != nil {
		t.Fatal(err)
	}

	select {
	case <-time.After(1 * time.Second):
		t.Fatalf("expected to receive a signal but got nothing!")
	case s := <-b.Signals():
		signal := s.(*Signal)
		if !bytes.Equal(transmit, signal.Transmit) {
			t.Fatalf("expected % X, got % X", transmit, signal.Transmit)
		}
	}

	b.Silence()

	select {
	case <-time.After(300 * time.Millisecond):
	case s := <-b.Signals():
		signal := s.(*Signal)
		t.Fatalf("expected silence, got %v", signal)
	}

	b.NoEcho()

	select {
	case <-time.After(300 * time.Millisecond):
	case s := <-b.Signals():
		signal := s.(*Signal)
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

	node1 := New()
	defer node1.Close()
	node2 := New()
	defer node2.Close()
	node3 := New()
	defer node3.Close()

	node1.SetPort(5670).SetInterval(50 * time.Millisecond)
	node2.SetPort(5670).SetInterval(50 * time.Millisecond)
	node3.SetPort(5670).SetInterval(50 * time.Millisecond)

	node1.NoEcho()
	node1.Subscribe([]byte("NODE"))
	node2.Subscribe([]byte("NODE/1"))
	node3.Subscribe([]byte("SOMETHING"))

	err = node1.Publish([]byte("NODE/1"))
	if err != nil {
		t.Fatal(err)
	}
	err = node2.Publish([]byte("NODE/2"))
	if err != nil {
		t.Fatal(err)
	}
	err = node3.Publish([]byte("GARBAGE"))
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		select {
		case s := <-node1.Signals():
			signal := s.(*Signal)
			expected := []byte("NODE/2")
			if !bytes.Equal(expected, signal.Transmit) {
				t.Fatalf("expected %s, got %s", expected, signal.Transmit)
			}
		case s := <-node2.Signals():
			signal := s.(*Signal)
			expected := []byte("NODE/1")
			if !bytes.Equal(expected, signal.Transmit) {
				t.Fatalf("expected %s, got %s", expected, signal.Transmit)
			}
		case s := <-node3.Signals():
			signal := s.(*Signal)
			t.Fatalf("not expected to recieve any signal from node3, got %q, %q", signal.Addr, signal.Transmit)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
