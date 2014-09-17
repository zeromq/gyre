package gyre

import (
	"bytes"
	"log"
	"reflect"
	"strconv"
	"testing"
	"time"
)

const (
	numOfNodes = 5
)

var (
	gyre    = make([]*Gyre, numOfNodes)
	nodes   = make([]*node, numOfNodes)
	headers = make([]map[string]string, numOfNodes)
)

func launchNodes(n int) {

	var err error

	for i := 0; i < n; i++ {
		gyre[i], nodes[i], err = newGyre()
		if err != nil {
			log.Fatal(err)
		}
		gyre[i].SetName("node" + strconv.Itoa(i))
		gyre[i].SetHeader("X-HELLO-"+strconv.Itoa(i), "World-"+strconv.Itoa(i))
		headers[i] = make(map[string]string)
		headers[i]["X-HELLO-"+strconv.Itoa(i)] = "World-" + strconv.Itoa(i)
		// You might want to make it verbose
		gyre[i].SetVerbose()
		gyre[i].SetPort(5660)
		gyre[i].SetInterface("lo")
		err = gyre[i].Start()
		if err != nil {
			log.Fatal(err)
		}
		gyre[i].Join("GLOBAL")
	}

	// Give time for them to interconnect
	time.Sleep(1500 * time.Millisecond)
}

func stopNodes(n int) {
	for i := 0; i < n; i++ {
		gyre[i].Stop()
		gyre[i] = nil
		nodes[i] = nil
		time.Sleep(100 * time.Millisecond)
	}
}

func TestNode(t *testing.T) {

	launchNodes(2)
	defer stopNodes(2)

	gyre[0].Shout("GLOBAL", []byte("Hello, World!"))

	if gyre[1].Addr() == "" {
		t.Errorf("Addr() shouldn't return empty string")
	}

	select {
	case event := <-gyre[1].Events():

		if event.Type() != EventEnter {
			t.Errorf("expected to recieve EventEnter but got %#v", event.Type())
		}
		if event.Name() != "node0" {
			t.Errorf("expected node0 but got %s", event.Name())
		}
	case <-time.After(1 * time.Second):
		t.Error("No event has been received from gyre[1]")
	}

	select {
	case event := <-gyre[1].Events():
		if event.Type() != EventJoin {
			t.Errorf("expected to recieve EventJoin but got %#v", event.Type())
		}
	case <-time.After(1 * time.Second):
		t.Error("No event has been received from node1")
	}

	select {
	case event := <-gyre[1].Events():
		if event.Type() != EventShout {
			t.Errorf("expected to recieve EventShout but got %#v", event.Type())
		}
		if !bytes.Equal(event.Msg(), []byte("Hello, World!")) {
			t.Error("expected to recieve 'Hello, World!'")
		}
	case <-time.After(1 * time.Second):
		t.Error("No event has been received from node1")
	}
}

func TestSyncedHeaders(t *testing.T) {
	launchNodes(numOfNodes)
	defer stopNodes(numOfNodes)

	for i := 0; i < numOfNodes; i++ {
		if !reflect.DeepEqual(gyre[i].Headers(), headers[i]) {
			t.Errorf("expected %v got %v", headers[i], gyre[i].Headers())
		}
	}

	// Make sure exchanged headers between peers are the consistent
	for i := 0; i < numOfNodes; i++ {
		for j := 0; j < numOfNodes; j++ {
			if j == i {
				continue
			}
			identity := nodes[i].identity()

			if nodes[j].peers[identity] == nil {
				t.Errorf("headers of node%d and node%d are not synced. expected %v but its empty", i, j, nodes[i].headers)
			} else if !reflect.DeepEqual(nodes[i].headers, nodes[j].peers[identity].headers) {
				t.Errorf("headers of node%d and node%d are not synced. expected %v but got %v", i, j, nodes[i].headers, nodes[j].peers[identity].headers)
			} else if nodes[i].name != nodes[j].peers[identity].name {
				t.Errorf("name of node%d and stored name in node%d are not same.expected %v but got %v", i, j, nodes[i].name, nodes[j].peers[identity].name)
			}
		}
	}
}
