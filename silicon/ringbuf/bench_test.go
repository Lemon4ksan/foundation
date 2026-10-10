package ringbuf

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/testing/require"
)

// TestChallenge_GCLeakVerification proves that popped elements in Buffer[T]
// are swapped to nil immediately, preventing memory leaks of unreferenced objects.
func TestChallenge_GCLeakVerification(t *testing.T) {
	type trackedObj struct {
		payload [64]byte
		id      int
	}

	var finalized atomic.Int32

	buf := New[trackedObj](4)
	func() {
		obj := &trackedObj{id: 42}

		runtime.AddCleanup(obj, func(counter *atomic.Int32) {
			counter.Add(1)
		}, &finalized)

		require.True(t, buf.Push(obj))

		popped := buf.Pop()
		require.NotNil(t, popped)
		require.Equal(t, 42, popped.id)
	}()

	// Force multiple GC cycles
	for range 5 {
		runtime.GC()
		if finalized.Load() > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	require.Equal(t, int32(1), finalized.Load(), "object reference was retained in silicon/ringbuf cell")
}

// TestChallenge_MPMCHighContentionTorture launches multiple concurrent producers and consumers
// under high contention, verifying exact conservation of elements without duplicates or lost data.
func TestChallenge_MPMCHighContentionTorture(t *testing.T) {
	const (
		numProducers = 8
		numConsumers = 8
		itemsPerProd = 10_000
		totalItems   = numProducers * itemsPerProd
		bufCap       = 128
	)

	buf := New[int](bufCap)

	received := make([]atomic.Int32, totalItems)
	var totalConsumed atomic.Int64

	var wg sync.WaitGroup
	wg.Add(numProducers + numConsumers)

	// Producers
	for p := range numProducers {
		start := p * itemsPerProd
		go func(base int) {
			defer wg.Done()
			for i := range itemsPerProd {
				val := base + i
				for !buf.Push(&val) {
					runtime.Gosched()
				}
			}
		}(start)
	}

	// Consumers
	for range numConsumers {
		go func() {
			defer wg.Done()
			for totalConsumed.Load() < int64(totalItems) {
				item := buf.Pop()
				if item == nil {
					runtime.Gosched()
					continue
				}
				val := *item
				require.True(t, val >= 0 && val < totalItems, "item out of bounds")
				received[val].Add(1)
				totalConsumed.Add(1)
			}
		}()
	}

	wg.Wait()

	require.Equal(t, int64(totalItems), totalConsumed.Load())
	for i := range totalItems {
		require.Equal(t, int32(1), received[i].Load(), "item was either dropped or duplicated")
	}
}

// TestChallenge_CapacityEdgeCases validates boundary conditions on capacity sizing and wrapping.
func TestChallenge_CapacityEdgeCases(t *testing.T) {
	testCaps := []struct {
		input       int
		expectedCap uint64
	}{
		{-10, 2},
		{0, 2},
		{1, 2},
		{2, 2},
		{3, 4},
		{4, 4},
		{5, 8},
		{31, 32},
		{32, 32},
		{33, 64},
	}

	for _, tc := range testCaps {
		b := New[int](tc.input)
		require.Equal(t, tc.expectedCap, b.capacity)
		require.Equal(t, tc.expectedCap-1, b.mask)

		// Fill to capacity
		items := make([]int, tc.expectedCap)
		for i := uint64(0); i < tc.expectedCap; i++ {
			items[i] = int(i)
			require.True(t, b.Push(&items[i]))
		}

		// Next push must fail (full)
		overflowVal := 999
		require.False(t, b.Push(&overflowVal))

		// Pop all
		for i := uint64(0); i < tc.expectedCap; i++ {
			popped := b.Pop()
			require.NotNil(t, popped)
			require.Equal(t, int(i), *popped)
		}

		// Next pop must return nil (empty)
		require.Nil(t, b.Pop())
	}
}

// TestChallenge_DrainEarlyExit verifies that the Drain iterator supports early termination.
func TestChallenge_DrainEarlyExit(t *testing.T) {
	buf := New[int](8)
	vals := []int{10, 20, 30, 40}
	for i := range vals {
		buf.Push(&vals[i])
	}

	var collected []int
	for item := range buf.Drain() {
		collected = append(collected, *item)
		if len(collected) == 2 {
			break
		}
	}

	require.Equal(t, []int{10, 20}, collected)
	// Remaining elements should still be in the buffer
	p3 := buf.Pop()
	require.NotNil(t, p3)
	require.Equal(t, 30, *p3)
	p4 := buf.Pop()
	require.NotNil(t, p4)
	require.Equal(t, 40, *p4)
	require.Nil(t, buf.Pop())
}

func TestRepro_New_NegativeCapacityHangs(t *testing.T) {
	done := make(chan struct{})

	go func() {
		// Passing negative capacity should not hang in an infinite loop
		rb := New[int](-1)
		if rb.capacity < 2 {
			t.Errorf("expected capacity >= 2, got %d", rb.capacity)
		}
		close(done)
	}()

	select {
	case <-done:
		// Succeeded without hanging
	case <-time.After(1 * time.Second):
		t.Fatal("CRITICAL: New(-1) hung in infinite bitshift loop!")
	}
}
