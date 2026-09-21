// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package minheap provides a high-performance, generic binary minimum heap ordered by key.
package minheap

import (
	"cmp"
	"iter"
)

type entry[K, V any] struct {
	key   K
	value V
}

// A Heap is a minimum heap ordered by key.
type Heap[K cmp.Ordered, V any] []entry[K, V]

// Len returns the number of elements currently in the heap.
func (h Heap[K, V]) Len() int { return len(h) }

// Empty reports whether the heap contains zero elements.
func (h Heap[K, V]) Empty() bool { return len(h) == 0 }

// Peek returns the smallest element in the heap.
// It must not be called when the heap is empty.
func (h Heap[K, V]) Peek() (K, V) {
	v := h[0]
	return v.key, v.value
}

// Push adds an element to the heap.
func (h *Heap[K, V]) Push(key K, value V) {
	*h = append(*h, entry[K, V]{key: key, value: value})
	h.up(len(*h) - 1)
}

// Pop removes and returns the smallest element in the heap.
// It must not be called when the heap is empty.
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

// Clear removes all elements from the heap.
func (h *Heap[K, V]) Clear() {
	clear(*h)
	*h = (*h)[:0]
}

// All yields all key-value entries stored in the heap without consuming elements.
func (h Heap[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for i := range h {
			if !yield(h[i].key, h[i].value) {
				return
			}
		}
	}
}

// Values yields all values stored in the heap without consuming elements.
func (h Heap[K, V]) Values() iter.Seq[V] {
	return func(yield func(V) bool) {
		for i := range h {
			if !yield(h[i].value) {
				return
			}
		}
	}
}

// Drain pops and yields elements in ascending key priority order until the heap is empty.
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

// DrainValues pops and yields values in ascending key priority order until the heap is empty.
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
