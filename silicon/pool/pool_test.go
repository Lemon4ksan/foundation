// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pool_test

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/lemon4ksan/foundation/silicon/pool"
)

func TestTimerPool(t *testing.T) {
	t.Parallel()

	// 1. Expired timer release
	timer1 := pool.AcquireTimer(10 * time.Millisecond)
	assert.NotNil(t, timer1)
	<-timer1.C
	pool.ReleaseTimer(timer1)

	// 2. Unexpired timer release
	timer2 := pool.AcquireTimer(1 * time.Hour)
	assert.NotNil(t, timer2)
	pool.ReleaseTimer(timer2)

	// 3. Nil timer release safety
	pool.ReleaseTimer(nil)

	// 4. Reacquire recycled timer
	timer3 := pool.AcquireTimer(10 * time.Millisecond)
	assert.NotNil(t, timer3)
	<-timer3.C
	pool.ReleaseTimer(timer3)
}

func TestRequestArena(t *testing.T) {
	t.Parallel()

	arena := pool.GetRequestArena()
	assert.NotNil(t, arena)

	// Zero or negative alloc
	assert.Nil(t, arena.Alloc(0))
	assert.Nil(t, arena.Alloc(-5))

	// Fast bump allocation within 4096 bytes
	b1 := arena.Alloc(1024)
	assert.Len(t, b1, 1024)
	b2 := arena.Alloc(2048)
	assert.Len(t, b2, 2048)

	// Overflow allocation falls back to heap slice
	bOverflow := arena.Alloc(4096)
	assert.Len(t, bOverflow, 4096)

	// Reset
	arena.Reset()
	bReset := arena.Alloc(4096)
	assert.Len(t, bReset, 4096)

	// Release to pool
	pool.ReleaseRequestArena(arena)
	pool.ReleaseRequestArena(nil) // nil safety
}

func TestPerPStorage(t *testing.T) {
	t.Parallel()

	// With factory
	storage := pool.NewPerPStorage(func() []byte {
		return make([]byte, 128)
	})

	item := storage.Get()
	assert.Len(t, item, 128)
	storage.Put(item)

	item2 := storage.Get()
	assert.Len(t, item2, 128)

	// Without factory (returns zero value)
	storageNoFactory := pool.NewPerPStorage[int](nil)
	assert.Equal(t, 0, storageNoFactory.Get())

	// Fill shard capacity
	for i := 1; i <= 100; i++ {
		storageNoFactory.Put(i)
	}

	for i := 0; i < 50; i++ {
		_ = storageNoFactory.Get()
	}

	// Concurrent Put/Get
	var wg sync.WaitGroup
	const goroutines = 32
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				b := storage.Get()
				storage.Put(b)
			}
		}()
	}

	wg.Wait()
}
