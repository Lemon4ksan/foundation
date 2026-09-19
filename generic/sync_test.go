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

	for i := 0; i < numWaiters; i++ {
		go func(idx int) {
			defer wg.Done()
			results[idx], errs[idx] = sf.Do(100, func() (string, error) {
				time.Sleep(20 * time.Millisecond)
				return "computed_val", nil
			})
		}(i)
	}

	wg.Wait()

	for i := 0; i < numWaiters; i++ {
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
		var pe *PanicError
		if !errors.As(err, &pe) {
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
