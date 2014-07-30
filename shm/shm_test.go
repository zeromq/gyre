package shm

import (
	"bytes"
	"sync"
	"testing"
)

func TestShm(t *testing.T) {
	testVal := []byte("Sample Value")
	testVal2 := []byte("Sample Value2")

	hm := New()
	hm.Subtree("This/Is/It").Node("1").SetVal(testVal)
	val := hm.Subtree("This/Is/It").Node("1").Val()

	if !bytes.Equal(val, testVal) {
		t.Errorf("expected [% x] but got [% x]", testVal, val)
	}

	hm.Subtree("This/Is/It").Node("1").Props["foo"] = "bar"

	if val := hm.Subtree("This/Is/It").Node("1").Props["foo"]; val != "bar" {
		t.Errorf("expected %q but got %q", "bar", val)
	}

	if val := hm.Subtree("This/Is/It").Node("1").Props.Get("foo"); val != "bar" {
		t.Errorf("expected %q but got %q", "bar", val)
	}

	empty := hm.Subtree("This/Is/It").Node("1").Props.Get("foo2")
	if empty != "" {
		t.Errorf("expected an empty property but got %q", empty)
	}

	empty, ok := hm.Subtree("This/Is/It").Node("1").Props.GetOk("foo3")
	if empty != "" || ok {
		t.Errorf("expected an empty property but got %q", empty)
	}

	hm.Subtree("This/Is/It").DelNode("1")
	emptyVal := hm.Subtree("This/Is/It").Node("1").Val()
	if !bytes.Equal(emptyVal, []byte("")) {
		t.Errorf("expected an empty val but got [% x]", emptyVal)
	}

	hm.Subtree("This/Is/It2").Node("2").SetVal(testVal2)
	val = hm.Subtree("This/Is/It2").Node("2").Val()
	if !bytes.Equal(val, testVal2) {
		t.Errorf("expected [% x] but got [% x]", testVal, val)
	}

	hm.DelSubtree("This/Is/It2")
	emptyVal = hm.Subtree("This/Is/It2").Node("2").Val()
	if !bytes.Equal(emptyVal, []byte("")) {
		t.Errorf("expected an empty val but got [% x]", emptyVal)
	}
}

// Just to test with -race parameter: go test -race
func TestShmParallel(t *testing.T) {
	testVal := []byte("Sample Value")
	testVal2 := []byte("Sample Value2")

	hm := New()

	var wg sync.WaitGroup
	wg.Add(1000)

	bcast := make(chan struct{})

	for i := 0; i < 1000; i++ {

		go func() {
			defer wg.Done()

			select {
			case <-bcast:
			}

			hm.Subtree("This/Is/It").Node("1").SetVal(testVal)
			val := hm.Subtree("This/Is/It").Node("1").Val()
			hm.Subtree("This/Is/It").Node("1").Props["foo"] = "bar"
			hm.Subtree("This/Is/It").Node("1").Props.Get("foo")
			hm.Subtree("This/Is/It").Node("1").Props.Get("foo2")
			hm.Subtree("This/Is/It").Node("1").Props.GetOk("foo3")
			hm.Subtree("This/Is/It").DelNode("1")
			hm.Subtree("This/Is/It").Node("1").SetVal(val)
			val = hm.Subtree("This/Is/It").Node("1").Val()
			hm.Subtree("This/Is/It2").Node("2").SetVal(testVal2)
			val = hm.Subtree("This/Is/It2").Node("2").Val()
			hm.DelSubtree("This/Is/It2")
			val = hm.Subtree("This/Is/It2").Node("2").Val()
		}()
	}

	// Start all the go routins by broadcasting
	close(bcast)

	// Wait for them
	wg.Wait()
}
