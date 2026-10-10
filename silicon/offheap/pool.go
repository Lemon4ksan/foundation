// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package offheap

import (
	"sync"
	"sync/atomic"
)

// ArenaPool is a thread-safe pool of [Arena] instances for concurrent goroutine access.
//
// Each goroutine calls [ArenaPool.Acquire] to check out an arena, uses it for
// scoped allocations, and then calls [ArenaPool.Release] to reset and return it.
// Released arenas are recycled by sync.Pool and may be evicted under GC pressure,
// in which case a fresh arena is allocated automatically on the next Acquire.
//
// ArenaPool is safe for concurrent use by multiple goroutines.
type ArenaPool struct {
	pool         sync.Pool
	pageSize     int
	activeArenas atomic.Int64
	totalCreated atomic.Int64
}

// NewArenaPool creates an [ArenaPool] that provisions arenas of pageSize bytes each.
// If pageSize <= 0, a 2MB default is used.
func NewArenaPool(pageSize int) *ArenaPool {
	if pageSize <= 0 {
		pageSize = 2 * 1024 * 1024
	}

	p := &ArenaPool{pageSize: pageSize}
	p.pool = sync.Pool{
		New: func() any {
			a, err := NewArena(p.pageSize)
			if err != nil {
				return nil
			}
			p.totalCreated.Add(1)

			return a
		},
	}

	return p
}

// Acquire checks out a clean, reset [Arena] from the pool.
// Returns nil only if the OS kernel refused to allocate memory.
func (p *ArenaPool) Acquire() *Arena {
	v := p.pool.Get()
	if v == nil {
		return nil
	}

	a := v.(*Arena)

	// Defensive: if the finalizer ran while in pool (e.g. explicit Release misuse),
	// allocate a fresh arena rather than returning a dead one.
	if a.page == nil {
		fresh, err := NewArena(p.pageSize)
		if err != nil {
			return nil
		}
		p.totalCreated.Add(1)
		p.activeArenas.Add(1)

		return fresh
	}

	a.Reset()
	p.activeArenas.Add(1)

	return a
}

// Release resets the arena to offset 0 and returns it to the pool for reuse.
// The caller MUST NOT use the arena after calling Release.
func (p *ArenaPool) Release(a *Arena) {
	if a == nil || a.page == nil {
		return
	}

	a.Reset()
	p.activeArenas.Add(-1)
	p.pool.Put(a)
}

// PageSize returns the configured per-arena page size in bytes.
func (p *ArenaPool) PageSize() int {
	return p.pageSize
}

// ActiveCount returns the number of arenas currently checked out of the pool.
func (p *ArenaPool) ActiveCount() int {
	c := p.activeArenas.Load()
	if c < 0 {
		return 0
	}
	return int(c)
}

// TotalCreated returns the total number of physical arenas allocated by the pool.
func (p *ArenaPool) TotalCreated() int {
	return int(p.totalCreated.Load())
}

// AllocatedBytes returns the total off-heap bytes currently checked out by active callers.
func (p *ArenaPool) AllocatedBytes() int64 {
	c := p.activeArenas.Load()
	if c < 0 {
		return 0
	}
	return c * int64(p.pageSize)
}
