// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package deque implements a high-performance, generic double-ended circular queue (deque).
// Elements can be pushed and popped from both front and back in amortized O(1) time
// with zero heap allocations on steady-state operations.
package deque

import "iter"

// Deque represents a generic double-ended circular queue backed by a contiguous slice.
// The zero value of Deque is empty and ready to use.
//
// Read-only operations are safe for concurrent access across multiple goroutines.
// Concurrent mutations or concurrent read/writes must be externally synchronized.
type Deque[T any] struct {
	buf  []T
	head int // physical index in buf of the front element
	len  int // number of queued elements
}

// New creates and returns an empty Deque.
// Backing storage is lazily allocated on the first push operation.
func New[T any]() *Deque[T] {
	return &Deque[T]{}
}

// NewWithCapacity creates and returns an empty Deque with preallocated backing capacity.
// Panics if capacity < 0.
func NewWithCapacity[T any](capacity int) *Deque[T] {
	if capacity < 0 {
		panic("deque: negative capacity")
	}
	var buf []T
	if capacity > 0 {
		buf = make([]T, capacity)
	}
	return &Deque[T]{
		buf: buf,
	}
}

// Len returns the number of elements currently stored in the deque.
func (d *Deque[T]) Len() int {
	if d == nil {
		return 0
	}
	return d.len
}

// Cap returns the total capacity of the backing slice without reallocating.
func (d *Deque[T]) Cap() int {
	if d == nil {
		return 0
	}
	return len(d.buf)
}

// Empty reports whether the deque contains zero elements.
func (d *Deque[T]) Empty() bool {
	return d == nil || d.len == 0
}

// Clear removes all elements from the deque while retaining the allocated storage.
// All evacuated slots are cleared to allow garbage collection of referenced objects.
func (d *Deque[T]) Clear() {
	if d == nil {
		return
	}
	clear(d.buf)
	d.head = 0
	d.len = 0
}

// PushBack appends elem to the back of the deque in amortized O(1) time.
func (d *Deque[T]) PushBack(elem T) {
	if d.len == len(d.buf) {
		d.grow()
	}
	idx := d.head + d.len
	if idx >= len(d.buf) {
		idx -= len(d.buf)
	}
	d.buf[idx] = elem
	d.len++
}

// PushFront prepends elem to the front of the deque in amortized O(1) time.
func (d *Deque[T]) PushFront(elem T) {
	if d.len == len(d.buf) {
		d.grow()
	}
	if d.head == 0 {
		d.head = len(d.buf) - 1
	} else {
		d.head--
	}
	d.buf[d.head] = elem
	d.len++
}

// PopFront removes and returns the front element from the deque.
// If the deque is empty, it returns the zero value of T and false.
func (d *Deque[T]) PopFront() (T, bool) {
	if d == nil || d.len == 0 {
		var zero T
		return zero, false
	}
	elem := d.buf[d.head]
	var zero T
	d.buf[d.head] = zero

	d.head++
	if d.head == len(d.buf) {
		d.head = 0
	}
	d.len--
	return elem, true
}

// PopBack removes and returns the back element from the deque.
// If the deque is empty, it returns the zero value of T and false.
func (d *Deque[T]) PopBack() (T, bool) {
	if d == nil || d.len == 0 {
		var zero T
		return zero, false
	}
	backIdx := d.head + d.len - 1
	if backIdx >= len(d.buf) {
		backIdx -= len(d.buf)
	}
	elem := d.buf[backIdx]
	var zero T
	d.buf[backIdx] = zero
	d.len--
	return elem, true
}

// MustPopFront removes and returns the front element.
// Panics if the deque is empty.
func (d *Deque[T]) MustPopFront() T {
	elem, ok := d.PopFront()
	if !ok {
		panic("deque: pop from an empty deque")
	}
	return elem
}

// MustPopBack removes and returns the back element.
// Panics if the deque is empty.
func (d *Deque[T]) MustPopBack() T {
	elem, ok := d.PopBack()
	if !ok {
		panic("deque: pop from an empty deque")
	}
	return elem
}

// Front returns the front element without removing it.
// If the deque is empty, it returns the zero value of T and false.
func (d *Deque[T]) Front() (T, bool) {
	if d == nil || d.len == 0 {
		var zero T
		return zero, false
	}
	return d.buf[d.head], true
}

// Back returns the back element without removing it.
// If the deque is empty, it returns the zero value of T and false.
func (d *Deque[T]) Back() (T, bool) {
	if d == nil || d.len == 0 {
		var zero T
		return zero, false
	}
	backIdx := d.head + d.len - 1
	if backIdx >= len(d.buf) {
		backIdx -= len(d.buf)
	}
	return d.buf[backIdx], true
}

// At returns the element at logical index i.
// Index 0 represents the front; len-1 represents the back.
// Negative indices index from the back: -1 represents the back element, -len represents the front.
// Panics if i < -len or i >= len.
func (d *Deque[T]) At(i int) T {
	if d == nil || i < -d.len || i >= d.len {
		panic("deque: index out of range")
	}
	if i < 0 {
		i += d.len
	}
	idx := d.head + i
	if idx >= len(d.buf) {
		idx -= len(d.buf)
	}
	return d.buf[idx]
}

// Set replaces the element at logical index i with elem.
// Supports negative indices (-1 is back, -len is front).
// Panics if i < -len or i >= len.
func (d *Deque[T]) Set(i int, elem T) {
	if d == nil || i < -d.len || i >= d.len {
		panic("deque: index out of range")
	}
	if i < 0 {
		i += d.len
	}
	idx := d.head + i
	if idx >= len(d.buf) {
		idx -= len(d.buf)
	}
	d.buf[idx] = elem
}

// Values returns an inlined push iterator over queued elements from front to back without mutating the deque.
func (d *Deque[T]) Values() iter.Seq[T] {
	return func(yield func(T) bool) {
		if d == nil || d.len == 0 {
			return
		}
		c := len(d.buf)
		for i := 0; i < d.len; i++ {
			idx := d.head + i
			if idx >= c {
				idx -= c
			}
			if !yield(d.buf[idx]) {
				return
			}
		}
	}
}

// All returns an inlined push iterator over queued elements yielding logical index (0 to len-1) and element.
func (d *Deque[T]) All() iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		if d == nil || d.len == 0 {
			return
		}
		c := len(d.buf)
		for i := 0; i < d.len; i++ {
			idx := d.head + i
			if idx >= c {
				idx -= c
			}
			if !yield(i, d.buf[idx]) {
				return
			}
		}
	}
}

// Backward returns an inlined push iterator over queued elements from back to front without mutating the deque.
func (d *Deque[T]) Backward() iter.Seq[T] {
	return func(yield func(T) bool) {
		if d == nil || d.len == 0 {
			return
		}
		c := len(d.buf)
		for i := d.len - 1; i >= 0; i-- {
			idx := d.head + i
			if idx >= c {
				idx -= c
			}
			if !yield(d.buf[idx]) {
				return
			}
		}
	}
}

// Drain returns a consuming iterator that pops elements from front to back until the deque is empty or early break.
func (d *Deque[T]) Drain() iter.Seq[T] {
	return func(yield func(T) bool) {
		for d != nil && d.len > 0 {
			elem, _ := d.PopFront()
			if !yield(elem) {
				return
			}
		}
	}
}

// grow doubles the buffer capacity and unwraps elements into contiguous FIFO order.
func (d *Deque[T]) grow() {
	newCap := len(d.buf) * 2
	if newCap < 4 {
		newCap = 4
	}
	newBuf := make([]T, newCap)
	n := copy(newBuf, d.buf[d.head:])
	copy(newBuf[n:], d.buf[:d.head])
	d.buf = newBuf
	d.head = 0
}
