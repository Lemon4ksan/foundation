// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bufkit

import (
	"iter"
	"math/bits"
	"sync/atomic"

	"github.com/lemon4ksan/foundation/silicon/memkit"
)

// Ring represents a lock-free, cacheline-aligned generic circular ring buffer.
// It is optimized for ultra-low latency single-producer single-consumer (SPSC) pipelines.
//
// Mechanical Sympathy & Cache Coherence:
// Standard SPSC queues experience severe cross-core cache invalidation traffic if the producer
// and consumer constantly poll each other's modified atomic variables. This implementation maintains
// producer-local cachedHead and consumer-local cachedTail variables, keeping cacheline ownership
// private to each core during sustained streaming and eliminating bus contention.
type Ring[T any] struct {
	_ memkit.NoCopy

	// Producer cacheline
	tail       atomic.Uint64
	cachedHead uint64
	_          memkit.CacheLinePad

	// Consumer cacheline
	head       atomic.Uint64
	cachedTail uint64
	_          memkit.CacheLinePad

	// Read-only shared cacheline
	mask     uint64
	capacity uint64
	slots    []T
}

// NewRing returns a new [Ring] buffer with capacity rounded up to the nearest power of two
// to replace modulo division with fast bitwise masking.
func NewRing[T any](capacity int) *Ring[T] {
	if capacity < 2 {
		capacity = 2
	}

	var capPow2 uint64
	if (capacity & (capacity - 1)) != 0 {
		capPow2 = 1 << bits.Len(uint(capacity-1))
	} else {
		capPow2 = uint64(capacity)
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
	return int(r.tail.Load() - r.head.Load())
}

// Free returns the instantaneous number of available slots for writing in the ring buffer.
func (r *Ring[T]) Free() int {
	return r.Cap() - r.Len()
}

// IsEmpty reports whether the ring buffer contains no elements.
func (r *Ring[T]) IsEmpty() bool {
	return r.Len() == 0
}

// IsFull reports whether the ring buffer is at capacity.
func (r *Ring[T]) IsFull() bool {
	return r.Len() >= int(r.capacity)
}

// Push enqueues an element into the ring buffer, returning false if the buffer is full.
func (r *Ring[T]) Push(val T) bool {
	tail := r.tail.Load()
	if tail-r.cachedHead >= r.capacity {
		r.cachedHead = r.head.Load()
		if tail-r.cachedHead >= r.capacity {
			return false
		}
	}

	r.slots[tail&r.mask] = val
	r.tail.Store(tail + 1)
	return true
}

// Pop dequeues an element from the ring buffer, zeroing the underlying slot for GC,
// and returns (val, true) or (zero, false) if the buffer is empty.
func (r *Ring[T]) Pop() (T, bool) {
	head := r.head.Load()
	if head >= r.cachedTail {
		r.cachedTail = r.tail.Load()
		if head >= r.cachedTail {
			var zero T
			return zero, false
		}
	}

	slotIdx := head & r.mask
	val := r.slots[slotIdx]
	var zero T
	r.slots[slotIdx] = zero // Clear reference for GC
	r.head.Store(head + 1)
	return val, true
}

// Peek returns the next element in the ring without removing it, or (zero, false) if empty.
func (r *Ring[T]) Peek() (T, bool) {
	head := r.head.Load()
	if head >= r.cachedTail {
		r.cachedTail = r.tail.Load()
		if head >= r.cachedTail {
			var zero T
			return zero, false
		}
	}
	return r.slots[head&r.mask], true
}

// PushBatch enqueues multiple elements in a single batch operation and returns the number of items pushed.
func (r *Ring[T]) PushBatch(items []T) int {
	if len(items) == 0 {
		return 0
	}

	tail := r.tail.Load()
	avail := int(r.capacity - (tail - r.cachedHead))
	if avail < len(items) {
		r.cachedHead = r.head.Load()
		avail = int(r.capacity - (tail - r.cachedHead))
		if avail <= 0 {
			return 0
		}
	}

	toPush := min(len(items), avail)
	start := int(tail & r.mask)
	firstBatch := min(toPush, int(r.capacity)-start)

	copy(r.slots[start:], items[:firstBatch])
	if toPush > firstBatch {
		copy(r.slots[:toPush-firstBatch], items[firstBatch:toPush])
	}

	r.tail.Store(tail + uint64(toPush))
	return toPush
}

// PopBatch dequeues multiple elements into dst in a single batch operation and returns the number of items popped.
// Source slots are aggressively zeroed to allow GC collection of generic elements.
func (r *Ring[T]) PopBatch(dst []T) int {
	if len(dst) == 0 {
		return 0
	}

	head := r.head.Load()
	avail := int(r.cachedTail - head)
	if avail < len(dst) {
		r.cachedTail = r.tail.Load()
		avail = int(r.cachedTail - head)
		if avail <= 0 {
			return 0
		}
	}

	toPop := min(len(dst), avail)
	start := int(head & r.mask)
	firstBatch := min(toPop, int(r.capacity)-start)

	copy(dst[:firstBatch], r.slots[start:start+firstBatch])
	var zero T
	for i := start; i < start+firstBatch; i++ {
		r.slots[i] = zero
	}

	if toPop > firstBatch {
		remaining := toPop - firstBatch
		copy(dst[firstBatch:toPop], r.slots[:remaining])
		for i := range remaining {
			r.slots[i] = zero
		}
	}

	r.head.Store(head + uint64(toPop))
	return toPop
}

// Reset clears all items in the ring buffer, zeroes all slot references for GC, and resets indices.
func (r *Ring[T]) Reset() {
	var zero T
	for i := range r.slots {
		r.slots[i] = zero
	}
	r.head.Store(0)
	r.cachedTail = 0
	r.tail.Store(0)
	r.cachedHead = 0
}

// All returns an iterator yielding logical index (0 to len-1) and element in FIFO order without popping them.
func (r *Ring[T]) All() iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		head := r.head.Load()
		tail := r.tail.Load()
		idx := 0
		for i := head; i < tail; i++ {
			if !yield(idx, r.slots[i&r.mask]) {
				return
			}
			idx++
		}
	}
}

// Values returns an iterator over elements currently in the ring buffer in FIFO order without popping them.
func (r *Ring[T]) Values() iter.Seq[T] {
	return func(yield func(T) bool) {
		head := r.head.Load()
		tail := r.tail.Load()
		for i := head; i < tail; i++ {
			if !yield(r.slots[i&r.mask]) {
				return
			}
		}
	}
}

// Drain pops and yields all items from the ring buffer in FIFO order until empty or early termination.
func (r *Ring[T]) Drain() iter.Seq[T] {
	return func(yield func(T) bool) {
		for {
			val, ok := r.Pop()
			if !ok {
				return
			}
			if !yield(val) {
				return
			}
		}
	}
}
