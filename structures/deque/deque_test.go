// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package deque

import (
	"math/rand"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"weak"

	"github.com/lemon4ksan/foundation/testing/require"
)

func TestConstructors(t *testing.T) {
	// New creates empty deque
	d := New[int]()
	require.Equal(t, 0, d.Len())
	require.Equal(t, 0, d.Cap())
	require.True(t, d.Empty())

	// NewWithCapacity panics on negative capacity
	require.PanicsWithError(t, "deque: negative capacity", func() {
		NewWithCapacity[int](-1)
	})
	require.PanicsWithError(t, "deque: negative capacity", func() {
		NewWithCapacity[string](-100)
	})

	// NewWithCapacity with zero
	d0 := NewWithCapacity[int](0)
	require.Equal(t, 0, d0.Len())
	require.Equal(t, 0, d0.Cap())
	require.True(t, d0.Empty())

	// NewWithCapacity with positive values
	for _, capVal := range []int{1, 4, 16, 1024} {
		dc := NewWithCapacity[int](capVal)
		require.Equal(t, 0, dc.Len())
		require.Equal(t, capVal, dc.Cap())
		require.True(t, dc.Empty())
	}
}

func TestNilReceiver(t *testing.T) {
	var nilD *Deque[int]

	require.Equal(t, 0, nilD.Len())
	require.Equal(t, 0, nilD.Cap())
	require.True(t, nilD.Empty())

	// Clear on nil should not panic
	nilD.Clear()

	// Peeking on nil
	v, ok := nilD.Front()
	require.False(t, ok)
	require.Equal(t, 0, v)

	v, ok = nilD.Back()
	require.False(t, ok)
	require.Equal(t, 0, v)

	// Pop on nil
	v, ok = nilD.PopFront()
	require.False(t, ok)
	require.Equal(t, 0, v)

	v, ok = nilD.PopBack()
	require.False(t, ok)
	require.Equal(t, 0, v)

	// MustPop on nil panics
	require.PanicsWithError(t, "deque: pop from an empty deque", func() {
		nilD.MustPopFront()
	})
	require.PanicsWithError(t, "deque: pop from an empty deque", func() {
		nilD.MustPopBack()
	})

	// At and Set on nil panic
	require.PanicsWithError(t, "deque: index out of range", func() {
		nilD.At(0)
	})
	require.PanicsWithError(t, "deque: index out of range", func() {
		nilD.Set(0, 10)
	})

	// Iterators on nil yield nothing
	for range nilD.Values() {
		t.Fatal("nil deque should yield nothing on Values()")
	}
	for range nilD.All() {
		t.Fatal("nil deque should yield nothing on All()")
	}
	for range nilD.Backward() {
		t.Fatal("nil deque should yield nothing on Backward()")
	}
	for range nilD.Drain() {
		t.Fatal("nil deque should yield nothing on Drain()")
	}
}

func TestZeroValueReceiver(t *testing.T) {
	var d Deque[int]

	require.Equal(t, 0, d.Len())
	require.Equal(t, 0, d.Cap())
	require.True(t, d.Empty())

	d.Clear()
	require.Equal(t, 0, d.Len())

	// PushBack on zero value lazily initializes capacity
	d.PushBack(100)
	require.Equal(t, 1, d.Len())
	require.GreaterOrEqual(t, d.Cap(), 4)
	require.False(t, d.Empty())
	require.Equal(t, 100, d.At(0))

	// PushFront on fresh zero value
	var d2 Deque[string]
	d2.PushFront("hello")
	require.Equal(t, 1, d2.Len())
	require.GreaterOrEqual(t, d2.Cap(), 4)
	require.Equal(t, "hello", d2.At(0))
}

func TestPushPopFIFO(t *testing.T) {
	d := New[int]()

	for i := 0; i < 10; i++ {
		d.PushBack(i)
		require.Equal(t, i+1, d.Len())
	}

	for i := 0; i < 10; i++ {
		v, ok := d.PopFront()
		require.True(t, ok)
		require.Equal(t, i, v)
		require.Equal(t, 9-i, d.Len())
	}

	require.True(t, d.Empty())
	v, ok := d.PopFront()
	require.False(t, ok)
	require.Equal(t, 0, v)
}

func TestPushPopLIFO(t *testing.T) {
	// LIFO at Back
	d := New[int]()
	for i := 0; i < 10; i++ {
		d.PushBack(i)
	}
	for i := 9; i >= 0; i-- {
		v, ok := d.PopBack()
		require.True(t, ok)
		require.Equal(t, i, v)
	}
	require.True(t, d.Empty())

	// LIFO at Front
	for i := 0; i < 10; i++ {
		d.PushFront(i)
	}
	for i := 9; i >= 0; i-- {
		v, ok := d.PopFront()
		require.True(t, ok)
		require.Equal(t, i, v)
	}
	require.True(t, d.Empty())
}

func TestPushPopReverseFIFO(t *testing.T) {
	d := New[int]()
	for i := 0; i < 10; i++ {
		d.PushFront(i) // front: 9, 8, ..., 0
	}
	for i := 0; i < 10; i++ {
		v, ok := d.PopBack() // back popped first: 0, 1, ..., 9
		require.True(t, ok)
		require.Equal(t, i, v)
	}
	require.True(t, d.Empty())
}

func TestEmptyPopAndPeek(t *testing.T) {
	d := New[string]()

	v, ok := d.PopFront()
	require.False(t, ok)
	require.Equal(t, "", v)

	v, ok = d.PopBack()
	require.False(t, ok)
	require.Equal(t, "", v)

	v, ok = d.Front()
	require.False(t, ok)
	require.Equal(t, "", v)

	v, ok = d.Back()
	require.False(t, ok)
	require.Equal(t, "", v)
}

func TestMustPopPanics(t *testing.T) {
	d := New[int]()

	require.PanicsWithError(t, "deque: pop from an empty deque", func() {
		d.MustPopFront()
	})
	require.PanicsWithError(t, "deque: pop from an empty deque", func() {
		d.MustPopBack()
	})

	d.PushBack(42)
	require.Equal(t, 42, d.MustPopFront())

	d.PushFront(99)
	require.Equal(t, 99, d.MustPopBack())

	require.PanicsWithError(t, "deque: pop from an empty deque", func() {
		d.MustPopFront()
	})
	require.PanicsWithError(t, "deque: pop from an empty deque", func() {
		d.MustPopBack()
	})
}

func TestFrontAndBack(t *testing.T) {
	d := NewWithCapacity[int](8)

	// Shift head to index 6 to test wrapped back index
	for i := 0; i < 6; i++ {
		d.PushBack(0)
		d.PopFront()
	}

	d.PushBack(10)
	d.PushBack(20)
	d.PushBack(30)

	f, ok := d.Front()
	require.True(t, ok)
	require.Equal(t, 10, f)

	b, ok := d.Back()
	require.True(t, ok)
	require.Equal(t, 30, b)

	// Ensure Front and Back do not mutate length
	require.Equal(t, 3, d.Len())
}

func TestRandomAccess_At_Set(t *testing.T) {
	d := NewWithCapacity[int](4)

	// Shift head
	d.PushBack(0)
	d.PushBack(0)
	d.PopFront()
	d.PopFront() // head is now 2

	d.PushBack(10) // index 0 (phys 2)
	d.PushBack(20) // index 1 (phys 3)
	d.PushBack(30) // index 2 (phys 0 - wrapped)
	require.Equal(t, 3, d.Len())

	// Positive indexing
	require.Equal(t, 10, d.At(0))
	require.Equal(t, 20, d.At(1))
	require.Equal(t, 30, d.At(2))

	// Negative indexing
	require.Equal(t, 30, d.At(-1))
	require.Equal(t, 20, d.At(-2))
	require.Equal(t, 10, d.At(-3))

	// Out of bounds panics
	require.PanicsWithError(t, "deque: index out of range", func() { d.At(3) })
	require.PanicsWithError(t, "deque: index out of range", func() { d.At(100) })
	require.PanicsWithError(t, "deque: index out of range", func() { d.At(-4) })
	require.PanicsWithError(t, "deque: index out of range", func() { d.At(-100) })

	// Set with positive and negative indices
	d.Set(0, 11)
	require.Equal(t, 11, d.At(0))

	d.Set(-1, 33)
	require.Equal(t, 33, d.At(2))
	require.Equal(t, 33, d.At(-1))

	d.Set(1, 22)
	require.Equal(t, 22, d.At(1))

	// Set out of bounds panics
	require.PanicsWithError(t, "deque: index out of range", func() { d.Set(3, 99) })
	require.PanicsWithError(t, "deque: index out of range", func() { d.Set(-4, 99) })
}

func TestClear(t *testing.T) {
	d := NewWithCapacity[int](16)
	for i := 0; i < 10; i++ {
		d.PushBack(i)
	}
	require.Equal(t, 10, d.Len())
	require.Equal(t, 16, d.Cap())

	d.Clear()
	require.Equal(t, 0, d.Len())
	require.Equal(t, 16, d.Cap()) // capacity preserved
	require.True(t, d.Empty())

	// Can push again without allocation
	d.PushBack(100)
	require.Equal(t, 1, d.Len())
	require.Equal(t, 100, d.At(0))
}

func TestBufferGrowth_Unwrapped(t *testing.T) {
	d := NewWithCapacity[int](4)
	for i := 0; i < 4; i++ {
		d.PushBack(i)
	}
	require.Equal(t, 4, d.Cap())

	// Push 5th element triggers growth with head == 0
	d.PushBack(4)
	require.Equal(t, 8, d.Cap())
	require.Equal(t, 5, d.Len())

	for i := 0; i < 5; i++ {
		require.Equal(t, i, d.At(i))
	}
}

func TestBufferGrowth_Wrapped(t *testing.T) {
	d := NewWithCapacity[int](4)
	d.PushBack(99)
	d.PushBack(99)
	d.PopFront()
	d.PopFront() // head is at index 2

	// Fill to capacity
	d.PushBack(10) // phys 2
	d.PushBack(20) // phys 3
	d.PushBack(30) // phys 0 (wrapped)
	d.PushBack(40) // phys 1 (wrapped)
	require.Equal(t, 4, d.Len())

	// Growth triggers while wrapped across boundary
	d.PushBack(50)
	require.Equal(t, 8, d.Cap())
	require.Equal(t, 5, d.Len())

	expected := []int{10, 20, 30, 40, 50}
	for i, exp := range expected {
		require.Equal(t, exp, d.At(i))
	}
}

func TestBufferGrowth_SmallCap(t *testing.T) {
	// From cap 1, doubles to 2, then min 4
	d := NewWithCapacity[int](1)
	d.PushBack(1)
	d.PushBack(2)
	require.Equal(t, 4, d.Cap())
	require.Equal(t, 2, d.Len())
	require.Equal(t, 1, d.At(0))
	require.Equal(t, 2, d.At(1))
}

func TestIterators_Values(t *testing.T) {
	// Empty deque iteration
	emptyD := New[int]()
	for range emptyD.Values() {
		t.Fatal("empty deque should yield nothing")
	}

	d := NewWithCapacity[int](8)
	// Shift head to test wrapped iterator
	for i := 0; i < 6; i++ {
		d.PushBack(0)
		d.PopFront()
	}
	for i := 0; i < 5; i++ {
		d.PushBack(i)
	}

	// Full traversal
	var collected []int
	for v := range d.Values() {
		collected = append(collected, v)
	}
	require.Equal(t, []int{0, 1, 2, 3, 4}, collected)

	// Early termination
	collected = nil
	for v := range d.Values() {
		collected = append(collected, v)
		if len(collected) == 3 {
			break
		}
	}
	require.Equal(t, []int{0, 1, 2}, collected)
	require.Equal(t, 5, d.Len()) // deque not mutated
}

func TestIterators_All(t *testing.T) {
	// Empty deque iteration
	emptyD := New[string]()
	for range emptyD.All() {
		t.Fatal("empty deque should yield nothing")
	}

	d := NewWithCapacity[string](8)
	// Shift head to test wrapped iterator
	for i := 0; i < 6; i++ {
		d.PushBack("")
		d.PopFront()
	}
	d.PushBack("a")
	d.PushBack("b")
	d.PushBack("c")

	indices := []int{}
	values := []string{}
	for i, v := range d.All() {
		indices = append(indices, i)
		values = append(values, v)
	}
	require.Equal(t, []int{0, 1, 2}, indices)
	require.Equal(t, []string{"a", "b", "c"}, values)

	// Early termination
	count := 0
	for range d.All() {
		count++
		break
	}
	require.Equal(t, 1, count)
}

func TestIterators_Backward(t *testing.T) {
	// Empty deque iteration
	emptyD := New[int]()
	for range emptyD.Backward() {
		t.Fatal("empty deque should yield nothing")
	}

	d := NewWithCapacity[int](8)
	// Shift head to test wrapped iterator
	for i := 0; i < 6; i++ {
		d.PushBack(0)
		d.PopFront()
	}
	d.PushBack(10)
	d.PushBack(20)
	d.PushBack(30)

	var collected []int
	for v := range d.Backward() {
		collected = append(collected, v)
	}
	require.Equal(t, []int{30, 20, 10}, collected)

	// Early termination
	collected = nil
	for v := range d.Backward() {
		collected = append(collected, v)
		break
	}
	require.Equal(t, []int{30}, collected)
}

func TestIterators_Drain(t *testing.T) {
	// Empty deque drain
	emptyD := New[int]()
	for range emptyD.Drain() {
		t.Fatal("empty deque should yield nothing on Drain")
	}

	d := New[int]()
	for i := 0; i < 5; i++ {
		d.PushBack(i)
	}

	// Partial drain with break
	var collected []int
	for v := range d.Drain() {
		collected = append(collected, v)
		if len(collected) == 2 {
			break
		}
	}
	require.Equal(t, []int{0, 1}, collected)
	require.Equal(t, 3, d.Len()) // items 2, 3, 4 remain

	// Finish draining
	collected = nil
	for v := range d.Drain() {
		collected = append(collected, v)
	}
	require.Equal(t, []int{2, 3, 4}, collected)
	require.True(t, d.Empty())
}

// -----------------------------------------------------------------------------
// GC Memory Leak Tests
// -----------------------------------------------------------------------------

func TestGCLeak_WeakPointers(t *testing.T) {
	d := New[*int]()

	// Push 10 elements tracked by weak pointers
	wps := make([]weak.Pointer[int], 10)
	for i := 0; i < 10; i++ {
		func(idx int) {
			obj := new(int)
			*obj = idx
			wps[idx] = weak.Make(obj)
			d.PushBack(obj)
		}(i)
	}

	// Verify weak pointers are still valid while in queue
	runtime.GC()
	for i := 0; i < 10; i++ {
		require.NotNil(t, wps[i].Value())
	}

	// Pop 5 from front
	for i := 0; i < 5; i++ {
		popped, ok := d.PopFront()
		require.True(t, ok)
		require.Equal(t, i, *popped)
		_ = popped // drop local variable
	}

	// Pop 5 from back
	for i := 9; i >= 5; i-- {
		popped, ok := d.PopBack()
		require.True(t, ok)
		require.Equal(t, i, *popped)
		_ = popped // drop local variable
	}

	require.True(t, d.Empty())

	// Run GC; all 10 objects should be collected because slots were zeroed
	for retry := 0; retry < 5; retry++ {
		runtime.GC()
		runtime.Gosched()
	}

	for i := 0; i < 10; i++ {
		require.Nil(t, wps[i].Value(), "memory leak: popped slot retained pointer")
	}
}

func TestGCLeak_Clear_WeakPointers(t *testing.T) {
	d := New[*int]()
	wps := make([]weak.Pointer[int], 16)
	for i := 0; i < 16; i++ {
		func(idx int) {
			obj := new(int)
			*obj = idx
			wps[idx] = weak.Make(obj)
			d.PushBack(obj)
		}(i)
	}

	d.Clear()
	require.True(t, d.Empty())

	for retry := 0; retry < 5; retry++ {
		runtime.GC()
		runtime.Gosched()
	}

	for i := 0; i < 16; i++ {
		require.Nil(t, wps[i].Value(), "memory leak: Clear() failed to zero slots")
	}
}

func TestGCLeak_Finalizers(t *testing.T) {
	type trackedNode struct {
		id int
	}

	var finalized atomic.Int32
	d := New[*trackedNode]()

	const count = 50
	for i := 0; i < count; i++ {
		func(idx int) {
			node := &trackedNode{id: idx}
			runtime.SetFinalizer(node, func(_ *trackedNode) {
				finalized.Add(1)
			})
			d.PushBack(node)
		}(i)
	}

	// Pop all
	for i := 0; i < count; i++ {
		_, ok := d.PopFront()
		require.True(t, ok)
	}

	// Trigger GC repeatedly
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && finalized.Load() < count {
		runtime.GC()
		runtime.Gosched()
		time.Sleep(5 * time.Millisecond)
	}

	require.Equal(t, int32(count), finalized.Load(), "finalizers failed to run: memory leak in pop")
}

// -----------------------------------------------------------------------------
// Concurrency & Race Tests
// -----------------------------------------------------------------------------

func TestConcurrentReaders(t *testing.T) {
	const n = 2000
	d := NewWithCapacity[int](n)
	for i := 0; i < n; i++ {
		d.PushBack(i * 3)
	}

	var wg sync.WaitGroup
	const goroutines = 32
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(gid int) {
			defer wg.Done()
			for iter := 0; iter < 200; iter++ {
				_ = d.Len()
				_ = d.Cap()
				_ = d.Empty()
				f, _ := d.Front()
				b, _ := d.Back()
				_ = f + b

				idx := (gid*17 + iter) % n
				_ = d.At(idx)
				negIdx := -1 - ((gid*13 + iter) % n)
				_ = d.At(negIdx)

				// Iterators with early exit
				cnt := 0
				for range d.Values() {
					cnt++
					if cnt > 10 {
						break
					}
				}
				cnt = 0
				for range d.Backward() {
					cnt++
					if cnt > 10 {
						break
					}
				}
			}
		}(g)
	}
	wg.Wait()
}

func TestConcurrentSynchronizedWriters(t *testing.T) {
	d := NewWithCapacity[int](128)
	var mu sync.RWMutex
	var wg sync.WaitGroup

	// Readers
	for r := 0; r < 16; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				mu.RLock()
				_ = d.Len()
				_ = d.Empty()
				if d.Len() > 0 {
					_ = d.At(0)
					_, _ = d.Front()
					_, _ = d.Back()
				}
				mu.RUnlock()
			}
		}()
	}

	// Writers
	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func(wid int) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				mu.Lock()
				if wid%2 == 0 {
					d.PushBack(wid*1000 + i)
					if d.Len() > 50 {
						d.PopFront()
					}
				} else {
					d.PushFront(wid*1000 + i)
					if d.Len() > 50 {
						d.PopBack()
					}
				}
				mu.Unlock()
			}
		}(w)
	}

	wg.Wait()
}

func TestConcurrentPipeline_ProducerConsumer(t *testing.T) {
	d := New[int]()
	var mu sync.Mutex
	var wg sync.WaitGroup

	const producers = 4
	const consumers = 4
	const itemsPerProducer = 500
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
				d.PushBack(val)
				mu.Unlock()
			}
		}(p)
	}

	// Consumers
	for c := 0; c < consumers; c++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for consumedCount.Load() < totalItems {
				mu.Lock()
				val, ok := d.PopFront()
				mu.Unlock()
				if ok {
					consumedSum.Add(int64(val))
					consumedCount.Add(1)
				} else {
					runtime.Gosched()
				}
			}
		}()
	}

	wg.Wait()
	require.Equal(t, int64(totalItems), consumedCount.Load())
	expectedSum := int64(totalItems) * int64(totalItems+1) / 2
	require.Equal(t, expectedSum, consumedSum.Load())
}

// -----------------------------------------------------------------------------
// Invariant Challenge & Boundary Tests
// -----------------------------------------------------------------------------

// TestChallenge_DifferentialSliceModel tests Deque against an authoritative reference slice model
// across 50,000 randomized operations.
func TestChallenge_DifferentialSliceModel(t *testing.T) {
	rng := rand.New(rand.NewSource(20261001))
	d := New[int]()
	var ref []int

	const operations = 50_000
	for op := 0; op < operations; op++ {
		action := rng.Intn(10)
		val := rng.Intn(1000_000)

		switch action {
		case 0, 1: // PushBack
			d.PushBack(val)
			ref = append(ref, val)

		case 2, 3: // PushFront
			d.PushFront(val)
			ref = append([]int{val}, ref...)

		case 4: // PopFront
			dVal, dOk := d.PopFront()
			if len(ref) == 0 {
				require.False(t, dOk)
			} else {
				require.True(t, dOk)
				require.Equal(t, ref[0], dVal)
				ref = ref[1:]
			}

		case 5: // PopBack
			dVal, dOk := d.PopBack()
			if len(ref) == 0 {
				require.False(t, dOk)
			} else {
				require.True(t, dOk)
				require.Equal(t, ref[len(ref)-1], dVal)
				ref = ref[:len(ref)-1]
			}

		case 6: // Front and Back peek
			fVal, fOk := d.Front()
			bVal, bOk := d.Back()
			if len(ref) == 0 {
				require.False(t, fOk)
				require.False(t, bOk)
			} else {
				require.True(t, fOk)
				require.True(t, bOk)
				require.Equal(t, ref[0], fVal)
				require.Equal(t, ref[len(ref)-1], bVal)
			}

		case 7: // Random access At and Set
			if len(ref) > 0 {
				idx := rng.Intn(len(ref))
				require.Equal(t, ref[idx], d.At(idx))
				require.Equal(t, ref[len(ref)-1-idx], d.At(-1-idx))

				newVal := rng.Intn(1000_000)
				d.Set(idx, newVal)
				ref[idx] = newVal
				require.Equal(t, newVal, d.At(idx))
			}

		case 8: // Values iterator matches ref
			collected := slices.Collect(d.Values())
			if len(ref) == 0 {
				require.Equal(t, 0, len(collected))
			} else {
				require.Equal(t, ref, collected)
			}

		case 9: // Clear occasionally
			if rng.Intn(500) == 0 {
				d.Clear()
				ref = nil
			}
		}

		require.Equal(t, len(ref), d.Len())
		require.Equal(t, len(ref) == 0, d.Empty())
	}
}

// TestChallenge_RingWrapGrowthExhaustive tests buffer growth unwrapping at every possible
// head offset for capacities from 1 to 32.
func TestChallenge_RingWrapGrowthExhaustive(t *testing.T) {
	for initCap := 1; initCap <= 32; initCap++ {
		for headOffset := 0; headOffset < initCap; headOffset++ {
			d := NewWithCapacity[int](initCap)

			// Advance head to headOffset
			for i := 0; i < headOffset; i++ {
				d.PushBack(0)
				d.PopFront()
			}

			// Fill to initial capacity
			for i := 0; i < initCap; i++ {
				d.PushBack(i)
			}
			require.Equal(t, initCap, d.Len())

			// Trigger capacity growth
			d.PushBack(initCap)
			require.Equal(t, initCap+1, d.Len())

			// Verify all elements are preserved in exact order
			for i := 0; i <= initCap; i++ {
				require.Equal(t, i, d.At(i))
			}
		}
	}
}

// TestChallenge_BoundaryIndexTable tests exact boundary panic edges for lengths 0 to 64.
func TestChallenge_BoundaryIndexTable(t *testing.T) {
	for n := 0; n <= 64; n++ {
		d := New[int]()
		for i := 0; i < n; i++ {
			d.PushBack(i)
		}

		// Negative boundary
		require.PanicsWithError(t, "deque: index out of range", func() { d.At(-n - 1) })
		require.PanicsWithError(t, "deque: index out of range", func() { d.Set(-n-1, 0) })

		if n > 0 {
			require.Equal(t, 0, d.At(-n))
			require.Equal(t, n-1, d.At(-1))
			require.Equal(t, 0, d.At(0))
			require.Equal(t, n-1, d.At(n-1))
		}

		// Positive boundary
		require.PanicsWithError(t, "deque: index out of range", func() { d.At(n) })
		require.PanicsWithError(t, "deque: index out of range", func() { d.Set(n, 0) })
	}
}
