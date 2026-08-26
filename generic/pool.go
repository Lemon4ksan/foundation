// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generic

import (
	"sync"
)

// SlicePool provides a reusable, type-safe slice pool for reducing heap allocations.
type SlicePool[T any] struct {
	pool sync.Pool
	cap  int
}

// NewSlicePool creates a new [SlicePool] with a default initial capacity for allocated slices.
func NewSlicePool[T any](capacity int) *SlicePool[T] {
	if capacity <= 0 {
		capacity = 32
	}
	return &SlicePool[T]{
		cap: capacity,
		pool: sync.Pool{
			New: func() any {
				s := make([]T, 0, capacity)
				return &s
			},
		},
	}
}

// Get acquires an empty slice with at least the default capacity.
func (p *SlicePool[T]) Get() []T {
	ptr := p.pool.Get().(*[]T)
	return (*ptr)[:0]
}

// Put returns a slice to the pool. If the slice capacity exceeds 4x the default capacity,
// it is discarded to prevent unbounded memory growth.
func (p *SlicePool[T]) Put(s []T) {
	if cap(s) > p.cap*4 {
		return
	}
	s = s[:0]
	p.pool.Put(&s)
}
