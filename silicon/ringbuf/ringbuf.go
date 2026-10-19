// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ringbuf

import (
	"iter"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/cpu"
)

type cell[T any] struct {
	sequence atomic.Uint64
	data     unsafe.Pointer
}

// Buffer is a lock-free power-of-two capacity MPMC queue.
type Buffer[T any] struct {
	_          cpu.CacheLinePad
	enqueuePos atomic.Uint64
	_          cpu.CacheLinePad
	dequeuePos atomic.Uint64
	_          cpu.CacheLinePad
	capacity   uint64
	mask       uint64
	buffer     []cell[T]
}

func roundUpPow2(capacity int) uint64 {
	if capacity < 2 {
		return 2
	}

	capPow2 := uint64(2)
	for capPow2 < uint64(capacity) {
		capPow2 <<= 1
	}

	return capPow2
}

// New instantiates a lock-free [Buffer] with power-of-two capacity.
func New[T any](capacity int) *Buffer[T] {
	capPow2 := roundUpPow2(capacity)

	buf := make([]cell[T], capPow2)
	for i := uint64(0); i < capPow2; i++ {
		buf[i].sequence.Store(i)
	}

	return &Buffer[T]{
		capacity: capPow2,
		mask:     capPow2 - 1,
		buffer:   buf,
	}
}

// Push enqueues item into the ring buffer using atomic CAS.
// Returns false if buffer is full.
func (r *Buffer[T]) Push(item *T) bool {
	pos := r.enqueuePos.Load()
	for {
		c := &r.buffer[pos&r.mask]
		seq := c.sequence.Load()
		dif := int64(seq) - int64(pos)

		switch {
		case dif == 0:
			if r.enqueuePos.CompareAndSwap(pos, pos+1) {
				atomic.StorePointer(&c.data, unsafe.Pointer(item))
				c.sequence.Store(pos + 1)

				return true
			}

		case dif < 0:
			return false // Buffer full
		default:
			pos = r.enqueuePos.Load()
		}
	}
}

// Pop dequeues item from the ring buffer using atomic CAS.
// Returns nil if buffer is empty.
func (r *Buffer[T]) Pop() *T {
	pos := r.dequeuePos.Load()
	for {
		c := &r.buffer[pos&r.mask]
		seq := c.sequence.Load()
		dif := int64(seq) - int64(pos+1)

		switch {
		case dif == 0:
			if r.dequeuePos.CompareAndSwap(pos, pos+1) {
				ptr := atomic.LoadPointer(&c.data)
				atomic.StorePointer(&c.data, nil)
				c.sequence.Store(pos + r.mask + 1)

				return (*T)(ptr)
			}

		case dif < 0:
			return nil // Buffer empty
		default:
			pos = r.dequeuePos.Load()
		}
	}
}

// Drain pops and yields all buffered items in FIFO order until the buffer is empty.
func (r *Buffer[T]) Drain() iter.Seq[*T] {
	return func(yield func(*T) bool) {
		for {
			item := r.Pop()
			if item == nil {
				return
			}
			if !yield(item) {
				return
			}
		}
	}
}
