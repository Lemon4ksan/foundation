// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package test

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/generic"
	"github.com/lemon4ksan/foundation/silicon/offheap"
	"github.com/lemon4ksan/foundation/silicon/sysnet"
	"github.com/lemon4ksan/foundation/structures/deque"
	"github.com/lemon4ksan/foundation/structures/ringbuffer"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

func TestChallenge_Buffer_GCReclamationWithoutRelease(t *testing.T) {
	// Rapidly allocate 5,000 off-heap buffers without explicit Release.
	// Verify that runtime.AddCleanup properly reclaims OS kernel memory via GC
	// without exhausting Windows commit quota or leaking address space.
	const batches = 10
	const perBatch = 500
	const bufSize = 64 * 1024 // 64KB per buffer

	for b := range batches {
		func() {
			bufs := make([]*offheap.Buffer, perBatch)
			for i := range perBatch {
				buf, err := offheap.NewBuffer(bufSize)
				require.NoError(t, err, fmt.Sprintf("batch %d buffer %d allocation failed", b, i))
				require.NotNil(t, buf)

				// Write a distinct pattern to verify page validity
				raw := buf.RawBytes(16)
				raw[0] = byte(i & 0xFF)
				bufs[i] = buf
			}
			// Let bufs go out of scope here
			_ = bufs
		}()

		// Force GC to trigger runtime.AddCleanup handlers
		runtime.GC()
		runtime.Gosched()
	}
}

func TestChallenge_Buffer_ExplicitReleaseThenGC(t *testing.T) {
	const count = 1000
	bufs := make([]*offheap.Buffer, count)

	for i := range count {
		buf, err := offheap.NewBuffer(32 * 1024)
		require.NoError(t, err)
		bufs[i] = buf
	}

	// Explicitly release all buffers
	for _, buf := range bufs {
		buf.Release()
		// Double release must be safe and idempotent
		buf.Release()
		assert.Nil(t, buf.Bytes())
	}

	// Trigger GC multiple cycles to ensure cleanup.Stop() prevented double-free
	for range 3 {
		runtime.GC()
		runtime.Gosched()
	}
}

func TestChallenge_Buffer_ConcurrentAccessAndRelease(t *testing.T) {
	const numWorkers = 10
	const iters = 100

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for w := range numWorkers {
		go func(workerID int) {
			defer wg.Done()
			for i := range iters {
				buf, err := offheap.NewBuffer(16 * 1024)
				if err != nil {
					t.Errorf("worker %d iter %d alloc failed: %v", workerID, i, err)
					return
				}

				raw := buf.RawBytes(8)
				raw[0] = byte(workerID)

				if (workerID+i)%2 == 0 {
					buf.Release()
				}
				// Otherwise abandoned to runtime.AddCleanup
			}
		}(w)
	}

	// Concurrent GC stress
	stopGC := make(chan struct{})
	go func() {
		for {
			select {
			case <-stopGC:
				return
			case <-time.After(5 * time.Millisecond):
				runtime.GC()
			}
		}
	}()

	wg.Wait()
	close(stopGC)
	runtime.GC()
}

func TestChallenge_Arena_GCReclamationWithoutRelease(t *testing.T) {
	// Each default Arena is 2MB of OS kernel virtual memory.
	// Allocating 500 arenas = 1 GB of virtual memory commitment.
	// If GC cleanups fail, this would exhaust commit limit and fail.
	const count = 500
	const arenaSize = 2 * 1024 * 1024

	for batch := range 5 {
		func() {
			arenas := make([]*offheap.Arena, count/5)
			for i := range count / 5 {
				a, err := offheap.NewArena(arenaSize)
				require.NoError(t, err, fmt.Sprintf("batch %d arena %d alloc failed", batch, i))
				require.NotNil(t, a)

				ptr := a.Alloc(128)
				require.NotNil(t, ptr)
				arenas[i] = a
			}
			_ = arenas
		}()

		runtime.GC()
		runtime.Gosched()
	}
}

func TestChallenge_Arena_ExplicitReleaseThenGC(t *testing.T) {
	const count = 200
	arenas := make([]*offheap.Arena, count)

	for i := range count {
		a, err := offheap.NewArena(1024 * 1024)
		require.NoError(t, err)
		ptr := a.Alloc(64)
		require.NotNil(t, ptr)
		arenas[i] = a
	}

	for _, a := range arenas {
		a.Release()
		// Double release idempotence
		a.Release()
	}

	for range 3 {
		runtime.GC()
		runtime.Gosched()
	}
}

func TestChallenge_Arena_ConcurrentAllocations(t *testing.T) {
	const numWorkers = 8
	const iters = 50

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for w := range numWorkers {
		go func(workerID int) {
			defer wg.Done()
			for i := range iters {
				a, err := offheap.NewArena(256 * 1024)
				if err != nil {
					t.Errorf("worker %d arena alloc failed: %v", workerID, err)
					return
				}

				ptr := a.Alloc(256)
				if ptr == nil {
					t.Errorf("worker %d bump alloc failed", workerID)
					return
				}

				if (workerID+i)%2 == 0 {
					a.Release()
				}
			}
		}(w)
	}

	wg.Wait()
	runtime.GC()
}

func TestChallenge_Slab_GCReclamationWithoutRelease(t *testing.T) {
	type slotData struct {
		id    int64
		flags uint64
		data  [32]byte
	}

	const slabsCount = 100
	for range 5 {
		func() {
			slabs := make([]*offheap.SlabAllocator[slotData], slabsCount/5)
			for i := range slabsCount / 5 {
				s, err := offheap.NewSlabAllocator[slotData](256)
				require.NoError(t, err)
				require.NotNil(t, s)

				p1 := s.Alloc()
				require.NotNil(t, p1)
				p1.id = int64(i)

				slabs[i] = s
			}
			_ = slabs
		}()

		runtime.GC()
		runtime.Gosched()
	}
}

func TestChallenge_Slab_ExplicitReleaseThenGC(t *testing.T) {
	type node struct {
		val int32
	}

	const count = 100
	slabs := make([]*offheap.SlabAllocator[node], count)

	for i := range count {
		s, err := offheap.NewSlabAllocator[node](64)
		require.NoError(t, err)
		item := s.Alloc()
		require.NotNil(t, item)
		item.val = int32(i)
		s.Free(item)
		slabs[i] = s
	}

	for _, s := range slabs {
		s.Release()
		// Double release idempotence
		s.Release()
	}

	for range 3 {
		runtime.GC()
		runtime.Gosched()
	}
}

func TestChallenge_Cache_JanitorGoroutineGCReclamation(t *testing.T) {
	// Baseline goroutine measurement
	time.Sleep(30 * time.Millisecond)
	runtime.GC()
	time.Sleep(30 * time.Millisecond)
	baseGoroutines := runtime.NumGoroutine()

	const cacheCount = 100
	func() {
		caches := make([]*generic.Cache[int, int], cacheCount)
		for i := range cacheCount {
			c := generic.NewCacheWithJanitor[int, int](15 * time.Millisecond)
			c.Set(i, i*10, time.Hour)
			caches[i] = c
		}

		// Verify goroutines were spawned
		createdGoroutines := runtime.NumGoroutine()
		assert.GreaterOrEqual(t, createdGoroutines, baseGoroutines+cacheCount-10)

		// Abandon caches without Close()
		_ = caches
	}()

	// Force GC cycles and verify all janitors terminate via runtime.AddCleanup
	var finalGoroutines int
	success := false
	for range 10 {
		runtime.GC()
		time.Sleep(50 * time.Millisecond)
		finalGoroutines = runtime.NumGoroutine()
		if finalGoroutines <= baseGoroutines+5 {
			success = true
			break
		}
	}

	assert.True(t, success, fmt.Sprintf("expected goroutines to return to baseline %d, got %d", baseGoroutines, finalGoroutines))
}

func TestChallenge_Cache_ExplicitCloseThenGC(t *testing.T) {
	time.Sleep(20 * time.Millisecond)
	runtime.GC()
	baseGoroutines := runtime.NumGoroutine()

	const cacheCount = 100
	caches := make([]*generic.Cache[int, int], cacheCount)
	for i := range cacheCount {
		caches[i] = generic.NewCacheWithJanitor[int, int](15 * time.Millisecond)
		caches[i].Set(i, i, time.Hour)
	}

	// Explicit Close
	for _, c := range caches {
		c.Close()
		// Double Close idempotence
		c.Close()
	}

	time.Sleep(50 * time.Millisecond)
	assert.LessOrEqual(t, runtime.NumGoroutine(), baseGoroutines+5)

	// GC after Close
	for range 3 {
		runtime.GC()
		runtime.Gosched()
	}
}

func TestChallenge_Cache_ConcurrentOpsAndCloseRace(t *testing.T) {
	const numWorkers = 8
	const iters = 200

	c := generic.NewCacheWithJanitor[int, int](10 * time.Millisecond)

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for w := range numWorkers {
		go func(workerID int) {
			defer wg.Done()
			for i := range iters {
				if workerID == 0 && i == iters/2 {
					c.Close()
				}
				key := (workerID * iters) + i
				c.Set(key, i, 50*time.Millisecond)
				_, _ = c.Get(key)
				if i%10 == 0 {
					c.PurgeExpired()
				}
			}
		}(w)
	}

	wg.Wait()
	c.Close()
	runtime.GC()
}

func TestChallenge_RIO_BufferRegistration_Lifecycle(t *testing.T) {
	if !sysnet.IsRIOSupported() {
		t.Skip("Windows RIO / VirtualLock not supported in current environment")
	}

	const count = 50
	dataSlices := make([][]byte, count)
	for i := range count {
		dataSlices[i] = make([]byte, 4096)
	}

	// 1. Explicit Deregister followed by GC
	for i := range count {
		reg, err := sysnet.RegisterBuffer(dataSlices[i])
		if err == nil && reg != nil {
			reg.Deregister()
			// Double Deregister idempotence
			reg.Deregister()
		}
	}
	runtime.GC()

	// 2. Abandoned registration reclaimed by runtime.AddCleanup
	func() {
		regs := make([]*sysnet.BufferRegistration, count)
		for i := range count {
			reg, err := sysnet.RegisterBuffer(dataSlices[i])
			if err == nil && reg != nil {
				regs[i] = reg
			}
		}
		_ = regs
	}()

	for range 3 {
		runtime.GC()
		runtime.Gosched()
	}
}

func TestChallenge_Tracking_AddCleanup(t *testing.T) {
	// Verify that runtime.AddCleanup behaves properly on deque and ringbuffer
	var finalized atomic.Int32

	d := deque.New[*int]()
	for i := range 100 {
		val := i
		runtime.AddCleanup(&val, func(counter *atomic.Int32) {
			counter.Add(1)
		}, &finalized)
		d.PushBack(&val)
	}

	d.Clear()
	for range 5 {
		runtime.GC()
		time.Sleep(10 * time.Millisecond)
		if finalized.Load() > 0 {
			break
		}
	}

	assert.Greater(t, finalized.Load(), int32(0))

	// RingBuffer tracking test
	var rbFinalized atomic.Int32
	rb := new(ringbuffer.Buffer[*int])
	rb.Init(64)
	for i := range 50 {
		val := i
		runtime.AddCleanup(&val, func(counter *atomic.Int32) {
			counter.Add(1)
		}, &rbFinalized)
		rb.PushBack(&val)
	}

	for !rb.Empty() {
		_ = rb.PopFront()
	}

	for range 5 {
		runtime.GC()
		time.Sleep(10 * time.Millisecond)
		if rbFinalized.Load() > 0 {
			break
		}
	}

	assert.Greater(t, rbFinalized.Load(), int32(0))
}

func TestChallenge_ExtremePressure_MixedAllocatorsAndGC(t *testing.T) {
	time.Sleep(50 * time.Millisecond)
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	baseGoroutines := runtime.NumGoroutine()

	const numWorkers = 8
	const iters = 50

	var stopGC atomic.Bool
	var gcCount atomic.Int64

	// GC hammer goroutine
	go func() {
		for !stopGC.Load() {
			runtime.GC()
			gcCount.Add(1)
			time.Sleep(3 * time.Millisecond)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	type testItem struct {
		id   int64
		data [64]byte
	}

	for w := range numWorkers {
		go func(workerID int) {
			defer wg.Done()

			for i := range iters {
				// 1. Offheap Buffer
				buf, errBuf := offheap.NewBuffer(32 * 1024)
				if errBuf == nil && buf != nil {
					raw := buf.RawBytes(16)
					raw[0] = byte(workerID ^ i)
					if (workerID+i)%2 == 0 {
						buf.Release()
					}
				}

				// 2. Offheap Arena
				arena, errArena := offheap.NewArena(128 * 1024)
				if errArena == nil && arena != nil {
					ptr := arena.Alloc(128)
					_ = ptr
					if (workerID+i)%2 == 0 {
						arena.Release()
					}
				}

				// 3. Offheap Slab
				slab, errSlab := offheap.NewSlabAllocator[testItem](64)
				if errSlab == nil && slab != nil {
					item := slab.Alloc()
					if item != nil {
						item.id = int64(workerID*1000 + i)
						slab.Free(item)
					}
					if (workerID+i)%2 == 0 {
						slab.Release()
					}
				}

				// 4. Cache with Janitor
				c := generic.NewCacheWithJanitor[int, string](5 * time.Millisecond)
				c.Set(i, "val", 20*time.Millisecond)
				_, _ = c.Get(i)
				if (workerID+i)%2 == 0 {
					c.Close()
				}
			}
		}(w)
	}

	wg.Wait()
	stopGC.Store(true)

	// Wait for abandoned caches to have their janitors stopped via AddCleanup
	var settledGoroutines int
	clean := false
	for range 20 {
		runtime.GC()
		time.Sleep(50 * time.Millisecond)
		settledGoroutines = runtime.NumGoroutine()
		if settledGoroutines <= baseGoroutines+5 {
			clean = true
			break
		}
	}

	assert.True(t, clean, fmt.Sprintf("expected goroutines to settle near baseline %d, got %d (gc iterations: %d)", baseGoroutines, settledGoroutines, gcCount.Load()))
}
