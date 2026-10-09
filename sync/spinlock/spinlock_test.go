// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package spinlock

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

)

func TestLock_Basic(t *testing.T) {
	var mu Lock

	// Initial lock
	mu.Lock()
	// TryLock should fail when locked
	if mu.TryLock() {
		t.Fatal("TryLock should fail when already locked")
	}

	mu.Unlock()

	// TryLock should succeed when unlocked
	if !mu.TryLock() {
		t.Fatal("TryLock should succeed when unlocked")
	}

	// TryLock should fail again
	if mu.TryLock() {
		t.Fatal("TryLock should fail when already locked")
	}

	mu.Unlock()
}

func TestLock_Concurrent(t *testing.T) {
	var (
		mu Lock
		wg sync.WaitGroup
	)

	counter := 0
	numGoroutines := 100
	numIterations := 1000

	wg.Add(numGoroutines)

	for range numGoroutines {
		go func() {
			defer wg.Done()

			for range numIterations {
				mu.Lock()
				counter++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	expected := numGoroutines * numIterations
	if counter != expected {
		t.Errorf("expected counter to be %d, got %d", expected, counter)
	}
}

func TestLock_TryLockConcurrent(t *testing.T) {
	var (
		mu Lock
		wg sync.WaitGroup
	)

	counter := 0
	numGoroutines := 10
	numIterations := 100

	wg.Add(numGoroutines)

	for range numGoroutines {
		go func() {
			defer wg.Done()

			for j := 0; j < numIterations; {
				if mu.TryLock() {
					counter++
					mu.Unlock()

					j++
				} else {
					time.Sleep(time.Microsecond)
				}
			}
		}()
	}

	wg.Wait()

	expected := numGoroutines * numIterations
	if counter != expected {
		t.Errorf("expected counter to be %d, got %d", expected, counter)
	}
}

func TestLock_UnlockPanic(t *testing.T) {
	var mu Lock

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected unlock of unlocked spinlock to panic")
		}
	}()

	mu.Unlock()
}

func BenchmarkLock_Sequential(b *testing.B) {
	var mu Lock
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		mu.Lock()
		mu.Unlock()
	}
}

func BenchmarkMutex_Sequential(b *testing.B) {
	var mu sync.Mutex
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		mu.Lock()
		mu.Unlock()
	}
}

func BenchmarkLock_Parallel(b *testing.B) {
	var mu Lock
	counter := 0
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.Lock()
			counter++
			mu.Unlock()
		}
	})
}

func BenchmarkMutex_Parallel(b *testing.B) {
	var mu sync.Mutex
	counter := 0
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.Lock()
			counter++
			mu.Unlock()
		}
	})
}

// Merged from doc_test.go

// TestSpinlock_HeavyContention_MutualExclusion verifies that Lock guarantees strict
// mutual exclusion under extreme multi-core contention across hundreds of goroutines.
func TestSpinlock_HeavyContention_MutualExclusion(t *testing.T) {
	var mu Lock
	const (
		numGoroutines = 128
		iterations    = 1000
	)

	counter := 0
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	startGate := make(chan struct{})

	for range numGoroutines {
		go func() {
			defer wg.Done()
			<-startGate

			for range iterations {
				mu.Lock()
				counter++
				mu.Unlock()
			}
		}()
	}

	close(startGate)
	wg.Wait()

	expected := numGoroutines * iterations
	if counter != expected {
		t.Fatalf("spinlock failed mutual exclusion: expected %d, got %d", expected, counter)
	}
}

// TestSpinlock_TryLock_Contention verifies that TryLock respects mutual exclusion
// and never succeeds when another goroutine holds the lock.
func TestSpinlock_TryLock_Contention(t *testing.T) {
	var mu Lock
	const numGoroutines = 32

	var (
		wg         sync.WaitGroup
		acquired   atomic.Int64
		conflicts  atomic.Int64
		inCritical atomic.Int32
		stop       atomic.Bool
	)

	wg.Add(numGoroutines)
	for range numGoroutines {
		go func() {
			defer wg.Done()

			for !stop.Load() {
				if mu.TryLock() {
					// Verify no one else is in critical section
					current := inCritical.Add(1)
					if current != 1 {
						t.Errorf("TryLock race detected! Multiple holders: %d", current)
					}

					acquired.Add(1)
					runtime.Gosched()

					inCritical.Add(-1)
					mu.Unlock()
				} else {
					conflicts.Add(1)
					runtime.Gosched()
				}
			}
		}()
	}

	// Run for a bit to generate heavy TryLock conflicts
	for acquired.Load() < 5000 {
		runtime.Gosched()
	}
	stop.Store(true)
	wg.Wait()

	if acquired.Load() == 0 {
		t.Fatal("no TryLock succeeded")
	}
	if conflicts.Load() == 0 {
		t.Fatal("expected TryLock conflicts under concurrent stress")
	}
}

// TestSpinlock_MixedLockAndTryLock verifies consistency when goroutines interleave
// blocking Lock and opportunistic TryLock.
func TestSpinlock_MixedLockAndTryLock(t *testing.T) {
	var mu Lock
	var inCritical atomic.Int32
	var wg sync.WaitGroup

	const numWorkers = 64
	const iterations = 500

	wg.Add(numWorkers)
	for i := range numWorkers {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < iterations; j++ {
				if (id+j)%2 == 0 {
					mu.Lock()
				} else {
					for !mu.TryLock() {
						runtime.Gosched()
					}
				}

				// Inside critical section
				cur := inCritical.Add(1)
				if cur != 1 {
					t.Errorf("mutual exclusion violated in mixed test: %d concurrent holders", cur)
				}
				runtime.Gosched()
				inCritical.Add(-1)

				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()
}

