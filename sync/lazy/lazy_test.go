// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lazy

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
)

func TestValue_Basic(t *testing.T) {
	var calls int32

	l := New(func() (int, error) {
		atomic.AddInt32(&calls, 1)
		return 42, nil
	})

	// First call
	val, err := l.Get()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val != 42 {
		t.Fatalf("expected 42, got %d", val)
	}

	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}

	// Second call
	val, err = l.Get()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val != 42 {
		t.Fatalf("expected 42, got %d", val)
	}

	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestValue_Error(t *testing.T) {
	expectedErr := errors.New("init error")
	l := New(func() (string, error) {
		return "", expectedErr
	})

	val, err := l.Get()
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}

	if val != "" {
		t.Fatalf("expected empty string, got %q", val)
	}
}

func TestValue_Reset(t *testing.T) {
	var calls atomic.Int32

	l := New(func() (int, error) {
		c := calls.Add(1)
		return int(c), nil
	})

	val, err := l.Get()
	if err != nil || val != 1 {
		t.Fatalf("expected (1, nil), got (%d, %v)", val, err)
	}

	l.Reset()

	val, err = l.Get()
	if err != nil || val != 2 {
		t.Fatalf("expected (2, nil), got (%d, %v)", val, err)
	}
}

func TestValue_Concurrent(t *testing.T) {
	var calls int32

	l := New(func() (int, error) {
		atomic.AddInt32(&calls, 1)
		return 100, nil
	})

	var wg sync.WaitGroup

	numGoroutines := 50
	wg.Add(numGoroutines)

	for range numGoroutines {
		go func() {
			defer wg.Done()

			val, err := l.Get()
			if err != nil || val != 100 {
				t.Errorf("expected (100, nil), got (%d, %v)", val, err)
			}
		}()
	}

	wg.Wait()

	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("expected init to be called exactly once, got %d", calls)
	}
}

func TestValue_ConcurrentWithReset(t *testing.T) {
	var calls atomic.Int32
	l := New(func() (int, error) {
		return int(calls.Add(1)), nil
	})

	stop := make(chan struct{})
	var wg sync.WaitGroup

	// Readers
	for range 16 {
		wg.Go(func() {
			for {
				select {
				case <-stop:
					return
				default:
					val, err := l.Get()
					if err != nil || val <= 0 {
						t.Errorf("unexpected value: %v, %v", val, err)
					}
				}
			}
		})
	}

	// Resetters
	for range 2 {
		wg.Go(func() {
			for {
				select {
				case <-stop:
					return
				default:
					l.Reset()
				}
			}
		})
	}

	// Run for a short burst
	for range 50 {
		l.Get()
	}
	close(stop)
	wg.Wait()
}

func BenchmarkValue_Get_Initialized(b *testing.B) {
	l := New(func() (int, error) {
		return 42, nil
	})
	if _, err := l.Get(); err != nil {
		b.Fatalf("unexpected error: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		val, _ := l.Get()
		if val != 42 {
			b.Fatalf("unexpected val: %d", val)
		}
	}
}

func BenchmarkValue_Get_Parallel(b *testing.B) {
	l := New(func() (int, error) {
		return 42, nil
	})
	if _, err := l.Get(); err != nil {
		b.Fatalf("unexpected error: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			val, _ := l.Get()
			if val != 42 {
				b.Fatalf("unexpected val: %d", val)
			}
		}
	})
}

type stressPayload struct {
	id       int64
	data     [256]byte
	checksum [32]byte
}

func makePayload(id int64) stressPayload {
	var p stressPayload
	p.id = id
	for i := range p.data {
		p.data[i] = byte(id + int64(i))
	}
	h := sha256.New()
	var idBytes [8]byte
	binary.LittleEndian.PutUint64(idBytes[:], uint64(id))
	h.Write(idBytes[:])
	h.Write(p.data[:])
	copy(p.checksum[:], h.Sum(nil))
	return p
}

func verifyPayload(p stressPayload) bool {
	h := sha256.New()
	var idBytes [8]byte
	binary.LittleEndian.PutUint64(idBytes[:], uint64(p.id))
	h.Write(idBytes[:])
	h.Write(p.data[:])
	expected := h.Sum(nil)
	for i := range 32 {
		if p.checksum[i] != expected[i] {
			return false
		}
	}
	return true
}

// TestValue_HighContentionRace verifies that concurrent Get and Reset cycles
// under extreme contention never exhibit word tearing, partial struct publication,
// or data races.
func TestValue_HighContentionRace(t *testing.T) {
	var seq atomic.Int64

	l := New(func() (stressPayload, error) {
		id := seq.Add(1)
		return makePayload(id), nil
	})

	const (
		numReaders   = 64
		numResetters = 16
		durationOps  = 500
	)

	stop := make(chan struct{})
	var wg sync.WaitGroup

	// Readers
	for r := range numReaders {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					p, err := l.Get()
					if err != nil {
						t.Errorf("reader %d: unexpected error: %v", readerID, err)
						return
					}
					if !verifyPayload(p) {
						t.Errorf("reader %d: corrupted payload observed! word tearing detected: id=%d", readerID, p.id)
						return
					}
				}
			}
		}(r)
	}

	// Resetters
	for range numResetters {
		wg.Go(func() {
			for {
				select {
				case <-stop:
					return
				default:
					l.Reset()
				}
			}
		})
	}

	// Main controller drives progress
	for range durationOps {
		p, err := l.Get()
		if err != nil {
			t.Fatalf("main: unexpected error: %v", err)
		}
		if !verifyPayload(p) {
			t.Fatalf("main: corrupted payload observed! id=%d", p.id)
		}
	}

	close(stop)
	wg.Wait()
}

// TestValue_ErrorCachingAndReset verifies error caching semantics under concurrent access.
func TestValue_ErrorCachingAndReset(t *testing.T) {
	expectedErr := errors.New("transient failure")
	var shouldFail atomic.Bool
	shouldFail.Store(true)
	var callCount atomic.Int32

	l := New(func() (string, error) {
		callCount.Add(1)
		if shouldFail.Load() {
			return "", expectedErr
		}
		return "success", nil
	})

	// Concurrent reads while failing
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			val, err := l.Get()
			if !errors.Is(err, expectedErr) {
				t.Errorf("expected %v, got %v", expectedErr, err)
			}
			if val != "" {
				t.Errorf("expected empty string, got %q", val)
			}
		})
	}
	wg.Wait()

	if callCount.Load() != 1 {
		t.Fatalf("expected init to be called once while cached, got %d", callCount.Load())
	}

	// Reset and transition to success
	shouldFail.Store(false)
	l.Reset()

	for range 20 {
		wg.Go(func() {
			val, err := l.Get()
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if val != "success" {
				t.Errorf("expected 'success', got %q", val)
			}
		})
	}
	wg.Wait()

	if callCount.Load() != 2 {
		t.Fatalf("expected init to be called exactly twice across resets, got %d", callCount.Load())
	}
}

// TestValue_PanicSafety verifies that if init panics, the lock is properly
// released and does not deadlock future callers.
func TestValue_PanicSafety(t *testing.T) {
	var shouldPanic atomic.Bool
	shouldPanic.Store(true)

	l := New(func() (int, error) {
		if shouldPanic.Load() {
			panic("init panicked")
		}
		return 42, nil
	})

	// First call panics
	func() {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic, got nil")
			}
		}()
		_, _ = l.Get()
	}()

	// Second call with shouldPanic = false should acquire lock and succeed without deadlocking
	shouldPanic.Store(false)
	val, err := l.Get()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 42 {
		t.Fatalf("expected 42, got %d", val)
	}
}

// TestValue_FastPathZeroAlloc verifies that initialized Get produces 0 allocations.
func TestValue_FastPathZeroAlloc(t *testing.T) {
	l := New(func() (int, error) {
		return 12345, nil
	})

	if _, err := l.Get(); err != nil {
		t.Fatalf("failed initial Get: %v", err)
	}

	allocs := testing.AllocsPerRun(1000, func() {
		val, err := l.Get()
		if err != nil || val != 12345 {
			t.Fatalf("unexpected: val=%d err=%v", val, err)
		}
	})

	if allocs != 0 {
		t.Fatalf("expected 0 allocs on initialized Get(), got %f", allocs)
	}
}

type complexData struct {
	id     int
	values []string
	lookup map[string]int
}

// TestLazy_MassiveFirstGetRace verifies that under 256 concurrent goroutines racing on uninitialized Value,
// the initializer function runs strictly once, and all readers observe the completely published data structure.
func TestLazy_MassiveFirstGetRace(t *testing.T) {
	var initCount atomic.Int32
	gate := make(chan struct{})

	v := New(func() (*complexData, error) {
		initCount.Add(1)
		// Small yield to maximize contention during initialization
		runtime.Gosched()
		return &complexData{
			id:     42,
			values: []string{"alpha", "beta", "gamma"},
			lookup: map[string]int{"alpha": 1, "beta": 2, "gamma": 3},
		}, nil
	})

	const numGoroutines = 256
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()
			<-gate

			res, err := v.Get()
			if err != nil {
				t.Errorf("goroutine %d: unexpected error: %v", id, err)
				return
			}
			if res == nil || res.id != 42 || len(res.values) != 3 || res.lookup["beta"] != 2 {
				t.Errorf("goroutine %d: observed torn/uninitialized struct: %+v", id, res)
			}
		}(i)
	}

	close(gate)
	wg.Wait()

	if initCount.Load() != 1 {
		t.Fatalf("expected init to run exactly once, ran %d times", initCount.Load())
	}
}

// TestLazy_ConcurrentGetAndResetStorm stresses concurrent Get and Reset calls under race detector.
func TestLazy_ConcurrentGetAndResetStorm(t *testing.T) {
	var generation atomic.Int64

	v := New(func() (int64, error) {
		gen := generation.Add(1)
		return gen, nil
	})

	const (
		numReaders   = 64
		numResetters = 16
		iterations   = 500
	)

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Readers
	wg.Add(numReaders)
	for i := range numReaders {
		go func(id int) {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					val, err := v.Get()
					if err != nil {
						t.Errorf("reader %d: unexpected error: %v", id, err)
						return
					}
					if val <= 0 {
						t.Errorf("reader %d: invalid value observed: %d", id, val)
						return
					}
				}
			}
		}(i)
	}

	// Resetters
	wg.Add(numResetters)
	for range numResetters {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					v.Reset()
					runtime.Gosched()
				}
			}
		}()
	}

	// Main driver
	for range iterations {
		_, err := v.Get()
		if err != nil {
			t.Fatalf("driver error: %v", err)
		}
	}

	close(stop)
	wg.Wait()
}

// TestLazy_ErrorPropagationAndRecovery verifies that transient errors are cached until Reset,
// and recovery works as expected.
func TestLazy_ErrorPropagationAndRecovery(t *testing.T) {
	var shouldFail atomic.Bool
	shouldFail.Store(true)

	errTransient := errors.New("temporary outage")

	v := New(func() (string, error) {
		if shouldFail.Load() {
			return "", errTransient
		}
		return "recovered", nil
	})

	// Concurrently test error caching
	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			val, err := v.Get()
			if !errors.Is(err, errTransient) {
				t.Errorf("goroutine %d: expected errTransient, got %v", id, err)
			}
			if val != "" {
				t.Errorf("goroutine %d: expected empty string on error, got %q", id, val)
			}
		}(i)
	}
	wg.Wait()

	// Now recover and reset
	shouldFail.Store(false)
	v.Reset()

	for i := range 50 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			val, err := v.Get()
			if err != nil {
				t.Errorf("goroutine %d: unexpected error after reset: %v", id, err)
			}
			if val != "recovered" {
				t.Errorf("goroutine %d: expected 'recovered', got %q", id, val)
			}
		}(i)
	}
	wg.Wait()
}

// TestLazy_PanicRecoveryNonDeadlock ensures that panics inside init() do not permanently
// deadlock the mutex.
func TestLazy_PanicRecoveryNonDeadlock(t *testing.T) {
	var panicCount atomic.Int32
	var shouldPanic atomic.Bool
	shouldPanic.Store(true)

	v := New(func() (string, error) {
		if shouldPanic.Load() {
			panicCount.Add(1)
			panic("intentional init panic")
		}
		return "success after panic", nil
	})

	// Call 1 panics
	assertPanics(t, func() {
		_, _ = v.Get()
	})

	if panicCount.Load() != 1 {
		t.Fatalf("expected 1 panic, got %d", panicCount.Load())
	}

	// Turn off panic, next call should acquire lock and complete successfully
	shouldPanic.Store(false)
	val, err := v.Get()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "success after panic" {
		t.Fatalf("unexpected value: %q", val)
	}
}

func assertPanics(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic, function returned normally")
		}
	}()
	fn()
}
