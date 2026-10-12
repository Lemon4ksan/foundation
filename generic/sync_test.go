// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generic

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSingleflight_Generic_NormalExecution(t *testing.T) {
	sf := NewSingleflight[int, string]()
	const numWaiters = 5

	var wg sync.WaitGroup
	wg.Add(numWaiters)

	results := make([]string, numWaiters)
	errs := make([]error, numWaiters)

	for i := range numWaiters {
		go func(idx int) {
			defer wg.Done()
			results[idx], errs[idx] = sf.Do(100, func() (string, error) {
				time.Sleep(20 * time.Millisecond)
				return "computed_val", nil
			})
		}(i)
	}

	wg.Wait()

	for i := range numWaiters {
		if errs[i] != nil {
			t.Fatalf("waiter %d received unexpected error: %v", i, errs[i])
		}
		if results[i] != "computed_val" {
			t.Fatalf("waiter %d received %q, want 'computed_val'", i, results[i])
		}
	}
}

func TestSingleflight_Generic_Panic_UnblocksWaiters(t *testing.T) {
	sf := NewSingleflight[int, string]()
	const numWaiters = 5

	var (
		wg                sync.WaitGroup
		inFlight          = make(chan struct{})
		initiatorPanicked bool
	)

	wg.Add(numWaiters + 1)

	// Initiator goroutine: panics during execution
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				initiatorPanicked = true
			}
		}()

		_, _ = sf.Do(999, func() (string, error) {
			close(inFlight)
			time.Sleep(20 * time.Millisecond)
			panic("generic singleflight panic")
		})
	}()

	errs := make([]error, numWaiters)
	for i := range numWaiters {
		go func(idx int) {
			defer wg.Done()
			<-inFlight
			_, errs[idx] = sf.Do(999, func() (string, error) {
				return "unreachable", nil
			})
		}(i)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("DEADLOCK DETECTED: Singleflight[K, V].Do failed to unblock waiters on panic")
	}

	if !initiatorPanicked {
		t.Errorf("expected initiator to panic")
	}

	for i, err := range errs {
		if err == nil {
			t.Errorf("waiter %d: expected non-nil error on initiator panic", i)
		}
		if _, ok := errors.AsType[*PanicError](err); !ok {
			t.Errorf("waiter %d: expected error to wrap *PanicError, got: %v", i, err)
		}
	}
}

func TestSingleflight_Generic_Panic_KeyReclaimedForSubsequentCalls(t *testing.T) {
	sf := NewSingleflight[string, int]()

	// First call panics
	func() {
		defer func() {
			_ = recover()
		}()
		_, _ = sf.Do("test_key", func() (int, error) {
			panic("deliberate boom")
		})
	}()

	// Subsequent call for the same key must execute and return valid result without deadlocking
	done := make(chan struct{})
	var (
		val int
		err error
	)
	go func() {
		val, err = sf.Do("test_key", func() (int, error) {
			return 42, nil
		})
		close(done)
	}()

	select {
	case <-done:
		if err != nil {
			t.Fatalf("expected nil error on subsequent call, got: %v", err)
		}
		if val != 42 {
			t.Fatalf("expected val 42, got %d", val)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("DEADLOCK DETECTED: subsequent call for key hung after prior panic")
	}
}

func TestPanicError_Methods(t *testing.T) {
	innerErr := errors.New("underlying cause")
	pe := newPanicError(innerErr)

	if !errors.Is(pe, innerErr) {
		t.Errorf("expected errors.Is to match innerErr via Unwrap")
	}

	errStr := pe.Error()
	if !strings.Contains(errStr, "underlying cause") {
		t.Errorf("expected error string to contain panic value, got: %s", errStr)
	}
	if !strings.Contains(errStr, "TestPanicError_Methods") {
		t.Errorf("expected stack trace to contain test function name, got: %s", errStr)
	}

	// Test with non-error panic value
	peNonErr := newPanicError("string panic")
	if peNonErr.Unwrap() != nil {
		t.Errorf("expected nil from Unwrap on non-error panic, got: %v", peNonErr.Unwrap())
	}
	if !strings.Contains(peNonErr.Error(), "string panic") {
		t.Errorf("expected error string to contain 'string panic', got: %s", peNonErr.Error())
	}

	// Test idempotency of newPanicError with existing *PanicError
	peReWrapped := newPanicError(pe)
	if peReWrapped != pe {
		t.Errorf("expected newPanicError to return identical *PanicError pointer")
	}
}

func TestSingleflight_NilGuards(t *testing.T) {
	var nilSF *Singleflight[string, int]
	_, err := nilSF.Do("key", func() (int, error) { return 1, nil })
	if err == nil || err.Error() != "singleflight is nil" {
		t.Errorf("expected 'singleflight is nil', got: %v", err)
	}

	sf := NewSingleflight[string, int]()
	_, err2 := sf.Do("key", nil)
	if err2 == nil || err2.Error() != "singleflight fn is nil" {
		t.Errorf("expected 'singleflight fn is nil', got: %v", err2)
	}
}

func TestGuarded(t *testing.T) {
	g := NewGuarded[int](10)
	if got := g.Get(); got != 10 {
		t.Fatalf("expected 10, got %d", got)
	}

	g.Set(20)
	if got := g.Get(); got != 20 {
		t.Fatalf("expected 20, got %d", got)
	}

	old := g.Swap(30)
	if old != 20 || g.Get() != 30 {
		t.Fatalf("expected old 20, got %d; current %d", old, g.Get())
	}

	res := g.Update(func(c int) int { return c + 5 })
	if res != 35 || g.Get() != 35 {
		t.Fatalf("expected 35, got res %d, current %d", res, g.Get())
	}

	g.Mutate(func(v *int) { *v = 100 })
	if got := g.Get(); got != 100 {
		t.Fatalf("expected 100, got %d", got)
	}

	var readVal int
	g.Read(func(v int) { readVal = v })
	if readVal != 100 {
		t.Fatalf("expected readVal 100, got %d", readVal)
	}
}

func TestSyncMap(t *testing.T) {
	sm := NewSyncMap[string, int]()
	sm.Store("a", 1)
	sm.Store("b", 2)

	v, ok := sm.Load("a")
	if !ok || v != 1 {
		t.Fatalf("expected 1, got %v (ok=%t)", v, ok)
	}

	actual, loaded := sm.LoadOrStore("a", 99)
	if !loaded || actual != 1 {
		t.Fatalf("expected loaded=true, actual=1, got %v, %t", actual, loaded)
	}

	actual2, loaded2 := sm.LoadOrStore("c", 3)
	if loaded2 || actual2 != 3 {
		t.Fatalf("expected loaded=false, actual=3, got %v, %t", actual2, loaded2)
	}

	val, deleted := sm.LoadAndDelete("b")
	if !deleted || val != 2 {
		t.Fatalf("expected deleted=true, val=2, got %v, %t", val, deleted)
	}

	sm.Delete("c")
	if _, ok := sm.Load("c"); ok {
		t.Fatalf("expected key 'c' to be deleted")
	}

	count := 0
	sm.Range(func(k string, v int) bool {
		count++
		return true
	})
	if count != 1 {
		t.Fatalf("expected count 1, got %d", count)
	}

	sm.Clear()
	if _, ok := sm.Load("a"); ok {
		t.Fatalf("expected key 'a' to be cleared")
	}
}

func TestLazy_Infallible(t *testing.T) {
	var count int
	var mu sync.Mutex
	l := NewLazyValue(func() int {
		mu.Lock()
		count++
		mu.Unlock()
		return 42
	})

	const goroutines = 64
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			if v := l.Value(); v != 42 {
				t.Errorf("expected 42, got %d", v)
			}
		}()
	}
	wg.Wait()

	if count != 1 {
		t.Fatalf("expected init to be called exactly once, called %d times", count)
	}

	// Verify 0-alloc fast path
	allocs := testing.AllocsPerRun(1000, func() {
		_ = l.Value()
	})
	if allocs != 0 {
		t.Fatalf("expected 0 allocs on cached Value(), got %f", allocs)
	}
}

func TestLazy_FallibleAndReset(t *testing.T) {
	errBoom := errors.New("boom")
	shouldFail := true

	l := NewLazy(func() (string, error) {
		if shouldFail {
			return "", errBoom
		}
		return "recovered", nil
	})

	val, err := l.Get()
	if !errors.Is(err, errBoom) || val != "" {
		t.Fatalf("expected errBoom, got %v, %v", val, err)
	}

	// Error is cached
	val2, err2 := l.Get()
	if !errors.Is(err2, errBoom) || val2 != "" {
		t.Fatalf("expected cached errBoom, got %v, %v", val2, err2)
	}

	// Reset allows retry
	shouldFail = false
	l.Reset()

	val3, err3 := l.Get()
	if err3 != nil || val3 != "recovered" {
		t.Fatalf("expected recovered, got %v, %v", val3, err3)
	}
}

func TestPool_Generic(t *testing.T) {
	p := NewPool(func() *string {
		s := "fresh"
		return &s
	})

	item := p.Get()
	if *item != "fresh" {
		t.Fatalf("expected 'fresh', got %s", *item)
	}

	*item = "recycled"
	p.Put(item)

	item2 := p.Get()
	if *item2 != "recycled" {
		t.Fatalf("expected 'recycled', got %s", *item2)
	}

	// Nil safety
	p.Put(nil)
}
