// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lazy_test

import (
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/lemon4ksan/foundation/sync/lazy"
)

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

	v := lazy.New(func() (*complexData, error) {
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

	v := lazy.New(func() (int64, error) {
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

	v := lazy.New(func() (string, error) {
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

	v := lazy.New(func() (string, error) {
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
