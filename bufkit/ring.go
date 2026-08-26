// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bufkit

import (
	"sync/atomic"
)

// Ring represents a lock-free, cacheline-aligned generic circular ring buffer.
// It is optimized for ultra-low latency single-producer single-consumer (SPSC) pipelines.
type Ring[T any] struct {
	_        [56]byte // Cacheline padding to prevent false sharing
	head     atomic.Uint64
	_        [56]byte // Cacheline padding
	tail     atomic.Uint64
	_        [56]byte // Cacheline padding
	mask     uint64
	capacity uint64
	slots    []T
}

// NewRing instantiates a new [Ring] buffer. The capacity is automatically rounded
// up to the nearest power of two to allow fast bitwise modulo operations.
func NewRing[T any](capacity int) *Ring[T] {
	if capacity < 2 {
		capacity = 2
	}

	capPow2 := uint64(1)
	for capPow2 < uint64(capacity) {
		capPow2 <<= 1
	}

	return &Ring[T]{
		mask:     capPow2 - 1,
		capacity: capPow2,
		slots:    make([]T, capPow2),
	}
}

// Cap returns the maximum capacity of the ring buffer.
func (r *Ring[T]) Cap() int {
	return int(r.capacity)
}

// Len returns the instantaneous number of elements currently stored in the ring buffer.
func (r *Ring[T]) Len() int {
	head := r.head.Load()
	tail := r.tail.Load()
	if tail >= head {
		return int(tail - head)
	}
	return 0
}

// Push enqueues an element into the ring buffer. Returns false if the buffer is full.
func (r *Ring[T]) Push(val T) bool {
	tail := r.tail.Load()
	head := r.head.Load()

	if tail-head >= r.capacity {
		return false
	}

	r.slots[tail&r.mask] = val
	r.tail.Store(tail + 1)
	return true
}

// Pop dequeues an element from the ring buffer. Returns false if the buffer is empty.
func (r *Ring[T]) Pop() (T, bool) {
	head := r.head.Load()
	tail := r.tail.Load()

	if head >= tail {
		var zero T
		return zero, false
	}

	val := r.slots[head&r.mask]
	var zero T
	r.slots[head&r.mask] = zero // Clear reference for GC
	r.head.Store(head + 1)
	return val, true
}

// Reset clears all items in the ring buffer.
func (r *Ring[T]) Reset() {
	var zero T
	for i := range r.slots {
		r.slots[i] = zero
	}
	r.head.Store(0)
	r.tail.Store(0)
}
