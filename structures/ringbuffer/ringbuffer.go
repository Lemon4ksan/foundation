// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ringbuffer

import "iter"

// Buffer is a growable FIFO queue that reuses its backing storage.
// The slice length stores the queue length, so wrapped positions are accessed by reslicing to capacity.
type Buffer[T any] struct {
	ring    []T // len is the number of queued elements; cap is the storage size
	headPos int // index of the first element in the full backing storage
}

// New returns an empty, initialized ring buffer.
func New[T any]() *Buffer[T] {
	return &Buffer[T]{}
}

// NewWithCapacity returns an initialized ring buffer with preallocated capacity.
func NewWithCapacity[T any](size int) *Buffer[T] {
	r := &Buffer[T]{}
	r.Init(size)
	return r
}

// Init preallocates a buffer with a certain size.
func (r *Buffer[T]) Init(size int) {
	r.ring = make([]T, 0, size)
	r.headPos = 0
}

// Len returns the number of elements in the ring buffer.
func (r *Buffer[T]) Len() int {
	return len(r.ring)
}

// Empty says if the ring buffer is empty.
func (r *Buffer[T]) Empty() bool {
	return len(r.ring) == 0
}

// PushBack adds a new element.
// If the ring buffer is full, its capacity is increased first.
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

// PopFront removes and returns the next element, or (zero, false) if the buffer is empty.
func (r *Buffer[T]) PopFront() (T, bool) {
	if r.Empty() {
		var zero T
		return zero, false
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

	return t, true
}

// MustPopFront removes and returns the next element, or panics if the buffer is empty.
func (r *Buffer[T]) MustPopFront() T {
	t, ok := r.PopFront()
	if !ok {
		panic("ringbuffer: pop from an empty queue")
	}
	return t
}

// PeekFront returns the next element without removing it, or (zero, false) if the buffer is empty.
func (r *Buffer[T]) PeekFront() (T, bool) {
	if r.Empty() {
		var zero T
		return zero, false
	}

	return r.ring[:cap(r.ring)][r.headPos], true
}

// MustPeekFront returns the next element without removing it, or panics if the buffer is empty.
func (r *Buffer[T]) MustPeekFront() T {
	t, ok := r.PeekFront()
	if !ok {
		panic("ringbuffer: peek from an empty queue")
	}
	return t
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

// Clear removes all elements while retaining the allocated storage.
func (r *Buffer[T]) Clear() {
	clear(r.ring[:cap(r.ring)])
	r.ring = r.ring[:0]
	r.headPos = 0
}

// All returns an iterator over index-element pairs in FIFO order without mutating the ring buffer.
func (r *Buffer[T]) All() iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
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
			if !yield(i, ringCap[pos]) {
				return
			}
		}
	}
}

// Values returns an iterator over queued elements in FIFO order without mutating the ring buffer.
func (r *Buffer[T]) Values() iter.Seq[T] {
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

// Drain pops and yields all buffered items in FIFO order until the buffer is empty.
func (r *Buffer[T]) Drain() iter.Seq[T] {
	return func(yield func(T) bool) {
		for !r.Empty() {
			item, _ := r.PopFront()
			if !yield(item) {
				return
			}
		}
	}
}

// Backward returns an iterator over queued elements in reverse order (LIFO) without mutating the ring buffer.
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
