// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pool

import (
	"sync"
	"testing"

	"github.com/lemon4ksan/foundation/testing/assert"
)

func TestPerCorePool_Internal_NextPowerOfTwo(t *testing.T) {
	assert.Equal(t, 1, nextPowerOfTwo(-5))
	assert.Equal(t, 1, nextPowerOfTwo(0))
	assert.Equal(t, 1, nextPowerOfTwo(1))
	assert.Equal(t, 2, nextPowerOfTwo(2))
	assert.Equal(t, 4, nextPowerOfTwo(3))
	assert.Equal(t, 4, nextPowerOfTwo(4))
	assert.Equal(t, 8, nextPowerOfTwo(5))
}

func TestPerCorePool_Internal_NonPositiveShards(t *testing.T) {
	p := newPerCorePoolWithShards[int](0, nil)
	assert.Equal(t, 1, len(p.shards))
	assert.Equal(t, uint64(0), p.mask)
}

func TestPerCorePool_Internal_GetContentionRetry(t *testing.T) {
	p := newPerCorePoolWithShards[int](1, func() int { return 100 })
	p.shards[0].items[0] = 42
	p.shards[0].head.Store(1)

	// Lock the shard to force CAS contention across all scan attempts
	p.shards[0].mu.Store(1)

	// Contended Get falls back to factory when all shards are locked
	val := p.Get()
	assert.Equal(t, 100, val)

	// Once unlocked, normal Get retrieves the cached item
	p.shards[0].mu.Store(0)
	val2 := p.Get()
	assert.Equal(t, 42, val2)
}

func TestPerCorePool_Internal_PutContentionRetry(t *testing.T) {
	p := newPerCorePoolWithShards[int](1, nil)

	// Lock the shard to force CAS contention across all scan attempts
	p.shards[0].mu.Store(1)

	// Contended Put retries and exits gracefully without putting
	p.Put(99)
	assert.Equal(t, uint32(0), p.shards[0].head.Load())

	// Once unlocked, normal Put succeeds
	p.shards[0].mu.Store(0)
	p.Put(99)
	assert.Equal(t, uint32(1), p.shards[0].head.Load())
	assert.Equal(t, 99, p.shards[0].items[0])
}

func TestPerCorePool_Internal_FullShardPrecheckAndBreak(t *testing.T) {
	p := newPerCorePoolWithShards[int](1, nil)
	for i := range shardCapacity {
		p.Put(i)
	}

	// Shard is full (head == 32). Calling Put must pre-check full, break out, and not panic.
	p.Put(999)
	assert.Equal(t, uint32(shardCapacity), p.shards[0].head.Load())
}

func TestPerCorePool_Internal_RaceHeadChecks(t *testing.T) {
	// Race Get on single-item shard to trigger post-CAS head == 0 check
	for range 200 {
		p := newPerCorePoolWithShards[int](1, nil)
		p.shards[0].items[0] = 7
		p.shards[0].head.Store(1)

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			_ = p.Get()
		}()

		go func() {
			defer wg.Done()
			_ = p.Get()
		}()

		wg.Wait()
	}

	// Race Put on almost-full shard to trigger post-CAS head >= shardCapacity check
	for range 200 {
		p := newPerCorePoolWithShards[int](1, nil)
		for i := range shardCapacity - 1 {
			p.shards[0].items[i] = i
		}
		p.shards[0].head.Store(uint32(shardCapacity - 1))

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			p.Put(100)
		}()

		go func() {
			defer wg.Done()
			p.Put(200)
		}()

		wg.Wait()
	}
}
