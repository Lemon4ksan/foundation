// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pool

import (
	"runtime"
	"sync/atomic"
	_ "unsafe"

	"golang.org/x/sys/cpu"
)

// fastrand returns a lock-free pseudo-random 32-bit integer provided by the Go runtime.
//
//go:linkname fastrand runtime.fastrand
func fastrand() uint32

const shardCapacity = 32

type bufferShard[T any] struct {
	_     cpu.CacheLinePad
	items [shardCapacity]T
	head  atomic.Uint32
	mu    atomic.Uint32
	_     cpu.CacheLinePad
}

// PerCorePool provides a sharded per-CPU core memory pool with 0 cross-core CAS contention or Work Stealing locks.
type PerCorePool[T any] struct {
	shards  []bufferShard[T]
	mask    uint64
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

// NewPerCorePool constructs a PerCorePool sharded according to runtime.GOMAXPROCS(0) rounded to a power of 2.
func NewPerCorePool[T any](factory func() T) *PerCorePool[T] {
	return newPerCorePoolWithShards(runtime.GOMAXPROCS(0), factory)
}

func newPerCorePoolWithShards[T any](n int, factory func() T) *PerCorePool[T] {
	if n <= 0 {
		n = 1
	}

	powerOfTwoN := nextPowerOfTwo(n)
	shards := make([]bufferShard[T], powerOfTwoN)

	return &PerCorePool[T]{
		shards:  shards,
		mask:    uint64(powerOfTwoN - 1),
		factory: factory,
	}
}

// Get retrieves an item from a local CPU shard, falling back to scanning other shards before allocating.
func (p *PerCorePool[T]) Get() T {
	numShards := uint64(len(p.shards))
	startIdx := uint64(fastrand()) & p.mask

	var contended bool
	for range 2 {
		for i := range numShards {
			idx := (startIdx + i) & p.mask
			shard := &p.shards[idx]

			// Lockless Shared-state pre-check: skip empty shards without CAS bus lock
			if shard.head.Load() == 0 {
				continue
			}

			if !shard.mu.CompareAndSwap(0, 1) {
				contended = true
				continue
			}

			head := shard.head.Load()
			if head == 0 {
				shard.mu.Store(0)
				continue
			}

			item := shard.items[head-1]
			var zero T
			shard.items[head-1] = zero
			shard.head.Store(head - 1)
			shard.mu.Store(0)

			return item
		}

		if !contended {
			break
		}
		contended = false
		runtime.Gosched()
	}

	if p.factory != nil {
		return p.factory()
	}

	var zero T

	return zero
}

// Put recycles an item back into a local CPU shard, placing it in the first non-full shard.
func (p *PerCorePool[T]) Put(item T) {
	numShards := uint64(len(p.shards))
	startIdx := uint64(fastrand()) & p.mask

	var contended bool
	for range 2 {
		for i := range numShards {
			idx := (startIdx + i) & p.mask
			shard := &p.shards[idx]

			// Lockless Shared-state pre-check: skip full shards without CAS bus lock
			if shard.head.Load() >= shardCapacity {
				continue
			}

			if !shard.mu.CompareAndSwap(0, 1) {
				contended = true
				continue
			}

			head := shard.head.Load()
			if head >= shardCapacity {
				shard.mu.Store(0)
				continue
			}

			shard.items[head] = item
			shard.head.Store(head + 1)
			shard.mu.Store(0)

			return
		}

		if !contended {
			break
		}
		contended = false
		runtime.Gosched()
	}
}
