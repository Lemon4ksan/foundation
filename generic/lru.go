// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generic

import (
	"sync"
)

type lruNode[K comparable, V any] struct {
	key  K
	val  V
	prev *lruNode[K, V]
	next *lruNode[K, V]
}

// LRU implements a thread-safe, generic Least-Recently-Used (LRU) cache with O(1) Get and Put.
type LRU[K comparable, V any] struct {
	mu       sync.RWMutex
	capacity int
	items    map[K]*lruNode[K, V]
	head     *lruNode[K, V]
	tail     *lruNode[K, V]
}

// NewLRU creates a new [LRU] cache with the specified capacity limit.
func NewLRU[K comparable, V any](capacity int) *LRU[K, V] {
	if capacity <= 0 {
		capacity = 128
	}

	l := &LRU[K, V]{
		capacity: capacity,
		items:    make(map[K]*lruNode[K, V], capacity),
		head:     &lruNode[K, V]{},
		tail:     &lruNode[K, V]{},
	}
	l.head.next = l.tail
	l.tail.prev = l.head
	return l
}

// Get retrieves a value by key and marks it as most recently used.
func (l *LRU[K, V]) Get(key K) (V, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	node, exists := l.items[key]
	if !exists {
		var zero V
		return zero, false
	}

	l.moveToHead(node)
	return node.val, true
}

// Put inserts or updates a key-value pair. If capacity is exceeded, the least recently used item is evicted.
func (l *LRU[K, V]) Put(key K, val V) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if node, exists := l.items[key]; exists {
		node.val = val
		l.moveToHead(node)
		return
	}

	if len(l.items) >= l.capacity {
		l.evictTail()
	}

	newNode := &lruNode[K, V]{
		key: key,
		val: val,
	}
	l.items[key] = newNode
	l.addToHead(newNode)
}

// Delete removes a key from the cache.
func (l *LRU[K, V]) Delete(key K) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	node, exists := l.items[key]
	if !exists {
		return false
	}

	l.removeNode(node)
	delete(l.items, key)
	return true
}

// Len returns the number of items currently in the cache.
func (l *LRU[K, V]) Len() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.items)
}

// Cap returns the maximum capacity of the cache.
func (l *LRU[K, V]) Cap() int {
	return l.capacity
}

func (l *LRU[K, V]) addToHead(node *lruNode[K, V]) {
	node.prev = l.head
	node.next = l.head.next
	l.head.next.prev = node
	l.head.next = node
}

func (l *LRU[K, V]) removeNode(node *lruNode[K, V]) {
	node.prev.next = node.next
	node.next.prev = node.prev
}

func (l *LRU[K, V]) moveToHead(node *lruNode[K, V]) {
	l.removeNode(node)
	l.addToHead(node)
}

func (l *LRU[K, V]) evictTail() {
	lastNode := l.tail.prev
	if lastNode == l.head {
		return
	}
	l.removeNode(lastNode)
	delete(l.items, lastNode.key)
}
