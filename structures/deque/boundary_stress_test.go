// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package deque

import (
	"math/rand/v2"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"weak"

	"github.com/lemon4ksan/foundation/testing/require"
)

// -----------------------------------------------------------------------------
// Challenge 1: Buffer Growth and Unwrapping Across All Circular Head Positions
// -----------------------------------------------------------------------------

// TestStress_CircularGrowthAllHeadPositions verifies that buffer doubling and
// element unwrapping preserves exact FIFO sequence across every possible circular
// head offset (h = 0, 1, ..., cap-1) for a wide variety of initial capacities
// (both powers of 2 and arbitrary non-powers of 2), triggered by both PushBack and PushFront.
func TestStress_CircularGrowthAllHeadPositions(t *testing.T) {
	testCaps := []int{1, 2, 3, 4, 5, 7, 8, 9, 13, 16, 17, 31, 32, 33, 63, 64}

	for _, initCap := range testCaps {
		// Test every possible head offset in [0, initCap - 1]
		for headOffset := 0; headOffset < initCap; headOffset++ {
			// Sub-test A: Growth triggered by PushBack
			{
				d := NewWithCapacity[int](initCap)

				// Step 1: Advance head position to headOffset
				for i := 0; i < headOffset; i++ {
					d.PushBack(-1)
					d.PopFront()
				}
				require.Equal(t, 0, d.Len())

				// Step 2: Fill to capacity with known sequential integers
				for i := 0; i < initCap; i++ {
					d.PushBack(i * 10)
				}
				require.Equal(t, initCap, d.Len())
				require.Equal(t, initCap, d.Cap())

				// Step 3: Trigger growth via PushBack
				newItem := 99999
				d.PushBack(newItem)

				// Step 4: Verify capacity increased and length is correct
				require.Equal(t, initCap+1, d.Len())
				require.GreaterOrEqual(t, d.Cap(), initCap*2)

				// Step 5: Verify all elements retain correct order from front to back
				for i := 0; i < initCap; i++ {
					require.Equal(t, i*10, d.At(i))
				}
				require.Equal(t, newItem, d.At(initCap))

				// Step 6: Verify negative indexing
				require.Equal(t, newItem, d.At(-1))
				require.Equal(t, 0, d.At(-d.Len()))
			}

			// Sub-test B: Growth triggered by PushFront
			{
				d := NewWithCapacity[int](initCap)

				// Advance head position to headOffset
				for i := 0; i < headOffset; i++ {
					d.PushBack(-1)
					d.PopFront()
				}
				require.Equal(t, 0, d.Len())

				// Fill to capacity
				for i := 0; i < initCap; i++ {
					d.PushBack(i * 10)
				}
				require.Equal(t, initCap, d.Len())

				// Trigger growth via PushFront
				frontItem := -88888
				d.PushFront(frontItem)

				require.Equal(t, initCap+1, d.Len())
				require.GreaterOrEqual(t, d.Cap(), initCap*2)

				// Verify front item is at index 0 and rest are shifted by +1
				require.Equal(t, frontItem, d.At(0))
				for i := 0; i < initCap; i++ {
					require.Equal(t, i*10, d.At(i+1))
				}

				// Verify Front and Back
				f, okF := d.Front()
				b, okB := d.Back()
				require.True(t, okF)
				require.True(t, okB)
				require.Equal(t, frontItem, f)
				require.Equal(t, (initCap-1)*10, b)
			}
		}
	}
}

// TestStress_MultiStageSuccessiveGrowth tests multiple successive growth triggers
// (growing across 6 doubling generations from capacity 2 to 128) while repeatedly
// alternating front and back pushes to force irregular head wrap offsets.
func TestStress_MultiStageSuccessiveGrowth(t *testing.T) {
	d := NewWithCapacity[int](2)
	require.Equal(t, 2, d.Cap())

	const totalElements = 250
	for i := 0; i < totalElements; i++ {
		if i%3 == 0 {
			d.PushFront(i)
		} else {
			d.PushBack(i)
		}
	}

	require.Equal(t, totalElements, d.Len())
	require.GreaterOrEqual(t, d.Cap(), totalElements)

	// Pop all elements and verify order against reference
	var popped []int
	for !d.Empty() {
		val, ok := d.PopFront()
		require.True(t, ok)
		popped = append(popped, val)
	}
	require.Equal(t, totalElements, len(popped))
	require.True(t, d.Empty())
}

// -----------------------------------------------------------------------------
// Challenge 2: Alternating 1,000,000 Cycles Stress Test
// -----------------------------------------------------------------------------

// TestStress_Alternating1Million_PushFrontPopBack tests 1,000,000 cycles of alternating
// PushFront and PopBack, verifying continuous wrap-around without memory growth or data corruption.
func TestStress_Alternating1Million_PushFrontPopBack(t *testing.T) {
	// Mode A: Push then Pop (peak elements = queueSize + 1, so cap is 32, steady size 16)
	{
		const queueSize = 16
		d := NewWithCapacity[int](32)

		// Seed with initial elements: [0, 1, ..., queueSize-1]
		for i := 0; i < queueSize; i++ {
			d.PushBack(i)
		}
		initialCap := d.Cap()
		require.Equal(t, 32, initialCap)

		const iterations = 1_000_000
		expected := make([]int, queueSize)
		for i := 0; i < queueSize; i++ {
			expected[i] = i
		}

		for i := 0; i < iterations; i++ {
			newVal := i + 100
			d.PushFront(newVal)
			popped, ok := d.PopBack()
			require.True(t, ok)

			// Model: prepend newVal, pop last
			lastExpected := expected[len(expected)-1]
			expected = append([]int{newVal}, expected[:len(expected)-1]...)

			require.Equal(t, lastExpected, popped)

			// Periodic sanity check
			if i%100_000 == 0 {
				require.Equal(t, queueSize, d.Len())
				require.Equal(t, initialCap, d.Cap())
				require.Equal(t, newVal, d.At(0))
			}
		}

		require.Equal(t, queueSize, d.Len())
		require.Equal(t, initialCap, d.Cap(), "capacity must not grow during steady-state alternating operations")
	}

	// Mode B: Pop then Push on tight capacity (capacity 16, queueSize 16, never exceeds 16)
	{
		const queueSize = 16
		d := NewWithCapacity[int](queueSize)
		for i := 0; i < queueSize; i++ {
			d.PushBack(i)
		}
		require.Equal(t, queueSize, d.Cap())

		for i := 0; i < 100_000; i++ {
			popped, ok := d.PopBack()
			require.True(t, ok)
			_ = popped
			d.PushFront(i + 1000)
		}
		require.Equal(t, queueSize, d.Cap(), "tight capacity must not grow when pop precedes push")
		require.Equal(t, queueSize, d.Len())
	}
}

// TestStress_Alternating1Million_PushBackPopFront tests 1,000,000 cycles of alternating
// PushBack and PopFront (standard FIFO streaming queue behavior).
func TestStress_Alternating1Million_PushBackPopFront(t *testing.T) {
	// Mode A: Push then Pop with room for transient peak
	{
		const queueSize = 32
		d := NewWithCapacity[int](64)

		// Pre-fill queue
		for i := 0; i < queueSize; i++ {
			d.PushBack(i)
		}
		initialCap := d.Cap()
		require.Equal(t, 64, initialCap)

		const iterations = 1_000_000
		for i := 0; i < iterations; i++ {
			newVal := i + queueSize
			d.PushBack(newVal)
			popped, ok := d.PopFront()
			require.True(t, ok)
			require.Equal(t, i, popped)

			if i%100_000 == 0 {
				require.Equal(t, queueSize, d.Len())
				require.Equal(t, initialCap, d.Cap())
			}
		}

		require.Equal(t, queueSize, d.Len())
		require.Equal(t, initialCap, d.Cap(), "steady-state FIFO must never reallocate backing slice")
	}

	// Mode B: Pop then Push on tight capacity 32
	{
		const queueSize = 32
		d := NewWithCapacity[int](queueSize)
		for i := 0; i < queueSize; i++ {
			d.PushBack(i)
		}
		require.Equal(t, queueSize, d.Cap())

		for i := 0; i < 100_000; i++ {
			popped, ok := d.PopFront()
			require.True(t, ok)
			_ = popped
			d.PushBack(i + 5000)
		}
		require.Equal(t, queueSize, d.Cap(), "tight capacity must not grow when pop precedes push")
		require.Equal(t, queueSize, d.Len())
	}
}

// TestStress_Alternating1Million_MixedBurstCycles tests 1,000,000 mixed double-ended operations
// with variable burst sizes (pushes and pops at both ends), validating against a slice oracle.
func TestStress_Alternating1Million_MixedBurstCycles(t *testing.T) {
	d := New[int]()
	var oracle []int

	rng := rand.New(rand.NewPCG(20261001, 123456789))
	const totalOps = 1_000_000

	for op := 0; op < totalOps; op++ {
		action := rng.IntN(4)
		val := op

		switch action {
		case 0: // PushBack
			d.PushBack(val)
			oracle = append(oracle, val)

		case 1: // PushFront
			d.PushFront(val)
			oracle = append([]int{val}, oracle...)

		case 2: // PopFront
			dVal, dOk := d.PopFront()
			if len(oracle) == 0 {
				require.False(t, dOk)
			} else {
				require.True(t, dOk)
				require.Equal(t, oracle[0], dVal)
				oracle = oracle[1:]
			}

		case 3: // PopBack
			dVal, dOk := d.PopBack()
			if len(oracle) == 0 {
				require.False(t, dOk)
			} else {
				require.True(t, dOk)
				require.Equal(t, oracle[len(oracle)-1], dVal)
				oracle = oracle[:len(oracle)-1]
			}
		}

		// Keep queue bounded so memory does not explode
		if d.Len() > 500 {
			for k := 0; k < 250; k++ {
				dVal, _ := d.PopFront()
				require.Equal(t, oracle[0], dVal)
				oracle = oracle[1:]
			}
		}
	}

	require.Equal(t, len(oracle), d.Len())
	for i := 0; i < len(oracle); i++ {
		require.Equal(t, oracle[i], d.At(i))
	}
}

// -----------------------------------------------------------------------------
// Challenge 3: Dual GC Leak Protection (Weak Pointers & Finalizers)
// -----------------------------------------------------------------------------

// TestStress_GCLeak_WeakPointers_Massive verifies that evacuated slots after PopFront,
// PopBack, and Clear are completely zeroed and do not retain pointers in memory.
func TestStress_GCLeak_WeakPointers_Massive(t *testing.T) {
	const count = 1000
	d := New[*int]()

	// Shift head to test wrapped slot evacuation
	for i := 0; i < 50; i++ {
		dummy := new(int)
		d.PushBack(dummy)
		d.PopFront()
	}

	wps := make([]weak.Pointer[int], count)
	for i := 0; i < count; i++ {
		func(idx int) {
			obj := new(int)
			*obj = idx
			wps[idx] = weak.Make(obj)
			if idx%2 == 0 {
				d.PushBack(obj)
			} else {
				d.PushFront(obj)
			}
		}(i)
	}

	require.Equal(t, count, d.Len())

	// Verify all weak pointers are alive while in deque
	runtime.GC()
	for i := 0; i < count; i++ {
		require.NotNil(t, wps[i].Value())
	}

	// Pop 250 from Front
	for i := 0; i < 250; i++ {
		val, ok := d.PopFront()
		require.True(t, ok)
		_ = val
	}

	// Pop 250 from Back
	for i := 0; i < 250; i++ {
		val, ok := d.PopBack()
		require.True(t, ok)
		_ = val
	}

	require.Equal(t, count-500, d.Len())

	// Clear the remaining 500
	d.Clear()
	require.Equal(t, 0, d.Len())
	require.True(t, d.Empty())

	// Force garbage collection
	for retry := 0; retry < 5; retry++ {
		runtime.GC()
		runtime.Gosched()
	}

	// All 1,000 objects must have been collected
	leakedCount := 0
	for i := 0; i < count; i++ {
		if wps[i].Value() != nil {
			leakedCount++
		}
	}
	require.Equal(t, 0, leakedCount, "weak pointers retained: memory leak detected in deque slots")
}

// TestStress_GCLeak_Finalizers verifies that finalizers on objects stored in the deque
// reliably run after PopFront, PopBack, and Clear.
func TestStress_GCLeak_Finalizers(t *testing.T) {
	type finalizerTarget struct {
		payload [64]byte
		val     int
	}

	const count = 300
	var finalizerRunCount atomic.Int32

	d := New[*finalizerTarget]()

	// Pre-shift head
	for i := 0; i < 30; i++ {
		d.PushBack(&finalizerTarget{})
		d.PopFront()
	}

	for i := 0; i < count; i++ {
		func(id int) {
			target := &finalizerTarget{val: id}
			runtime.SetFinalizer(target, func(_ *finalizerTarget) {
				finalizerRunCount.Add(1)
			})
			if id%2 == 0 {
				d.PushBack(target)
			} else {
				d.PushFront(target)
			}
		}(i)
	}

	require.Equal(t, count, d.Len())

	// Pop half
	for i := 0; i < count/2; i++ {
		_, ok := d.PopFront()
		require.True(t, ok)
	}

	// Clear the other half
	d.Clear()
	require.True(t, d.Empty())

	// Wait for finalizers with timeout
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && finalizerRunCount.Load() < int32(count) {
		runtime.GC()
		runtime.Gosched()
		time.Sleep(5 * time.Millisecond)
	}

	require.Equal(t, int32(count), finalizerRunCount.Load(),
		"finalizers failed to execute: references retained in deque")
}

// TestStress_GCLeak_Drain verifies that draining a deque clears all slot references.
func TestStress_GCLeak_Drain(t *testing.T) {
	const count = 200
	d := New[*int]()

	wps := make([]weak.Pointer[int], count)
	for i := 0; i < count; i++ {
		func(idx int) {
			obj := new(int)
			*obj = idx
			wps[idx] = weak.Make(obj)
			d.PushBack(obj)
		}(i)
	}

	// Drain all items
	drainedCount := 0
	for range d.Drain() {
		drainedCount++
	}
	require.Equal(t, count, drainedCount)
	require.True(t, d.Empty())

	for retry := 0; retry < 5; retry++ {
		runtime.GC()
		runtime.Gosched()
	}

	for i := 0; i < count; i++ {
		require.Nil(t, wps[i].Value(), "Drain() failed to clear slot references")
	}
}

// -----------------------------------------------------------------------------
// Challenge 4: Multi-Goroutine Read Contention & Synchronized Writers
// -----------------------------------------------------------------------------

// TestStress_MultiGoroutineReadContention tests 64 concurrent goroutines continuously
// reading from a pre-populated deque across all read APIs under -race.
func TestStress_MultiGoroutineReadContention(t *testing.T) {
	const size = 5000
	d := NewWithCapacity[int](size)

	// Shift head into the middle to ensure readers read wrapped buffer indices
	for i := 0; i < size/2; i++ {
		d.PushBack(-1)
		d.PopFront()
	}
	for i := 0; i < size; i++ {
		d.PushBack(i * 7)
	}
	require.Equal(t, size, d.Len())

	const numGoroutines = 64
	const iterations = 500
	var wg sync.WaitGroup

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(gid int) {
			defer wg.Done()
			for iter := 0; iter < iterations; iter++ {
				// Inspection queries
				require.Equal(t, size, d.Len())
				require.False(t, d.Empty())
				require.GreaterOrEqual(t, d.Cap(), size)

				// Peek
				f, okF := d.Front()
				b, okB := d.Back()
				require.True(t, okF)
				require.True(t, okB)
				require.Equal(t, 0, f)
				require.Equal(t, (size-1)*7, b)

				// Random access (positive and negative)
				idx := (gid*31 + iter*17) % size
				expectedVal := idx * 7
				require.Equal(t, expectedVal, d.At(idx))

				negIdx := -1 - ((gid*13 + iter*7) % size)
				expectedNegVal := (size + negIdx) * 7
				require.Equal(t, expectedNegVal, d.At(negIdx))

				// Push iterators with early break
				count := 0
				for val := range d.Values() {
					require.Equal(t, count*7, val)
					count++
					if count >= 10 {
						break
					}
				}

				count = 0
				for val := range d.Backward() {
					require.Equal(t, (size-1-count)*7, val)
					count++
					if count >= 10 {
						break
					}
				}
			}
		}(g)
	}

	wg.Wait()
}

// TestStress_SynchronizedReadWriteContention tests 32 concurrent goroutines
// (16 readers, 16 writers) under high lock contention with sync.RWMutex.
func TestStress_SynchronizedReadWriteContention(t *testing.T) {
	d := NewWithCapacity[int](64)
	var rw sync.RWMutex
	var wg sync.WaitGroup

	const readers = 16
	const writers = 16
	const opsPerWorker = 300

	// Readers
	for r := 0; r < readers; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < opsPerWorker; i++ {
				rw.RLock()
				length := d.Len()
				empty := d.Empty()
				if !empty && length > 0 {
					_ = d.At(0)
					_ = d.At(-1)
					_, _ = d.Front()
					_, _ = d.Back()
				}
				rw.RUnlock()
				runtime.Gosched()
			}
		}()
	}

	// Writers
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(wid int) {
			defer wg.Done()
			for i := 0; i < opsPerWorker; i++ {
				rw.Lock()
				val := wid*10_000 + i

				switch i % 5 {
				case 0:
					d.PushBack(val)
				case 1:
					d.PushFront(val)
				case 2:
					if !d.Empty() {
						d.PopFront()
					}
				case 3:
					if !d.Empty() {
						d.PopBack()
					}
				case 4:
					if d.Len() > 0 {
						d.Set(0, val)
					}
				}

				// Keep queue bounded
				if d.Len() > 100 {
					d.PopFront()
				}
				rw.Unlock()
				runtime.Gosched()
			}
		}(w)
	}

	wg.Wait()
}

// TestStress_HeavyProducerConsumerPipeline validates high-throughput FIFO pipeline
// across 8 producers and 8 consumers totaling 40,000 items under -race.
func TestStress_HeavyProducerConsumerPipeline(t *testing.T) {
	d := New[int]()
	var mu sync.Mutex
	var wg sync.WaitGroup

	const producers = 8
	const consumers = 8
	const itemsPerProducer = 5000
	const totalItems = producers * itemsPerProducer

	var consumedCount atomic.Int64
	var consumedSum atomic.Int64

	// Producers
	for p := 0; p < producers; p++ {
		wg.Add(1)
		go func(pid int) {
			defer wg.Done()
			for i := 0; i < itemsPerProducer; i++ {
				val := pid*itemsPerProducer + i + 1
				mu.Lock()
				if i%2 == 0 {
					d.PushBack(val)
				} else {
					d.PushFront(val)
				}
				mu.Unlock()
			}
		}(p)
	}

	// Consumers
	for c := 0; c < consumers; c++ {
		wg.Add(1)
		go func(cid int) {
			defer wg.Done()
			for consumedCount.Load() < int64(totalItems) {
				mu.Lock()
				var val int
				var ok bool
				if cid%2 == 0 {
					val, ok = d.PopFront()
				} else {
					val, ok = d.PopBack()
				}
				mu.Unlock()

				if ok {
					consumedSum.Add(int64(val))
					consumedCount.Add(1)
				} else {
					runtime.Gosched()
				}
			}
		}(c)
	}

	wg.Wait()
	require.Equal(t, int64(totalItems), consumedCount.Load())
	expectedSum := int64(totalItems) * int64(totalItems+1) / 2
	require.Equal(t, expectedSum, consumedSum.Load())
}
