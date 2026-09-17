// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pool

import (
	"runtime"
	"sync/atomic"

	"golang.org/x/sys/cpu"
)

const shardCapacity = 32

type bufferShard[T any] struct {
	_     cpu.CacheLinePad
	items [shardCapacity]T
	head  atomic.Uint32
	mu    atomic.Uint32
	_     cpu.CacheLinePad
}

// PerPStorage provides a sharded per-CPU core memory pool with 0 cross-core CAS contention or Work Stealing locks.
type PerPStorage[T any] struct {
	shards  []bufferShard[T]
	mask    uint64
	cursor  atomic.Uint64
	factory func() T
}

// nextPowerOfTwo calculates the smallest power of 2 greater than or equal to n.
func nextPowerOfTwo(n int) int {
	if n <= 1 {
		return 1
	}

	p := 1
	for p < n {
		p <<= 1
	}

	return p
}

// NewPerPStorage constructs a PerPStorage sharded according to runtime.GOMAXPROCS(0) rounded to a power of 2.
func NewPerPStorage[T any](factory func() T) *PerPStorage[T] {
	n := runtime.GOMAXPROCS(0)
	if n <= 0 {
		n = 1
	}

	powerOfTwoN := nextPowerOfTwo(n)
	shards := make([]bufferShard[T], powerOfTwoN)

	return &PerPStorage[T]{
		shards:  shards,
		mask:    uint64(powerOfTwoN - 1),
		factory: factory,
	}
}

// Get retrieves an item from a local CPU shard, falling back to scanning other shards before allocating.
func (p *PerPStorage[T]) Get() T {
	numShards := uint64(len(p.shards))
	startIdx := p.cursor.Add(1) & p.mask

	for i := range numShards {
		idx := (startIdx + i) & p.mask
		shard := &p.shards[idx]

		// Lockless Shared-state pre-check: skip empty shards without CAS bus lock
		if shard.head.Load() == 0 {
			continue
		}

		if shard.mu.CompareAndSwap(0, 1) {
			head := shard.head.Load()
			if head > 0 {
				item := shard.items[head-1]

				var zero T

				shard.items[head-1] = zero
				shard.head.Store(head - 1)
				shard.mu.Store(0)

				return item
			}

			shard.mu.Store(0)
		}
	}

	if p.factory != nil {
		return p.factory()
	}

	var zero T

	return zero
}

// Put recycles an item back into a local CPU shard, placing it in the first non-full shard.
func (p *PerPStorage[T]) Put(item T) {
	numShards := uint64(len(p.shards))
	startIdx := p.cursor.Add(1) & p.mask

	for i := range numShards {
		idx := (startIdx + i) & p.mask
		shard := &p.shards[idx]

		// Lockless Shared-state pre-check: skip full shards without CAS bus lock
		if shard.head.Load() >= shardCapacity {
			continue
		}

		if shard.mu.CompareAndSwap(0, 1) {
			head := shard.head.Load()
			if head < shardCapacity {
				shard.items[head] = item
				shard.head.Store(head + 1)
				shard.mu.Store(0)

				return
			}

			shard.mu.Store(0)
		}
	}
}
