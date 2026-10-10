// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generic

import (
	"sync"
)

// SlicePool provides a reusable, type-safe slice pool for reducing heap allocations.
type SlicePool[T any] struct {
	pool       sync.Pool
	holderPool sync.Pool
	cap        int
}

// NewSlicePool creates a new [SlicePool] with a default initial capacity for allocated slices.
func NewSlicePool[T any](capacity int) *SlicePool[T] {
	if capacity <= 0 {
		capacity = 32
	}
	p := &SlicePool[T]{
		cap: capacity,
	}
	p.holderPool = sync.Pool{
		New: func() any {
			return new([]T)
		},
	}
	p.pool = sync.Pool{
		New: func() any {
			h := p.holderPool.Get().(*[]T)
			*h = make([]T, 0, capacity)
			return h
		},
	}
	return p
}

// Get acquires an empty slice with at least the default capacity.
func (p *SlicePool[T]) Get() []T {
	h := p.pool.Get().(*[]T)
	s := (*h)[:0]
	*h = nil
	p.holderPool.Put(h)
	return s
}

// Put returns a slice to the pool. If the slice capacity exceeds 4x the default capacity
// or is less than the pool capacity, it is discarded to prevent unbounded memory growth
// or pool corruption. All elements in the slice are cleared to prevent memory retention.
func (p *SlicePool[T]) Put(s []T) {
	if cap(s) < p.cap || cap(s) > p.cap*4 {
		return
	}
	clear(s[:cap(s)])
	h := p.holderPool.Get().(*[]T)
	*h = s[:0]
	p.pool.Put(h)
}
