// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ringbuf_test

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/silicon/ringbuf"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

func TestSPSCRingBuffer_BasicOperations(t *testing.T) {
	t.Parallel()

	buf := ringbuf.NewSPSCRingBuffer[int](4)
	assert.Equal(t, 4, buf.Cap())
	assert.Equal(t, 0, buf.Len())
	assert.True(t, buf.IsEmpty())
	assert.False(t, buf.IsFull())

	v1, v2, v3, v4, v5 := 10, 20, 30, 40, 50

	require.True(t, buf.Push(&v1))
	require.True(t, buf.Push(&v2))
	require.True(t, buf.Push(&v3))
	require.True(t, buf.Push(&v4))
	assert.True(t, buf.IsFull())

	// Push when full should fail
	require.False(t, buf.Push(&v5))

	// Pop elements in FIFO order
	p1 := buf.Pop()
	require.NotNil(t, p1)
	assert.Equal(t, 10, *p1)

	p2 := buf.Pop()
	require.NotNil(t, p2)
	assert.Equal(t, 20, *p2)

	assert.Equal(t, 2, buf.Len())

	// Push after pop should succeed
	require.True(t, buf.Push(&v5))

	p3 := buf.Pop()
	require.NotNil(t, p3)
	assert.Equal(t, 30, *p3)

	p4 := buf.Pop()
	require.NotNil(t, p4)
	assert.Equal(t, 40, *p4)

	p5 := buf.Pop()
	require.NotNil(t, p5)
	assert.Equal(t, 50, *p5)

	assert.True(t, buf.IsEmpty())
	assert.Nil(t, buf.Pop())
}

func TestSPSCRingBuffer_Drain(t *testing.T) {
	t.Parallel()

	buf := ringbuf.NewSPSCRingBuffer[int](8)
	// Empty drain
	for range buf.Drain() {
		t.Fatal("expected empty drain")
	}

	for i := 1; i <= 4; i++ {
		val := i * 10
		require.True(t, buf.Push(&val))
	}

	var drained []int
	for ptr := range buf.Drain() {
		drained = append(drained, *ptr)
		if len(drained) == 2 {
			break
		}
	}
	assert.Equal(t, []int{10, 20}, drained)

	drained = nil
	for ptr := range buf.Drain() {
		drained = append(drained, *ptr)
	}
	assert.Equal(t, []int{30, 40}, drained)
	assert.True(t, buf.IsEmpty())
}

func TestSPSCRingBuffer_ConcurrentProducerConsumer(t *testing.T) {
	t.Parallel()

	const itemCount = 100_000

	buf := ringbuf.NewSPSCRingBuffer[int](1024)

	var wg sync.WaitGroup

	wg.Add(2)

	// Single Producer Goroutine
	go func() {
		defer wg.Done()

		for i := range itemCount {
			val := i

			for !buf.Push(&val) {
				// Spin until space is available
			}
		}
	}()

	received := make([]int, 0, itemCount)

	// Single Consumer Goroutine
	go func() {
		defer wg.Done()

		for len(received) < itemCount {
			item := buf.Pop()
			if item != nil {
				received = append(received, *item)
			}
		}
	}()

	wg.Wait()

	require.Len(t, received, itemCount)

	for i := range itemCount {
		assert.Equal(t, i, received[i])
	}
}

func TestSPSCRingBuffer_LocalCacheCoherency(t *testing.T) {
	t.Parallel()

	buf := ringbuf.NewSPSCRingBuffer[int](8)

	// Step 1: Fill the buffer completely (8 items).
	// Under local caching, producer never loads consumer's tail until buffer is full.
	for i := 1; i <= 8; i++ {
		val := i
		require.True(t, buf.Push(&val))
	}

	// Step 2: 9th push must fail (full buffer)
	overflow := 99
	require.False(t, buf.Push(&overflow))

	// Step 3: Consumer pops 4 items.
	for i := 1; i <= 4; i++ {
		p := buf.Pop()
		require.NotNil(t, p)
		assert.Equal(t, i, *p)
	}

	// Step 4: Producer can now push 4 more items (9..12) as it re-checks tail on full detection.
	for i := 9; i <= 12; i++ {
		val := i
		require.True(t, buf.Push(&val))
	}

	// Step 5: Producer should now see buffer full again.
	require.False(t, buf.Push(&overflow))

	// Step 6: Consumer pops remaining 8 items (5..12).
	for i := 5; i <= 12; i++ {
		p := buf.Pop()
		require.NotNil(t, p)
		assert.Equal(t, i, *p)
	}

	// Step 7: Queue must now be empty.
	assert.True(t, buf.IsEmpty())
	assert.Nil(t, buf.Pop())
}

func BenchmarkSPSCRingBuffer_PushPop(b *testing.B) {
	buf := ringbuf.NewSPSCRingBuffer[int](1024)
	val := 42

	b.ReportAllocs()

	for b.Loop() {
		buf.Push(&val)
		_ = buf.Pop()
	}
}

func BenchmarkSPSCRingBuffer_Concurrent(b *testing.B) {
	buf := ringbuf.NewSPSCRingBuffer[int](1024)
	val := 42

	var active atomic.Bool
	active.Store(true)
	var wg sync.WaitGroup

	wg.Go(func() {
		for active.Load() {
			buf.Push(&val)
		}
	})

	b.ReportAllocs()

	for b.Loop() {
		for buf.Pop() == nil {
		}
	}

	b.StopTimer()
	active.Store(false)
	wg.Wait()
}

func BenchmarkSPSCRingBuffer_Concurrent_Cap64(b *testing.B) {
	buf := ringbuf.NewSPSCRingBuffer[int](64)
	val := 42

	var active atomic.Bool
	active.Store(true)
	var wg sync.WaitGroup

	wg.Go(func() {
		for active.Load() {
			buf.Push(&val)
		}
	})

	b.ReportAllocs()

	for b.Loop() {
		for buf.Pop() == nil {
		}
	}

	b.StopTimer()
	active.Store(false)
	wg.Wait()
}

func BenchmarkSPSCRingBuffer_Concurrent_Cap65536(b *testing.B) {
	buf := ringbuf.NewSPSCRingBuffer[int](65536)
	val := 42

	var active atomic.Bool
	active.Store(true)
	var wg sync.WaitGroup

	wg.Go(func() {
		for active.Load() {
			buf.Push(&val)
		}
	})

	b.ReportAllocs()

	for b.Loop() {
		for buf.Pop() == nil {
		}
	}

	b.StopTimer()
	active.Store(false)
	wg.Wait()
}

func TestSPSCRingBuffer_Clear(t *testing.T) {
	buf := ringbuf.NewSPSCRingBuffer[int](8)
	vals := []int{10, 20, 30}
	for i := range vals {
		require.True(t, buf.Push(&vals[i]))
	}
	require.Equal(t, 3, buf.Len())

	buf.Clear()
	require.Equal(t, 0, buf.Len())
	require.True(t, buf.IsEmpty())
	require.Nil(t, buf.Pop())
}

func TestSPSCRingBuffer_GCLeak(t *testing.T) {
	type trackedObj struct {
		payload [64]byte
		val     int
	}

	buf := ringbuf.NewSPSCRingBuffer[trackedObj](8)
	var finalized atomic.Int32

	obj := &trackedObj{val: 99}
	runtime.AddCleanup(obj, func(counter *atomic.Int32) {
		counter.Add(1)
	}, &finalized)

	require.True(t, buf.Push(obj))
	obj = nil

	popped := buf.Pop()
	require.NotNil(t, popped)
	require.Equal(t, 99, popped.val)
	popped = nil

	for range 5 {
		runtime.GC()
		if finalized.Load() > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	require.Equal(t, int32(1), finalized.Load(), "object reference was retained in SPSCRingBuffer slot")
}

func TestSPSCRingBuffer_WrapAround(t *testing.T) {
	buf := ringbuf.NewSPSCRingBuffer[int](4)
	val := 123
	// Cycle through multiple buffer wrap boundaries
	for range 100 {
		require.True(t, buf.Push(&val))
		p := buf.Pop()
		require.NotNil(t, p)
		require.Equal(t, 123, *p)
	}
	require.Equal(t, 0, buf.Len())
	require.True(t, buf.IsEmpty())
}

func TestSPSCRingBuffer_ZeroAlloc(t *testing.T) {
	buf := ringbuf.NewSPSCRingBuffer[int](64)
	val := 42

	allocs := testing.AllocsPerRun(100, func() {
		buf.Push(&val)
		_ = buf.Pop()
	})
	require.Equal(t, float64(0), allocs)
}
