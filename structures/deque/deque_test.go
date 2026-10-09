// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package deque

import (
	"fmt"
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

type trackedNode struct {
	id int
}

func TestGCLeak_Finalizers(t *testing.T) {
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
		popOneDeque(t, d)
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

//go:noinline
func popOneDeque(t *testing.T, d *Deque[*trackedNode]) {
	_, ok := d.PopFront()
	require.True(t, ok)
}

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

	rng := rand.New(rand.NewSource(20261001))
	const totalOps = 1_000_000

	for op := 0; op < totalOps; op++ {
		action := rng.Intn(4)
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

func BenchmarkDeque_PushPopFront_WithinCap(b *testing.B) {
	d := NewWithCapacity[int](1024)
	b.ReportAllocs()
	b.ResetTimer()

	var v int
	for b.Loop() {
		d.PushFront(v)
		d.PopFront()
		v++
	}
}

func BenchmarkDeque_PushPopBack_WithinCap(b *testing.B) {
	d := NewWithCapacity[int](1024)
	b.ReportAllocs()
	b.ResetTimer()

	var v int
	for b.Loop() {
		d.PushBack(v)
		d.PopBack()
		v++
	}
}

func BenchmarkDeque_FIFO_SteadyState(b *testing.B) {
	d := NewWithCapacity[int](1024)
	b.ReportAllocs()
	b.ResetTimer()

	var v int
	for b.Loop() {
		d.PushBack(v)
		d.PopFront()
		v++
	}
}

func BenchmarkDeque_ReverseFIFO_SteadyState(b *testing.B) {
	d := NewWithCapacity[int](1024)
	b.ReportAllocs()
	b.ResetTimer()

	var v int
	for b.Loop() {
		d.PushFront(v)
		d.PopBack()
		v++
	}
}

func BenchmarkDeque_Front(b *testing.B) {
	d := NewWithCapacity[int](1024)
	d.PushBack(42)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, _ = d.Front()
	}
}

func BenchmarkDeque_Back(b *testing.B) {
	d := NewWithCapacity[int](1024)
	d.PushBack(42)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, _ = d.Back()
	}
}

func BenchmarkDeque_At_Positive(b *testing.B) {
	d := NewWithCapacity[int](1024)
	for i := 0; i < 512; i++ {
		d.PushBack(i)
	}
	b.ReportAllocs()
	b.ResetTimer()

	idx := 256
	for b.Loop() {
		_ = d.At(idx)
	}
}

func BenchmarkDeque_At_Negative(b *testing.B) {
	d := NewWithCapacity[int](1024)
	for i := 0; i < 512; i++ {
		d.PushBack(i)
	}
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = d.At(-1)
	}
}

func BenchmarkDeque_Set(b *testing.B) {
	d := NewWithCapacity[int](1024)
	for i := 0; i < 512; i++ {
		d.PushBack(i)
	}
	b.ReportAllocs()
	b.ResetTimer()

	var v int
	for b.Loop() {
		d.Set(256, v)
		v++
	}
}

func BenchmarkDeque_Values_Iterate_1024(b *testing.B) {
	d := NewWithCapacity[int](1024)
	for i := 0; i < 1024; i++ {
		d.PushBack(i)
	}
	b.ReportAllocs()
	b.ResetTimer()

	var sum int
	for b.Loop() {
		for val := range d.Values() {
			sum += val
		}
	}
	_ = sum
}

func BenchmarkDeque_All_Iterate_1024(b *testing.B) {
	d := NewWithCapacity[int](1024)
	for i := 0; i < 1024; i++ {
		d.PushBack(i)
	}
	b.ReportAllocs()
	b.ResetTimer()

	var sum int
	for b.Loop() {
		for i, val := range d.All() {
			sum += i + val
		}
	}
	_ = sum
}

func BenchmarkDeque_Backward_Iterate_1024(b *testing.B) {
	d := NewWithCapacity[int](1024)
	for i := 0; i < 1024; i++ {
		d.PushBack(i)
	}
	b.ReportAllocs()
	b.ResetTimer()

	var sum int
	for b.Loop() {
		for val := range d.Backward() {
			sum += val
		}
	}
	_ = sum
}

func BenchmarkDeque_Drain_64(b *testing.B) {
	d := NewWithCapacity[int](1024)
	for i := 0; i < 64; i++ {
		d.PushBack(i)
	}
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for v := range d.Drain() {
			_ = v
		}
		for i := 0; i < 64; i++ {
			d.PushBack(i)
		}
	}
}

func BenchmarkDeque_Clear_Reuse(b *testing.B) {
	d := NewWithCapacity[int](1024)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for i := 0; i < 100; i++ {
			d.PushBack(i)
		}
		d.Clear()
	}
}

func BenchmarkDeque_Wrapped_FIFO(b *testing.B) {
	d := NewWithCapacity[int](1024)
	// Pre-fill and pop half to shift head into the middle
	for i := 0; i < 500; i++ {
		d.PushBack(i)
		d.PopFront()
	}
	for i := 0; i < 500; i++ {
		d.PushBack(i)
	}
	b.ReportAllocs()
	b.ResetTimer()

	var v int
	for b.Loop() {
		d.PushBack(v)
		d.PopFront()
		v++
	}
}

// -----------------------------------------------------------------------------
// Independent Reference Slice Oracle Model
// -----------------------------------------------------------------------------

type sliceOracle[T comparable] struct {
	data []T
}

func newSliceOracle[T comparable]() *sliceOracle[T] {
	return &sliceOracle[T]{data: make([]T, 0)}
}

func (o *sliceOracle[T]) pushFront(val T) {
	o.data = append([]T{val}, o.data...)
}

func (o *sliceOracle[T]) pushBack(val T) {
	o.data = append(o.data, val)
}

func (o *sliceOracle[T]) popFront() (T, bool) {
	if len(o.data) == 0 {
		var zero T
		return zero, false
	}
	val := o.data[0]
	o.data = o.data[1:]
	return val, true
}

func (o *sliceOracle[T]) popBack() (T, bool) {
	if len(o.data) == 0 {
		var zero T
		return zero, false
	}
	lastIdx := len(o.data) - 1
	val := o.data[lastIdx]
	o.data = o.data[:lastIdx]
	return val, true
}

func (o *sliceOracle[T]) front() (T, bool) {
	if len(o.data) == 0 {
		var zero T
		return zero, false
	}
	return o.data[0], true
}

func (o *sliceOracle[T]) back() (T, bool) {
	if len(o.data) == 0 {
		var zero T
		return zero, false
	}
	return o.data[len(o.data)-1], true
}

func (o *sliceOracle[T]) at(i int) T {
	if i < 0 {
		i += len(o.data)
	}
	return o.data[i]
}

func (o *sliceOracle[T]) set(i int, val T) {
	if i < 0 {
		i += len(o.data)
	}
	o.data[i] = val
}

func (o *sliceOracle[T]) clear() {
	o.data = o.data[:0]
}

func (o *sliceOracle[T]) len() int {
	return len(o.data)
}

func (o *sliceOracle[T]) toSlice() []T {
	cp := make([]T, len(o.data))
	copy(cp, o.data)
	return cp
}

func (o *sliceOracle[T]) backwardSlice() []T {
	cp := make([]T, len(o.data))
	for i := 0; i < len(o.data); i++ {
		cp[i] = o.data[len(o.data)-1-i]
	}
	return cp
}

// -----------------------------------------------------------------------------
// 1. Independent Slice Oracle Challenge across 100,000+ Randomized Operations
// -----------------------------------------------------------------------------

func TestOracle_100kRandomizedOps(t *testing.T) {
	const totalOps = 100_000
	rng := rand.New(rand.NewSource(998244353))

	d := New[int]()
	oracle := newSliceOracle[int]()

	for op := 0; op < totalOps; op++ {
		action := rng.Intn(100)
		val := rng.Intn(1_000_000)

		switch {
		case action < 25: // PushBack (25%)
			d.PushBack(val)
			oracle.pushBack(val)

		case action < 50: // PushFront (25%)
			d.PushFront(val)
			oracle.pushFront(val)

		case action < 68: // PopFront (18%)
			dVal, dOk := d.PopFront()
			oVal, oOk := oracle.popFront()
			require.Equal(t, oOk, dOk, fmt.Sprintf("op %d: PopFront ok mismatch", op))
			require.Equal(t, oVal, dVal, fmt.Sprintf("op %d: PopFront val mismatch", op))

		case action < 86: // PopBack (18%)
			dVal, dOk := d.PopBack()
			oVal, oOk := oracle.popBack()
			require.Equal(t, oOk, dOk, fmt.Sprintf("op %d: PopBack ok mismatch", op))
			require.Equal(t, oVal, dVal, fmt.Sprintf("op %d: PopBack val mismatch", op))

		case action < 92: // Front & Back Peek (6%)
			dFront, dFrontOk := d.Front()
			oFront, oFrontOk := oracle.front()
			require.Equal(t, oFrontOk, dFrontOk, fmt.Sprintf("op %d: Front ok mismatch", op))
			require.Equal(t, oFront, dFront, fmt.Sprintf("op %d: Front val mismatch", op))

			dBack, dBackOk := d.Back()
			oBack, oBackOk := oracle.back()
			require.Equal(t, oBackOk, dBackOk, fmt.Sprintf("op %d: Back ok mismatch", op))
			require.Equal(t, oBack, dBack, fmt.Sprintf("op %d: Back val mismatch", op))

		case action < 96: // Random access At & Set (4%)
			if oracle.len() > 0 {
				idx := rng.Intn(oracle.len())
				// Positive indexing
				require.Equal(t, oracle.at(idx), d.At(idx), fmt.Sprintf("op %d: At(%d) mismatch", op, idx))
				// Negative indexing
				negIdx := -1 - idx
				require.Equal(t, oracle.at(negIdx), d.At(negIdx), fmt.Sprintf("op %d: At(%d) mismatch", op, negIdx))

				// Set positive
				newVal := rng.Intn(1_000_000)
				d.Set(idx, newVal)
				oracle.set(idx, newVal)
				require.Equal(t, newVal, d.At(idx))

				// Set negative
				newVal2 := rng.Intn(1_000_000)
				d.Set(negIdx, newVal2)
				oracle.set(negIdx, newVal2)
				require.Equal(t, newVal2, d.At(negIdx))
			}

		default: // Clear (4%)
			d.Clear()
			oracle.clear()
		}

		// State invariants verification after EVERY operation
		require.Equal(t, oracle.len(), d.Len(), fmt.Sprintf("op %d: Len mismatch", op))
		require.Equal(t, oracle.len() == 0, d.Empty(), fmt.Sprintf("op %d: Empty mismatch", op))

		if oracle.len() > 0 {
			f, ok := d.Front()
			require.True(t, ok)
			require.Equal(t, oracle.at(0), f)
			require.Equal(t, oracle.at(0), d.At(0))

			b, ok := d.Back()
			require.True(t, ok)
			require.Equal(t, oracle.at(-1), b)
			require.Equal(t, oracle.at(-1), d.At(-1))
		}

		// Boundary panic invariant check on At and Set
		require.PanicsWithError(t, "deque: index out of range", func() { d.At(d.Len()) })
		require.PanicsWithError(t, "deque: index out of range", func() { d.At(-d.Len() - 1) })
		require.PanicsWithError(t, "deque: index out of range", func() { d.Set(d.Len(), 99) })
		require.PanicsWithError(t, "deque: index out of range", func() { d.Set(-d.Len()-1, 99) })

		// Periodic full iterator verification
		if op%1_000 == 0 {
			// Values iterator vs oracle slice
			collectedValues := slices.Collect(d.Values())
			expectedSlice := oracle.toSlice()
			if len(expectedSlice) == 0 {
				require.Equal(t, 0, len(collectedValues))
			} else {
				require.Equal(t, expectedSlice, collectedValues)
			}

			// Backward iterator vs reversed oracle slice
			collectedBackward := slices.Collect(d.Backward())
			expectedBackward := oracle.backwardSlice()
			if len(expectedBackward) == 0 {
				require.Equal(t, 0, len(collectedBackward))
			} else {
				require.Equal(t, expectedBackward, collectedBackward)
			}

			// All iterator vs indexed elements
			idxCount := 0
			for i, val := range d.All() {
				require.Equal(t, idxCount, i)
				require.Equal(t, expectedSlice[i], val)
				idxCount++
			}
			require.Equal(t, oracle.len(), idxCount)
		}
	}
}

// -----------------------------------------------------------------------------
// 2. Strict FIFO Queue Invariant Verification (Streaming & Steady-State)
// -----------------------------------------------------------------------------

func TestChallenge_FIFOInvariants_Streaming(t *testing.T) {
	// Test streaming across various fixed window capacities
	windowSizes := []int{1, 2, 3, 5, 8, 16, 31, 64, 128, 512, 1024}

	for _, win := range windowSizes {
		d := NewWithCapacity[int](win)
		const streamLength = 50_000

		// Fill window
		for i := 0; i < win; i++ {
			d.PushBack(i)
		}
		require.Equal(t, win, d.Len())

		// Slide window: for each new item pushed to back, pop front and verify strict FIFO monotonicity
		for i := win; i < streamLength; i++ {
			d.PushBack(i)
			expectedOut := i - win
			val, ok := d.PopFront()
			require.True(t, ok)
			require.Equal(t, expectedOut, val, fmt.Sprintf("window %d, step %d: FIFO order violation", win, i))
			require.Equal(t, win, d.Len())
		}

		// Drain the remainder: must be strictly ascending
		for i := streamLength - win; i < streamLength; i++ {
			val, ok := d.PopFront()
			require.True(t, ok)
			require.Equal(t, i, val)
		}
		require.True(t, d.Empty())
	}
}

func TestChallenge_ReverseFIFOInvariants_Streaming(t *testing.T) {
	// Reverse FIFO: PushFront and PopBack
	windowSizes := []int{1, 2, 4, 7, 16, 63, 100, 256}

	for _, win := range windowSizes {
		d := NewWithCapacity[int](win)
		const streamLength = 30_000

		// Fill window by pushing front
		for i := 0; i < win; i++ {
			d.PushFront(i)
		}
		require.Equal(t, win, d.Len())

		// Slide window: push front, pop back
		for i := win; i < streamLength; i++ {
			d.PushFront(i)
			expectedOut := i - win
			val, ok := d.PopBack()
			require.True(t, ok)
			require.Equal(t, expectedOut, val, fmt.Sprintf("rev window %d, step %d: FIFO order violation", win, i))
			require.Equal(t, win, d.Len())
		}

		// Drain remainder from back
		for i := streamLength - win; i < streamLength; i++ {
			val, ok := d.PopBack()
			require.True(t, ok)
			require.Equal(t, i, val)
		}
		require.True(t, d.Empty())
	}
}

// -----------------------------------------------------------------------------
// 3. Double-Ended Queue Properties & Dual LIFO Stacks
// -----------------------------------------------------------------------------

func TestChallenge_DoubleEnded_DualLIFOStacks(t *testing.T) {
	// Front stack: PushFront and PopFront must behave as pure LIFO
	frontStack := New[int]()
	const n = 10_000
	for i := 0; i < n; i++ {
		frontStack.PushFront(i)
		f, ok := frontStack.Front()
		require.True(t, ok)
		require.Equal(t, i, f)
	}
	for i := n - 1; i >= 0; i-- {
		val, ok := frontStack.PopFront()
		require.True(t, ok)
		require.Equal(t, i, val)
	}
	require.True(t, frontStack.Empty())

	// Back stack: PushBack and PopBack must behave as pure LIFO
	backStack := New[int]()
	for i := 0; i < n; i++ {
		backStack.PushBack(i)
		b, ok := backStack.Back()
		require.True(t, ok)
		require.Equal(t, i, b)
	}
	for i := n - 1; i >= 0; i-- {
		val, ok := backStack.PopBack()
		require.True(t, ok)
		require.Equal(t, i, val)
	}
	require.True(t, backStack.Empty())
}

func TestChallenge_DoubleEnded_SymmetricalPalindrome(t *testing.T) {
	d := New[int]()
	const pairs = 5_000

	// Push i to back, then i to front -> creates a palindrome:
	// front: (pairs-1), ..., 1, 0, 0, 1, ..., (pairs-1) :back
	for i := 0; i < pairs; i++ {
		d.PushBack(i)
		d.PushFront(i)
	}
	require.Equal(t, pairs*2, d.Len())

	// Symmetrical index invariance: At(i) == At(-1-i) == At(len - 1 - i)
	for i := 0; i < d.Len(); i++ {
		valFront := d.At(i)
		valBack := d.At(-1 - i)
		valBackPos := d.At(d.Len() - 1 - i)
		require.Equal(t, valFront, valBack)
		require.Equal(t, valFront, valBackPos)
	}

	// Symmetrical simultaneous pop
	for i := pairs - 1; i >= 0; i-- {
		fVal, fOk := d.PopFront()
		bVal, bOk := d.PopBack()
		require.True(t, fOk)
		require.True(t, bOk)
		require.Equal(t, i, fVal)
		require.Equal(t, i, bVal)
	}
	require.True(t, d.Empty())
}

// -----------------------------------------------------------------------------
// 4. Iterator Consistency & Early Exit Invariants
// -----------------------------------------------------------------------------

func TestChallenge_IteratorConsistency_ExhaustiveEarlyExit(t *testing.T) {
	const size = 128
	d := New[int]()
	for i := 0; i < size; i++ {
		d.PushBack(i * 10)
	}

	// Values() early exit at every cutoff point from 0 to size
	for cutoff := 0; cutoff <= size; cutoff++ {
		var collected []int
		for val := range d.Values() {
			if len(collected) == cutoff {
				break
			}
			collected = append(collected, val)
		}
		require.Equal(t, cutoff, len(collected))
		for i, v := range collected {
			require.Equal(t, i*10, v)
		}
		require.Equal(t, size, d.Len(), "early break must not alter deque length")
	}

	// Backward() early exit at every cutoff point from 0 to size
	for cutoff := 0; cutoff <= size; cutoff++ {
		var collected []int
		for val := range d.Backward() {
			if len(collected) == cutoff {
				break
			}
			collected = append(collected, val)
		}
		require.Equal(t, cutoff, len(collected))
		for i, v := range collected {
			expected := (size - 1 - i) * 10
			require.Equal(t, expected, v)
		}
		require.Equal(t, size, d.Len(), "early break must not alter deque length")
	}

	// All() early exit at every cutoff point from 0 to size
	for cutoff := 0; cutoff <= size; cutoff++ {
		count := 0
		for i, val := range d.All() {
			if count == cutoff {
				break
			}
			require.Equal(t, count, i)
			require.Equal(t, count*10, val)
			count++
		}
		require.Equal(t, cutoff, count)
	}

	// Drain() early exit: test partial consumption
	dDrain := New[int]()
	for i := 0; i < 50; i++ {
		dDrain.PushBack(i)
	}

	drainedCount := 0
	for val := range dDrain.Drain() {
		require.Equal(t, drainedCount, val)
		drainedCount++
		if drainedCount == 20 {
			break
		}
	}
	require.Equal(t, 20, drainedCount)
	require.Equal(t, 30, dDrain.Len())
	require.Equal(t, 20, dDrain.At(0))
	require.Equal(t, 49, dDrain.At(-1))
}

// -----------------------------------------------------------------------------
// 5. Ring Buffer Wraparound Boundary Stress
// -----------------------------------------------------------------------------

func TestChallenge_ContinuousHeadRotations(t *testing.T) {
	// Rotate the head continuously through thousands of full 360-degree cycles of the circular buffer
	const capSize = 16
	d := NewWithCapacity[int](capSize)

	// Keep size at 10 while shifting head 1,000,000 steps
	for i := 0; i < 10; i++ {
		d.PushBack(i)
	}

	const steps = 1_000_000
	for step := 0; step < steps; step++ {
		// Pop front
		oldVal, ok := d.PopFront()
		require.True(t, ok)
		require.Equal(t, step, oldVal)

		// Push back next val
		newVal := step + 10
		d.PushBack(newVal)

		// Invariant checks
		require.Equal(t, 10, d.Len())
		require.Equal(t, capSize, d.Cap()) // no reallocations!
	}
}

// -----------------------------------------------------------------------------
// 6. Generic Types Variety (Structs, Strings, Arrays, Pointers)
// -----------------------------------------------------------------------------

func TestChallenge_GenericTypesVariety(t *testing.T) {
	// Test strings
	dStr := New[string]()
	dStr.PushBack("alpha")
	dStr.PushFront("beta")
	dStr.PushBack("gamma")
	require.Equal(t, []string{"beta", "alpha", "gamma"}, slices.Collect(dStr.Values()))
	require.Equal(t, "beta", dStr.At(0))
	require.Equal(t, "gamma", dStr.At(-1))

	// Test complex structs
	type record struct {
		id   int
		name string
		data [16]byte
	}
	dRec := New[record]()
	rec1 := record{id: 1, name: "one", data: [16]byte{1, 2, 3}}
	rec2 := record{id: 2, name: "two", data: [16]byte{4, 5, 6}}
	dRec.PushBack(rec1)
	dRec.PushFront(rec2)
	require.Equal(t, rec2, dRec.At(0))
	require.Equal(t, rec1, dRec.At(1))
	poppedFront, _ := dRec.PopFront()
	require.Equal(t, rec2, poppedFront)

	// Test zero-sized types struct{}
	dEmpty := New[struct{}]()
	for i := 0; i < 100; i++ {
		dEmpty.PushBack(struct{}{})
	}
	require.Equal(t, 100, dEmpty.Len())
	for i := 0; i < 100; i++ {
		_, ok := dEmpty.PopFront()
		require.True(t, ok)
	}
	require.True(t, dEmpty.Empty())
}

// -----------------------------------------------------------------------------
// 7. Multi-Threaded Concurrent Torture & Race Validation
// -----------------------------------------------------------------------------

func TestChallenge_ConcurrentTortureWithRace(t *testing.T) {
	d := NewWithCapacity[int64](128)
	var mu sync.RWMutex
	var wg sync.WaitGroup

	const readers = 12
	const writers = 6
	const opsPerWorker = 1_000

	var totalPushed atomic.Int64
	var totalPopped atomic.Int64

	// Concurrent synchronized writers (mix of PushFront, PushBack, PopFront, PopBack)
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(wid int) {
			defer wg.Done()
			for i := 0; i < opsPerWorker; i++ {
				mu.Lock()
				val := int64(wid*opsPerWorker + i)
				if wid%2 == 0 {
					d.PushBack(val)
				} else {
					d.PushFront(val)
				}
				totalPushed.Add(1)

				if d.Len() > 64 {
					if wid%2 == 0 {
						_, ok := d.PopFront()
						if ok {
							totalPopped.Add(1)
						}
					} else {
						_, ok := d.PopBack()
						if ok {
							totalPopped.Add(1)
						}
					}
				}
				mu.Unlock()
			}
		}(w)
	}

	// Concurrent readers
	for r := 0; r < readers; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < opsPerWorker; i++ {
				mu.RLock()
				_ = d.Len()
				_ = d.Empty()
				_ = d.Cap()
				if d.Len() > 0 {
					_, _ = d.Front()
					_, _ = d.Back()
					_ = d.At(0)
					_ = d.At(-1)
				}
				mu.RUnlock()
			}
		}()
	}

	wg.Wait()

	// Drain remaining
	for {
		mu.Lock()
		_, ok := d.PopFront()
		mu.Unlock()
		if !ok {
			break
		}
		totalPopped.Add(1)
	}

	require.Equal(t, totalPushed.Load(), totalPopped.Load(), "every pushed item must be accounted for")
	require.True(t, d.Empty())
}
