// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pool_test

import (
	"fmt"
	"math"
	"math/rand/v2"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/lemon4ksan/foundation/silicon/pool"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
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

func TestTimerPool_ZeroAlloc(t *testing.T) {
	tm := pool.AcquireTimer(time.Hour)
	pool.ReleaseTimer(tm)

	allocs := testing.AllocsPerRun(100, func() {
		timer := pool.AcquireTimer(time.Hour)
		pool.ReleaseTimer(timer)
	})
	assert.Equal(t, float64(0), allocs)
}

func TestPerCorePool_ZeroAlloc(t *testing.T) {
	p := pool.NewPerCorePool(func() *[64]byte {
		return &[64]byte{}
	})
	// Prewarm
	item := p.Get()
	p.Put(item)

	allocs := testing.AllocsPerRun(100, func() {
		obj := p.Get()
		p.Put(obj)
	})
	assert.Equal(t, float64(0), allocs)
}

func TestRequestArena(t *testing.T) {
	t.Parallel()

	arena := pool.AcquireRequestArena()
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

func TestPerCorePool(t *testing.T) {
	t.Parallel()

	// With factory
	storage := pool.NewPerCorePool(func() []byte {
		return make([]byte, 128)
	})

	item := storage.Get()
	assert.Len(t, item, 128)
	storage.Put(item)

	item2 := storage.Get()
	assert.Len(t, item2, 128)

	// Without factory (returns zero value)
	storageNoFactory := pool.NewPerCorePool[int](nil)
	assert.Equal(t, 0, storageNoFactory.Get())

	// Fill shard capacity
	for i := 1; i <= 100; i++ {
		storageNoFactory.Put(i)
	}

	for range 50 {
		_ = storageNoFactory.Get()
	}

	// Concurrent Put/Get
	var wg sync.WaitGroup
	const goroutines = 32
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			for range 100 {
				b := storage.Get()
				storage.Put(b)
			}
		}()
	}

	wg.Wait()
}

func TestRepro_RequestArena_IntegerOverflowGuard(t *testing.T) {
	t.Parallel()

	arena := pool.AcquireRequestArena()
	defer pool.ReleaseRequestArena(arena)

	// Allocating large value > 4096 must not panic and should not corrupt arena offset
	b := arena.Alloc(5000)
	assert.Equal(t, 5000, len(b))

	// Large n near MaxInt must not wrap around negatively inside arena and access a.buf
	defer func() {
		r := recover()
		t.Logf("caught panic: %v", r)
		assert.NotNil(t, r)
		assert.NotContains(
			t,
			r.(error).Error(),
			"4096",
			"integer overflow caused out-of-bounds slice indexing on a.buf!",
		)
	}()

	_ = arena.Alloc(math.MaxInt - 1)
}

type trackedItem struct {
	id    uint64
	inUse atomic.Bool
	val   uint64
}

// TestPerCorePool_AdversarialConcurrency verifies that under high multi-goroutine
// concurrency across different GOMAXPROCS configurations, PerCorePool never hands out
// the same item to multiple goroutines simultaneously and produces 0 data races.
func TestPerCorePool_AdversarialConcurrency(t *testing.T) {
	origProcs := runtime.GOMAXPROCS(0)
	defer runtime.GOMAXPROCS(origProcs)

	procsConfigs := []int{1, 2, 4, 8}

	for _, procs := range procsConfigs {
		t.Run(fmt.Sprintf("GOMAXPROCS=%d", procs), func(t *testing.T) {
			runtime.GOMAXPROCS(procs)

			var idCounter atomic.Uint64
			p := pool.NewPerCorePool(func() *trackedItem {
				return &trackedItem{
					id: idCounter.Add(1),
				}
			})

			const goroutines = 64
			const iterations = 2000

			var wg sync.WaitGroup
			var doubleCheckoutCount atomic.Uint64

			wg.Add(goroutines)
			for g := range goroutines {
				go func(workerID int) {
					defer wg.Done()

					for i := range iterations {
						item := p.Get()
						if item == nil {
							t.Errorf("worker %d got nil item on iter %d", workerID, i)
							return
						}

						// Double checkout detection: if already in use, concurrency failure!
						if !item.inUse.CompareAndSwap(false, true) {
							doubleCheckoutCount.Add(1)
							t.Errorf("double checkout detected for item ID %d by worker %d", item.id, workerID)
							return
						}

						// Touch item memory to trigger race detector if multiple goroutines access
						item.val = uint64(workerID*iterations + i)
						if item.val != uint64(workerID*iterations+i) {
							t.Errorf("memory corruption in item ID %d", item.id)
						}

						// Release item
						item.inUse.Store(false)
						p.Put(item)
					}
				}(g)
			}

			wg.Wait()
			assert.Equal(t, uint64(0), doubleCheckoutCount.Load())
		})
	}
}

// TestPerCorePool_ShardOverflowAndDrain verifies that putting more items than the total
// pool capacity safely drops extra items without memory corruption or blocking,
// and draining subsequent items succeeds.
func TestPerCorePool_ShardOverflowAndDrain(t *testing.T) {
	p := pool.NewPerCorePool[int](nil)

	// Shard capacity is 32 per shard. Even with 16 shards, capacity is 512.
	// We put 2,000 items. Extra items must be dropped cleanly.
	for i := 1; i <= 2000; i++ {
		p.Put(i)
	}

	// Drain all items
	retrieved := 0
	for {
		item := p.Get()
		if item == 0 {
			break
		}
		retrieved++
		if retrieved > 2000 {
			t.Fatalf("retrieved more items than inserted!")
		}
	}

	// We should have retrieved at least 1 item and at most total capacity.
	assert.True(t, retrieved > 0, "should have retrieved cached items")
	t.Logf("Retrieved %d items from bounded pool", retrieved)

	// After drain, Get() with nil factory must return zero value (0)
	assert.Equal(t, 0, p.Get())
}

// TestPerCorePool_ZeroValueFactory verifies that when factory is nil, Get() returns zero value.
func TestPerCorePool_ZeroValueFactory(t *testing.T) {
	pInt := pool.NewPerCorePool[int](nil)
	assert.Equal(t, 0, pInt.Get())

	pPtr := pool.NewPerCorePool[*trackedItem](nil)
	assert.Nil(t, pPtr.Get())

	type customStruct struct {
		name string
		num  int
	}
	pStruct := pool.NewPerCorePool[customStruct](nil)
	assert.Equal(t, customStruct{}, pStruct.Get())
}

// TestRequestArena_MultiGoroutineRecycleStress verifies that AcquireRequestArena and
// ReleaseRequestArena operate safely under high concurrency across multiple goroutines,
// recycling arenas cleanly with 8-byte alignment and 0 cross-talk.
func TestRequestArena_MultiGoroutineRecycleStress(t *testing.T) {
	const goroutines = 64
	const iterations = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := range goroutines {
		go func(workerID int) {
			defer wg.Done()

			pattern := byte(workerID % 255)

			for range iterations {
				arena := pool.AcquireRequestArena()
				require.NotNil(t, arena)

				// Allocate multiple chunks of varying sizes
				sizes := []int{8, 16, 24, 64, 128, 256, 512, 1024}
				var allocatedSlices [][]byte

				for _, sz := range sizes {
					b := arena.Alloc(sz)
					if len(b) != sz {
						t.Errorf("expected len %d, got %d", sz, len(b))
						return
					}

					// Verify 8-byte alignment
					ptrVal := uintptr(unsafe.Pointer(&b[0]))
					if ptrVal%8 != 0 {
						t.Errorf("slice not 8-byte aligned: ptr=%x", ptrVal)
						return
					}

					// Fill slice with worker-specific pattern
					for i := range b {
						b[i] = pattern
					}
					allocatedSlices = append(allocatedSlices, b)
				}

				// Verify pattern in all allocated slices before release
				for _, b := range allocatedSlices {
					for _, v := range b {
						if v != pattern {
							t.Errorf("memory corruption: expected %d, got %d", pattern, v)
							return
						}
					}
				}

				pool.ReleaseRequestArena(arena)
			}
		}(g)
	}

	wg.Wait()
}

// TestRequestArena_AllocationBoundaries tests exact boundary and overflow behavior.
func TestRequestArena_AllocationBoundaries(t *testing.T) {
	arena := pool.AcquireRequestArena()
	defer pool.ReleaseRequestArena(arena)

	// 1. Zero and negative
	assert.Nil(t, arena.Alloc(0))
	assert.Nil(t, arena.Alloc(-100))

	// 2. Exact 4096 bytes allocation
	bExact := arena.Alloc(4096)
	assert.Len(t, bExact, 4096)

	// Next allocation should fall back to heap slice
	bHeap := arena.Alloc(1)
	assert.Len(t, bHeap, 1)

	// 3. Reset and allocate small chunks that add up to 4096
	arena.Reset()
	totalAlloc := 0
	for range 64 {
		chunk := arena.Alloc(64)
		assert.Len(t, chunk, 64)
		totalAlloc += len(chunk)
	}
	assert.Equal(t, 4096, totalAlloc)

	// Subsequent alloc goes to heap
	overflow := arena.Alloc(16)
	assert.Len(t, overflow, 16)

	// 4. Reset and verify large allocation > 4096
	arena.Reset()
	large := arena.Alloc(8192)
	assert.Len(t, large, 8192)

	// 5. Nil release safety
	pool.ReleaseRequestArena(nil)
}

// TestRequestArena_CapacityLeakAudit examines whether slice capacity leaks past requested size.
func TestRequestArena_CapacityLeakAudit(t *testing.T) {
	arena := pool.AcquireRequestArena()
	defer pool.ReleaseRequestArena(arena)

	b := arena.Alloc(10)
	assert.Len(t, b, 10)
	assert.Equal(t, 10, cap(b))

	// Verify appending doesn't touch subsequent arena allocations
	bNext := arena.Alloc(16)
	assert.Len(t, bNext, 16)
	assert.Equal(t, 16, cap(bNext))

	for i := range bNext {
		bNext[i] = 0xAA
	}

	_ = append(b, []byte{0x55, 0x55, 0x55, 0x55}...)

	for i := range bNext {
		assert.Equal(t, byte(0xAA), bNext[i])
	}
}

// TestPerCorePool_RandomizedFuzzing exercises random Get/Put sequences under race detector.
func TestPerCorePool_RandomizedFuzzing(t *testing.T) {
	const goroutines = 32
	const operationsPerGoroutine = 500

	p := pool.NewPerCorePool(func() []byte {
		return make([]byte, 64)
	})

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := range goroutines {
		go func(seed uint64) {
			defer wg.Done()
			rng := rand.New(rand.NewPCG(seed, seed+1))

			var held [][]byte
			for range operationsPerGoroutine {
				op := rng.IntN(2)
				if op == 0 || len(held) == 0 {
					// Get
					item := p.Get()
					require.NotNil(t, item)
					held = append(held, item)
				} else {
					// Put
					lastIdx := len(held) - 1
					p.Put(held[lastIdx])
					held = held[:lastIdx]
				}
			}

			// Clean up remaining held items
			for _, item := range held {
				p.Put(item)
			}
		}(uint64(g + 1))
	}

	wg.Wait()
}

func BenchmarkPerCorePool_GetPut_Sequential(b *testing.B) {
	storage := pool.NewPerCorePool(func() *[64]byte {
		var buf [64]byte
		return &buf
	})

	// Pre-fill shard
	storage.Put(&[64]byte{})

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		obj := storage.Get()
		storage.Put(obj)
	}
}

func BenchmarkPerCorePool_GetPut_Parallel(b *testing.B) {
	storage := pool.NewPerCorePool(func() *[64]byte {
		var buf [64]byte
		return &buf
	})

	// Pre-fill all shards to capacity
	for range 256 {
		storage.Put(&[64]byte{})
	}

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			obj := storage.Get()
			if obj != nil {
				storage.Put(obj)
			}
		}
	})
}

func BenchmarkPerCorePool_HighConcurrency_Contention(b *testing.B) {
	storage := pool.NewPerCorePool(func() *[64]byte {
		var buf [64]byte
		return &buf
	})

	for range 256 {
		storage.Put(&[64]byte{})
	}

	b.SetParallelism(64)
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			obj := storage.Get()
			if obj != nil {
				storage.Put(obj)
			}
		}
	})
}

func BenchmarkPerCorePool_Contention_Burst128(b *testing.B) {
	storage := pool.NewPerCorePool(func() *[64]byte {
		var buf [64]byte
		return &buf
	})

	for range 256 {
		storage.Put(&[64]byte{})
	}

	b.SetParallelism(128)
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			obj := storage.Get()
			if obj != nil {
				storage.Put(obj)
			}
		}
	})
}

func BenchmarkRequestArena_AllocReset(b *testing.B) {
	arena := pool.AcquireRequestArena()
	defer pool.ReleaseRequestArena(arena)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = arena.Alloc(128)
		arena.Reset()
	}
}

func BenchmarkRequestArena_MultipleAllocReset(b *testing.B) {
	arena := pool.AcquireRequestArena()
	defer pool.ReleaseRequestArena(arena)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = arena.Alloc(64)
		_ = arena.Alloc(128)
		_ = arena.Alloc(256)
		_ = arena.Alloc(512)
		arena.Reset()
	}
}

func BenchmarkRequestArena_GetRelease(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		arena := pool.AcquireRequestArena()
		pool.ReleaseRequestArena(arena)
	}
}

func BenchmarkTimerPool_AcquireRelease(b *testing.B) {
	// Pre-warm pool
	t := pool.AcquireTimer(time.Hour)
	pool.ReleaseTimer(t)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		tm := pool.AcquireTimer(time.Hour)
		pool.ReleaseTimer(tm)
	}
}

func BenchmarkTimerPool_AcquireRelease_Parallel(b *testing.B) {
	// Pre-warm pool
	for range 64 {
		t := pool.AcquireTimer(time.Hour)
		pool.ReleaseTimer(t)
	}

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tm := pool.AcquireTimer(time.Hour)
			pool.ReleaseTimer(tm)
		}
	})
}
