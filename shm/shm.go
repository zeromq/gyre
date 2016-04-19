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
	props *kvs   `json:"props"`
	sync.RWMutex
}

type kvs struct {
	m map[string]string
	sync.RWMutex
}

type subtree struct {
	m map[string]*node
	sync.RWMutex
}

// Map stores all the subtrees.
type Map struct {
	m map[string]*subtree
	sync.RWMutex
}

// New creates a new subtree hash map.
func New() *Map {
	return &Map{m: make(map[string]*subtree)}
}

// Subtree returns specified subtree from current hash map. A subtree can be iterated like a normal Go map.
func (m *Map) Subtree(key string) *subtree {
	m.Lock()
	defer m.Unlock()

	if s, ok := m.m[key]; ok && s != nil {
		return s
	}
	m.m[key] = &subtree{m: make(map[string]*node)}
	return m.m[key]
}

// SubtreeOk returns specified subtree from current hash map, it also returns ok flag which indicates nodes existence.
func (m *Map) SubtreeOk(key string) (s *subtree, ok bool) {
	m.RLock()
	defer m.RUnlock()

	s, ok = m.m[key]
	return
}

// DelSubtree deletes specified subtree from current hash map.
func (m *Map) DelSubtree(key string) *Map {
	m.Lock()
	defer m.Unlock()

	delete(m.m, key)
	return m
}

// Node returns specified node from current subtree or it creates an empty node if node doesn't exist.
func (s *subtree) Node(key string) *node {
	s.Lock()
	defer s.Unlock()

	if n, ok := s.m[key]; ok && n != nil {
		return n
	}

	s.m[key] = &node{props: &kvs{m: make(map[string]string)}}
	return s.m[key]
}

// NodeOk returns specified node from current subtree, it also returns ok flag which indicates nodes existence.
func (s *subtree) NodeOk(key string) (n *node, ok bool) {
	s.RLock()
	defer s.RUnlock()

	n, ok = s.m[key]
	return
}

// DelNode deletes a node from current subtree.
func (s *subtree) DelNode(key string) *subtree {
	s.Lock()
	defer s.Unlock()

	delete(s.m, key)
	return s
}

// SetVal sets the value of current node.
func (n *node) SetVal(val []byte) *node {
	n.Lock()
	defer n.Unlock()

	n.val = val
	return n
}

// String casts val to string.
func (n *node) String() string {
	n.RLock()
	defer n.RUnlock()

	return string(n.val)
}

// Returns val.
func (n *node) Val() []byte {
	n.RLock()
	defer n.RUnlock()

	return n.val
}

// SetProps sets properties from a map
func (n *node) SetProps(props map[string]string) *kvs {
	n.Lock()
	defer n.Unlock()

	for key, val := range props {
		n.props.m[key] = val
	}

	return n.props
}

// Props returns properties
func (n *node) Props() *kvs {
	n.RLock()
	defer n.RUnlock()

	return n.props
}

// Sets a new property or replaces it with new one.
func (kv *kvs) Set(key, val string) *kvs {
	kv.Lock()
	defer kv.Unlock()

	kv.m[key] = val
	return kv
}

// Get returns a property.
func (kv *kvs) Get(key string) string {
	kv.RLock()
	defer kv.RUnlock()

	return kv.m[key]
}

// GetOk returns a property and ok flag indicating if it really exists or not.
func (kv *kvs) GetOk(key string) (val string, ok bool) {
	kv.RLock()
	defer kv.RUnlock()

	val, ok = kv.m[key]
	return
}

// Deletes a property.
func (kv kvs) Del(key string) {
	kv.Lock()
	defer kv.Unlock()

	delete(kv.m, key)
}
