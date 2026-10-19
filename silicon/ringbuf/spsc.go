// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ringbuf

import (
	"iter"
	"sync/atomic"

	"golang.org/x/sys/cpu"
)

// SPSC implements a Single-Producer Single-Consumer lock-free ring buffer.
//
// Memory Architecture & Hardware Alignment:
//   - Zero CAS (CompareAndSwap) Operations: Eliminates atomic CAS loop contention.
//   - Local Cursor Caching: Producer caches consumer tail (cachedTail) and consumer
//     caches producer head (cachedHead) to eliminate cache-line bouncing across CPU cores
//     until buffer boundaries (full/empty) require cursor refreshes.
//   - Cache Line Padding: Producer state (head, cachedTail) and Consumer state (tail, cachedHead)
//     are separated by 64-byte cpu.CacheLinePad boundaries to prevent False Sharing across CPU cores.
//   - Bitwise Masking: Buffer capacity is rounded up to the nearest power of two,
//     enabling 1-cycle bitwise AND (& mask) indexing instead of costly division (% cap).
type SPSC[T any] struct {
	_          cpu.CacheLinePad
	head       atomic.Uint64 // Written ONLY by Producer thread
	cachedTail uint64        // Local to Producer thread; refreshed from tail only when queue appears full
	_          [2]cpu.CacheLinePad
	tail       atomic.Uint64 // Written ONLY by Consumer thread
	cachedHead uint64        // Local to Consumer thread; refreshed from head only when queue appears empty
	_          [2]cpu.CacheLinePad

	mask   uint64
	buffer []atomic.Pointer[T]
}

// NewSPSC constructs an [SPSC] ring buffer with the requested capacity.
// Capacity is automatically rounded up to the next power of two.
func NewSPSC[T any](capacity int) *SPSC[T] {
	capPow2 := roundUpPow2(capacity)

	return &SPSC[T]{
		mask:   capPow2 - 1,
		buffer: make([]atomic.Pointer[T], capPow2),
	}
}

// Push enqueues item into the ring buffer.
//
// Thread Safety Invariant: Must be invoked from EXACTLY ONE Producer goroutine.
// Returns false if the buffer is full.
func (s *SPSC[T]) Push(item *T) bool {
	head := s.head.Load()

	if head-s.cachedTail > s.mask {
		s.cachedTail = s.tail.Load()
		if head-s.cachedTail > s.mask {
			return false // Buffer is full
		}
	}

	idx := head & s.mask
	_ = &s.buffer[idx]
	s.buffer[idx].Store(item)
	s.head.Store(head + 1)

	return true
}

// Pop dequeues item from the ring buffer.
//
// Thread Safety Invariant: Must be invoked from EXACTLY ONE Consumer goroutine.
// Returns nil if the buffer is empty.
func (s *SPSC[T]) Pop() *T {
	tail := s.tail.Load()

	if s.cachedHead == tail {
		s.cachedHead = s.head.Load()
		if s.cachedHead == tail {
			return nil // Buffer is empty
		}
	}

	idx := tail & s.mask
	_ = &s.buffer[idx]
	item := s.buffer[idx].Load()
	s.buffer[idx].Store(nil)
	s.tail.Store(tail + 1)

	return item
}

// Len returns the current number of items buffered in the queue.
func (s *SPSC[T]) Len() int {
	head := s.head.Load()
	tail := s.tail.Load()

	return int(head - tail)
}

// Clear removes and drops all elements from the ring buffer, zeroing all pointer slots.
//
// Thread Safety Invariant: Must be invoked from EXACTLY ONE Consumer goroutine.
func (s *SPSC[T]) Clear() {
	for s.Pop() != nil {
	}
}

// Cap returns the total power-of-two storage capacity of the ring buffer.
func (s *SPSC[T]) Cap() int {
	return int(s.mask + 1)
}

// IsFull reports whether the ring buffer cannot accept more items without popping.
func (s *SPSC[T]) IsFull() bool {
	return s.Len() >= s.Cap()
}

// IsEmpty reports whether the ring buffer contains zero items.
func (s *SPSC[T]) IsEmpty() bool {
	return s.Len() == 0
}

// Drain pops and yields all buffered items in FIFO order until the buffer is empty.
// Thread Safety Invariant: Must be invoked from EXACTLY ONE Consumer goroutine.
func (s *SPSC[T]) Drain() iter.Seq[*T] {
	return func(yield func(*T) bool) {
		for {
			item := s.Pop()
			if item == nil {
				return
			}
			if !yield(item) {
				return
			}
		}
	}
}
