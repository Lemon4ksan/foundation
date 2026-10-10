// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ringbuffer

import "iter"

// Buffer implements a dynamically sizing, circular FIFO queue that reuses its underlying backing storage to minimize allocations.
//
// The zero value for Buffer is an uninitialized but valid queue ready for use.
// It is not safe for concurrent use.
type Buffer[T any] struct {
	ring    []T // len is the number of queued elements; cap is the storage size
	headPos int // index of the first element in the full backing storage
}

// Init preallocates the buffer's backing storage to accommodate the specified capacity.
//
// Complexity: O(N) time where N is the requested size; allocates memory.
func (r *Buffer[T]) Init(size int) {
	r.ring = make([]T, 0, size)
	r.headPos = 0
}

// Len returns the current number of elements residing in the ring buffer.
//
// Complexity: O(1) time; zero-allocation.
func (r *Buffer[T]) Len() int {
	return len(r.ring)
}

// Empty reports whether the [Buffer] is currently devoid of elements.
//
// Complexity: O(1) time; zero-allocation.
func (r *Buffer[T]) Empty() bool {
	return len(r.ring) == 0
}

// PushBack appends a new element to the tail of the queue.
//
// If the underlying storage is at full capacity, it automatically expands prior to insertion.
//
// Complexity: Amortized O(1) time. Allocates memory exclusively during storage expansion.
func (r *Buffer[T]) PushBack(t T) {
	if len(r.ring) == cap(r.ring) {
		r.grow()
	}

	tailPos := r.headPos + len(r.ring)
	if tailPos >= cap(r.ring) {
		// both values in the sum are smaller than cap, so one subtraction wraps the index
		tailPos -= cap(r.ring)
	}

	// r.ring's length is the element count, so tailPos may be beyond it after a wrap.
	// Grow the length before temporarily exposing the whole backing array; this order
	// lets the compiler eliminate a bounds check.
	r.ring = r.ring[:len(r.ring)+1]
	r.ring[:cap(r.ring)][tailPos] = t
}

// PopFront removes and returns the element at the head of the queue.
//
// It will panic if invoked on an empty [Buffer].
//
// Complexity: O(1) time; zero-allocation.
func (r *Buffer[T]) PopFront() T {
	if r.Empty() {
		panic("ringbuffer: pop from an empty queue")
	}

	// Shrink first. The backing storage remains accessible through cap,
	// and this ordering avoids an extra bounds check.
	r.ring = r.ring[:len(r.ring)-1]
	t := r.ring[:cap(r.ring)][r.headPos]
	clear(r.ring[:cap(r.ring)][r.headPos : r.headPos+1])

	r.headPos++
	if r.headPos == cap(r.ring) {
		r.headPos = 0
	}

	return t
}

// PeekFront retrieves the element at the head of the queue without removing it.
//
// It will panic if invoked on an empty [Buffer].
//
// Complexity: O(1) time; zero-allocation.
func (r *Buffer[T]) PeekFront() T {
	if r.Empty() {
		panic("ringbuffer: peek from an empty queue")
	}

	return r.ring[:cap(r.ring)][r.headPos]
}

// grow doubles the storage and copies the queued elements into FIFO order.
func (r *Buffer[T]) grow() {
	oldRing := r.ring

	newSize := cap(oldRing) * 2
	if newSize == 0 {
		newSize = 1
	}

	newRing := make([]T, len(oldRing), newSize)
	headLen := copy(newRing, oldRing[r.headPos:])
	copy(newRing[headLen:], oldRing[:r.headPos])
	r.ring = newRing
	r.headPos = 0
}

// Clear forcefully purges all queued elements while retaining the underlying allocated storage for future reuse.
//
// Complexity: O(N) time where N is the storage capacity; zero-allocation.
func (r *Buffer[T]) Clear() {
	clear(r.ring[:cap(r.ring)])
	r.ring = r.ring[:0]
	r.headPos = 0
}

// All provides a sequence iterator traversing the queued elements in FIFO order without mutating the [Buffer].
//
// Complexity: O(1) time to instantiate the iterator; zero-allocation.
func (r *Buffer[T]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		n := len(r.ring)
		c := cap(r.ring)
		if n == 0 || c == 0 {
			return
		}
		ringCap := r.ring[:c]
		for i := range n {
			pos := r.headPos + i
			if pos >= c {
				pos -= c
			}
			if !yield(ringCap[pos]) {
				return
			}
		}
	}
}

// Values provides a sequence iterator traversing the queued elements in FIFO order without mutating the [Buffer].
//
// This is an alias for [Buffer.All].
// Complexity: O(1) time to instantiate the iterator; zero-allocation.
func (r *Buffer[T]) Values() iter.Seq[T] {
	return r.All()
}

// Drain provides a mutating sequence iterator that sequentially pops and yields elements in FIFO order until the [Buffer] is exhausted.
//
// Complexity: O(1) time per yielded element; zero-allocation.
func (r *Buffer[T]) Drain() iter.Seq[T] {
	return func(yield func(T) bool) {
		for !r.Empty() {
			item := r.PopFront()
			if !yield(item) {
				return
			}
		}
	}
}

// Backward provides a sequence iterator traversing the queued elements in reverse FIFO (LIFO) order without mutating the [Buffer].
//
// Complexity: O(1) time to instantiate the iterator; zero-allocation.
func (r *Buffer[T]) Backward() iter.Seq[T] {
	return func(yield func(T) bool) {
		n := len(r.ring)
		c := cap(r.ring)
		if n == 0 || c == 0 {
			return
		}
		ringCap := r.ring[:c]
		for i := n - 1; i >= 0; i-- {
			pos := r.headPos + i
			if pos >= c {
				pos -= c
			}
			if !yield(ringCap[pos]) {
				return
			}
		}
	}
}
