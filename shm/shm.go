// Package shm provides a neat way of storing key-val data.
// The hash map contains one or multiple sub-trees and each sub-tree contains multiple nodes.
// Each node has its own properties and keeps its own value. Although sub-tree hash maps
// is very simple data structure on top of Go maps it's so powerful.
package shm

import (
	"sync"
)

type node struct {
	val   []byte `json:"val"`
	Props kvs    `json:"props"`
}

type kvs map[string]string

type subtree map[string]*node

// Map stores all the subtrees.
type Map map[string]subtree

var mx sync.RWMutex

// New creates a new subtree hash map.
func New() Map {
	m := make(Map)
	return m
}

// Subtree returns specified subtree from current hash map. A subtree can be iterated like a normal Go map.
func (m Map) Subtree(key string) subtree {
	mx.Lock()
	defer mx.Unlock()

	if s, ok := m[key]; ok && s != nil {
		return s
	}
	m[key] = make(subtree)
	return m[key]
}

// SubtreeOk returns specified subtree from current hash map, it also returns ok flag which indicates nodes existence.
func (m Map) SubtreeOk(key string) (s subtree, ok bool) {
	mx.RLock()
	defer mx.RUnlock()

	s, ok = m[key]
	return
}

// DelSubtree deletes specified subtree from current hash map.
func (m Map) DelSubtree(key string) Map {
	mx.Lock()
	defer mx.Unlock()

	delete(m, key)
	return m
}

// Node returns specified node from current subtree or it creates an empty node if node doesn't exist.
func (s subtree) Node(key string) *node {
	mx.Lock()
	defer mx.Unlock()

	if n, ok := s[key]; ok && n != nil {
		return n
	}

	s[key] = &node{Props: make(map[string]string)}
	return s[key]
}

// NodeOk returns specified node from current subtree, it also returns ok flag which indicates nodes existence.
func (s subtree) NodeOk(key string) (n *node, ok bool) {
	mx.RLock()
	defer mx.RUnlock()

	n, ok = s[key]
	return
}

// DelNode deletes a node from current subtree.
func (s subtree) DelNode(key string) subtree {
	mx.Lock()
	defer mx.Unlock()

	delete(s, key)
	return s
}

// SetVal sets the value of current node.
func (n *node) SetVal(val []byte) *node {
	mx.Lock()
	defer mx.Unlock()

	n.val = val
	return n
}

// String casts val to string.
func (n *node) String() string {
	mx.RLock()
	defer mx.RUnlock()

	return string(n.val)
}

// Returns val.
func (n *node) Val() []byte {
	mx.RLock()
	defer mx.RUnlock()

	return n.val
}

// SetProps sets properties from a map
func (n *node) SetProps(props map[string]string) kvs {
	mx.Lock()
	defer mx.Unlock()

	for key, val := range props {
		n.Props[key] = val
	}
	return n.Props
}

// Sets a new property or replaces it with new one.
func (kv kvs) Set(key, val string) kvs {
	mx.Lock()
	defer mx.Unlock()

	kv[key] = val
	return kv
}

// Get returns a property.
func (kv kvs) Get(key string) string {
	mx.RLock()
	defer mx.RUnlock()

	return kv[key]
}

// GetOk returns a property and ok flag indicating if it really exists or not.
func (kv kvs) GetOk(key string) (val string, ok bool) {
	mx.RLock()
	defer mx.RUnlock()

	val, ok = kv[key]
	return
}

// Deletes a property.
func (kv kvs) Del(key string) {
	mx.Lock()
	defer mx.Unlock()

	delete(kv, key)
}
