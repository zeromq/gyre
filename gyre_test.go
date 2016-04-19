package gyre

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"reflect"
	"strconv"
	"testing"
	"time"
)

const (
	numOfNodes = 3
)

var (
	gyre    = make([]*Gyre, numOfNodes)
	nodes   = make([]*node, numOfNodes)
	headers = make([]map[string]string, numOfNodes)
	id      int
)

func launchNodes(n, port int, wait time.Duration) {

	var err error

	rand.Seed(time.Now().UTC().UnixNano())
	id = rand.Int()

	for i := 0; i < n; i++ {
		gyre[i], nodes[i], err = newGyre()
		if err != nil {
			log.Fatal(err)
		}
		// You might want to make it verbose
		// gyre[i].SetVerbose()
		// log.SetFlags(log.LstdFlags | log.Lshortfile)

		gyre[i].SetPort(port)

		// Be aware that ZSYS_INTERFACE or BEACON_INTERFACE has precedence
		// So make sure they are not set. For testing with docker you might
		// want to set one of those env variables to docker0
		gyre[i].SetInterface("lo")

		gyre[i].SetName("node" + strconv.Itoa(i))
		gyre[i].SetHeader("X-HELLO-"+strconv.Itoa(i), "World-"+strconv.Itoa(i))
		headers[i] = make(map[string]string)
		headers[i]["X-HELLO-"+strconv.Itoa(i)] = "World-" + strconv.Itoa(i)

		if port == 0 {
			// Enable the gossip
			if i == 0 {
				gyre[i].GossipBind(fmt.Sprintf("inproc://gossip-hub-%d", id))
			} else {
				gyre[i].GossipConnect(fmt.Sprintf("inproc://gossip-hub-%d", id))
			}
		}

		err = gyre[i].Start()
		if err != nil {
			log.Fatal(err)
		}

		gyre[i].Join("GLOBAL")
	}

	// Give time for them to interconnect
	time.Sleep(wait)
}

func stopNodes(n int) {
	for i := 0; i < n; i++ {
		gyre[i].Stop()
		time.Sleep(500 * time.Millisecond)
		gyre[i] = nil
		nodes[i] = nil
	}
}

func TestTwoNodes(t *testing.T) {
	testTwoNodes(t, 5660, 1*time.Second)
}

func TestTwoNodesWithGossipDiscovery(t *testing.T) {
	testTwoNodes(t, 0, 1*time.Second) // Test with gossip discovery
}

func TestSyncedHeaders(t *testing.T) {
	testSyncedHeaders(t, numOfNodes, 5660, 1*time.Second)
}

func TestSyncedHeadersWithGossipDiscovery(t *testing.T) {
	testSyncedHeaders(t, numOfNodes, 0, 1*time.Second) // Test with gossip discovery
}

func TestJoinLeave(t *testing.T) {
	launchNodes(2, 5660, 1*time.Second)
	defer stopNodes(2)

	go func() {
		<-gyre[1].Events()
	}()

	select {
	case <-gyre[0].Events():
		gyre[0].Leave("GLOBAL")
	}
}

func testTwoNodes(t *testing.T, port int, wait time.Duration) {
	launchNodes(2, port, wait)
	defer stopNodes(2)

	gyre[0].Shout("GLOBAL", []byte("Hello, World!"))

	// Give them time to receive the msg
	time.Sleep(wait)

	if addr, err := gyre[1].Addr(); err != nil {
		t.Errorf(err.Error())
	} else if addr == "" {
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

	time.Sleep(wait)

	select {
	case event := <-gyre[1].Events():
		if event.Type() != EventJoin {
			t.Errorf("expected to recieve EventJoin but got %#v", event.Type())
		}
	case <-time.After(1 * time.Second):
		t.Error("No event has been received from node1")
	}

	time.Sleep(wait)

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

func testSyncedHeaders(t *testing.T, n, port int, wait time.Duration) {
	launchNodes(n, port, wait)
	defer stopNodes(n)

	for i := 0; i < n; i++ {
		h, err := gyre[i].Headers()
		if err != nil {
			t.Errorf(err.Error())
		}

		if !reflect.DeepEqual(h, headers[i]) {
			t.Errorf("expected %v got %v", headers[i], h)
		}
	}

	// Make sure exchanged headers between peers are the consistent
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
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
