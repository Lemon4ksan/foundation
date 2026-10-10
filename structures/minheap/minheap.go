// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package minheap

import (
	"cmp"
	"iter"
)

type entry[K, V any] struct {
	key   K
	value V
}

// Heap represents a minimum heap priority queue ordered by key.
//
// The zero value for Heap is an empty heap ready for use.
// It is not safe for concurrent use.
type Heap[K cmp.Ordered, V any] []entry[K, V]

// Len returns the current number of elements residing in the heap.
//
// Complexity: O(1) time; zero-allocation.
func (h Heap[K, V]) Len() int { return len(h) }

// Empty reports whether the heap is currently devoid of elements.
//
// Complexity: O(1) time; zero-allocation.
func (h Heap[K, V]) Empty() bool { return len(h) == 0 }

// Peek retrieves the smallest element in the heap without removing it.
//
// It panics if invoked on an empty heap.
// Complexity: O(1) time; zero-allocation.
func (h Heap[K, V]) Peek() (K, V) {
	v := h[0]
	return v.key, v.value
}

// Push adds a new element to the heap, maintaining the minimum heap property.
//
// Complexity: O(log N) time. Allocates memory exclusively during storage expansion.
func (h *Heap[K, V]) Push(key K, value V) {
	*h = append(*h, entry[K, V]{key: key, value: value})
	h.up(len(*h) - 1)
}

// Pop removes and returns the smallest element in the heap.
//
// It panics if invoked on an empty heap.
// Complexity: O(log N) time; zero-allocation.
func (h *Heap[K, V]) Pop() (K, V) {
	values := *h
	last := len(values) - 1
	v := values[0]
	values[0] = values[last]

	var zero entry[K, V]

	values[last] = zero

	values = values[:last]
	if last > 0 {
		values.down(0)
	}

	*h = values

	return v.key, v.value
}

// Clear forcefully purges all elements from the heap while retaining the underlying allocated storage.
//
// Complexity: O(N) time where N is the heap capacity; zero-allocation.
func (h *Heap[K, V]) Clear() {
	clear(*h)
	*h = (*h)[:0]
}

// All provides a sequence iterator traversing all key-value entries stored in the heap without consuming them.
//
// The order of elements is undefined.
// Complexity: O(1) time to instantiate the iterator; zero-allocation.
func (h Heap[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for i := range h {
			if !yield(h[i].key, h[i].value) {
				return
			}
		}
	}
}

// Values provides a sequence iterator traversing all values stored in the heap without consuming them.
//
// The order of elements is undefined.
// Complexity: O(1) time to instantiate the iterator; zero-allocation.
func (h Heap[K, V]) Values() iter.Seq[V] {
	return func(yield func(V) bool) {
		for i := range h {
			if !yield(h[i].value) {
				return
			}
		}
	}
}

// Drain provides a mutating sequence iterator that sequentially pops and yields key-value entries in ascending key priority order until the heap is exhausted.
//
// Complexity: O(log N) time per yielded element; zero-allocation.
func (h *Heap[K, V]) Drain() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for !h.Empty() {
			k, v := h.Pop()
			if !yield(k, v) {
				return
			}
		}
	}
}

// DrainValues provides a mutating sequence iterator that sequentially pops and yields values in ascending key priority order until the heap is exhausted.
//
// Complexity: O(log N) time per yielded element; zero-allocation.
func (h *Heap[K, V]) DrainValues() iter.Seq[V] {
	return func(yield func(V) bool) {
		for !h.Empty() {
			_, v := h.Pop()
			if !yield(v) {
				return
			}
		}
	}
}

func (h Heap[K, V]) up(i int) {
	for i > 0 {
		parent := (i - 1) / 2
		if h[parent].key <= h[i].key {
			return
		}

		h[parent], h[i] = h[i], h[parent]
		i = parent
	}
}

func (h Heap[K, V]) down(i int) {
	for {
		child := 2*i + 1
		if child >= len(h) {
			break
		}

		if right := child + 1; right < len(h) && h[right].key < h[child].key {
			child = right
		}

		if h[i].key <= h[child].key {
			break
		}

		h[i], h[child] = h[child], h[i]
		i = child
	}
}
