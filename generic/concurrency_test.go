// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generic

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSingleFlight_Panic_UnblocksWaiters(t *testing.T) {
	sf := NewSingleFlight[string]()
	const numWaiters = 10

	var (
		wg        sync.WaitGroup
		startGate = make(chan struct{})
		inFlight  = make(chan struct{})
	)

	wg.Add(numWaiters + 1)

	// Initiator goroutine: panics during execution
	var initiatorPanicked bool
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				initiatorPanicked = true
			}
		}()

		<-startGate
		_, _ = sf.Do("panic_key", func() (string, error) {
			close(inFlight) // Signal waiters that execution has started
			time.Sleep(20 * time.Millisecond)
			panic("deliberate worker panic")
		})
	}()

	// Waiter results
	errs := make([]error, numWaiters)

	// Secondary waiter goroutines
	for i := 0; i < numWaiters; i++ {
		go func(idx int) {
			defer wg.Done()
			<-inFlight // Ensure initiator is already inside Do and running fn
			_, errs[idx] = sf.Do("panic_key", func() (string, error) {
				return "should not be called", nil
			})
		}(i)
	}

	close(startGate)

	// Deadlock guard: test must finish within 2 seconds
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success: all goroutines returned without deadlocking
	case <-time.After(2 * time.Second):
		t.Fatal("DEADLOCK DETECTED: SingleFlight.Do failed to unblock waiters on panic")
	}

	if !initiatorPanicked {
		t.Errorf("expected initiator to panic, but it did not")
	}

	for i, err := range errs {
		if err == nil {
			t.Errorf("waiter %d: expected error on initiator panic, got nil", i)
		}
		var pe *PanicError
		if !errors.As(err, &pe) {
			t.Errorf("waiter %d: expected *PanicError, got %v", i, err)
		}
	}
}

func TestSingleFlight_Panic_KeyReclaimedForSubsequentCalls(t *testing.T) {
	sf := NewSingleFlight[int]()

	// First call panics
	func() {
		defer func() {
			_ = recover()
		}()
		_, _ = sf.Do("test_key", func() (int, error) {
			panic("boom")
		})
	}()

	// Subsequent call for the same key must execute and return valid result
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

func TestSingleFlight_NilGuards(t *testing.T) {
	var nilSF *SingleFlight[int]
	_, err := nilSF.Do("key", func() (int, error) { return 1, nil })
	if err == nil || err.Error() != "singleflight is nil" {
		t.Errorf("expected 'singleflight is nil', got: %v", err)
	}

	sf := NewSingleFlight[int]()
	_, err2 := sf.Do("key", nil)
	if err2 == nil || err2.Error() != "singleflight fn is nil" {
		t.Errorf("expected 'singleflight fn is nil', got: %v", err2)
	}
}

// TestDataLoader_PanicSafety verifies that when batchFn panics:
// 1. The background goroutine does not terminate the process.
// 2. All concurrent callers waiting on Load() unblock promptly.
// 3. Callers receive an error wrapping ErrBatchPanicked.
// 4. Subsequent batches on the same DataLoader instance continue working normally.
func TestDataLoader_PanicSafety(t *testing.T) {
	var callCount int32
	shouldPanic := true

	dl := NewDataLoader[string, int](
		20*time.Millisecond,
		func(ctx context.Context, keys []string) (map[string]int, error) {
			atomic.AddInt32(&callCount, 1)
			if shouldPanic {
				panic("database connection collapsed")
			}
			res := make(map[string]int)
			for _, k := range keys {
				res[k] = len(k)
			}
			return res, nil
		},
	)

	const callers = 5
	var wg sync.WaitGroup
	wg.Add(callers)

	errs := make([]error, callers)
	for i := 0; i < callers; i++ {
		idx := i
		go func() {
			defer wg.Done()
			_, errs[idx] = dl.Load(context.Background(), fmt.Sprintf("key-%d", idx))
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Succeeded in unblocking
	case <-time.After(2 * time.Second):
		t.Fatal("deadlock: callers failed to unblock after batchFn panic")
	}

	for i, err := range errs {
		if err == nil {
			t.Errorf("caller %d expected error, got nil", i)
		} else if !errors.Is(err, ErrBatchPanicked) {
			t.Errorf("caller %d expected ErrBatchPanicked, got %v", i, err)
		}
	}

	// Verify DataLoader recovery: subsequent batch should work normally
	shouldPanic = false
	val, err := dl.Load(context.Background(), "healthy-key")
	if err != nil {
		t.Fatalf("expected subsequent load to succeed, got error: %v", err)
	}
	if val != len("healthy-key") {
		t.Fatalf("expected value %d, got %d", len("healthy-key"), val)
	}
}

// TestDataLoader_MissingKeysAndNilResult verifies that when batchFn returns fewer
// results than requested keys or returns a nil map:
// 1. Existing keys receive their correct values.
// 2. Missing keys receive ErrKeyNotFound.
// 3. Nil map results in ErrKeyNotFound for all requested keys without panicking.
func TestDataLoader_MissingKeysAndNilResult(t *testing.T) {
	// Subtest 1: Partial results
	dlPartial := NewDataLoader[string, string](
		15*time.Millisecond,
		func(ctx context.Context, keys []string) (map[string]string, error) {
			return map[string]string{
				"present1": "val1",
				"present2": "val2",
				// "missing" omitted
			}, nil
		},
	)

	var wg sync.WaitGroup
	wg.Add(3)
	var (
		v1, v2, v3 string
		e1, e2, e3 error
	)

	go func() { defer wg.Done(); v1, e1 = dlPartial.Load(context.Background(), "present1") }()
	go func() { defer wg.Done(); v2, e2 = dlPartial.Load(context.Background(), "present2") }()
	go func() { defer wg.Done(); v3, e3 = dlPartial.Load(context.Background(), "missing") }()
	wg.Wait()

	if e1 != nil || v1 != "val1" {
		t.Errorf("expected present1='val1', got val=%q, err=%v", v1, e1)
	}
	if e2 != nil || v2 != "val2" {
		t.Errorf("expected present2='val2', got val=%q, err=%v", v2, e2)
	}
	if e3 == nil || !errors.Is(e3, ErrKeyNotFound) || v3 != "" {
		t.Errorf("expected ErrKeyNotFound and empty val for missing key, got val=%q, err=%v", v3, e3)
	}

	// Subtest 2: Nil map returned
	dlNilMap := NewDataLoader[string, int](
		15*time.Millisecond,
		func(ctx context.Context, keys []string) (map[string]int, error) {
			return nil, nil
		},
	)

	val, err := dlNilMap.Load(context.Background(), "any-key")
	if val != 0 || !errors.Is(err, ErrKeyNotFound) {
		t.Errorf("expected zero val and ErrKeyNotFound on nil map, got val=%d, err=%v", val, err)
	}
}

// TestDataLoader_ContextCancellationCleanup verifies that:
// 1. An already cancelled context returns immediately without executing batchFn.
// 2. A context cancelled during the delay window deregisters its channel.
// 3. If all callers cancel, the timer is stopped and batchFn is NOT executed.
// 4. If only some callers cancel, the remaining callers receive their results.
func TestDataLoader_ContextCancellationCleanup(t *testing.T) {
	var batchExecutions int32

	dl := NewDataLoader[string, int](
		50*time.Millisecond,
		func(ctx context.Context, keys []string) (map[string]int, error) {
			atomic.AddInt32(&batchExecutions, 1)
			res := make(map[string]int)
			for _, k := range keys {
				res[k] = 42
			}
			return res, nil
		},
	)

	// Case 1: Pre-cancelled context
	preCtx, preCancel := context.WithCancel(context.Background())
	preCancel()

	_, err := dl.Load(preCtx, "pre-canceled")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if atomic.LoadInt32(&batchExecutions) != 0 {
		t.Fatalf("expected 0 batch executions for pre-canceled context, got %d", batchExecutions)
	}

	// Case 2: All callers cancel before window fires
	ctxCancel, cancelFunc := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, _ = dl.Load(ctxCancel, "k1")
	}()
	go func() {
		defer wg.Done()
		_, _ = dl.Load(ctxCancel, "k2")
	}()

	time.Sleep(10 * time.Millisecond)
	cancelFunc() // Cancel all pending callers
	wg.Wait()

	// Wait past the batch window (50ms + margin)
	time.Sleep(70 * time.Millisecond)

	if atomic.LoadInt32(&batchExecutions) != 0 {
		t.Fatalf("expected 0 batch executions when all callers canceled, got %d", batchExecutions)
	}

	// Case 3: Partial cancellation - one cancels, one remains
	ctxCancel1, cancel1 := context.WithCancel(context.Background())
	var (
		valSurviving int
		errSurviving error
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = dl.Load(ctxCancel1, "will-cancel")
	}()
	go func() {
		defer wg.Done()
		valSurviving, errSurviving = dl.Load(context.Background(), "will-survive")
	}()

	time.Sleep(10 * time.Millisecond)
	cancel1() // Cancel only the first
	wg.Wait()

	if errSurviving != nil || valSurviving != 42 {
		t.Errorf("expected surviving caller to receive 42, nil; got val=%d, err=%v", valSurviving, errSurviving)
	}
	if atomic.LoadInt32(&batchExecutions) != 1 {
		t.Errorf("expected exactly 1 batch execution for partial cancellation, got %d", batchExecutions)
	}
}

// TestDataLoader_DuplicateKeys verifies that multiple callers requesting the
// identical key within the window are deduplicated in batchFn input, and all
// callers receive the identical result.
func TestDataLoader_DuplicateKeys(t *testing.T) {
	var requestedKeys []string
	var mu sync.Mutex

	dl := NewDataLoader[string, int](
		25*time.Millisecond,
		func(ctx context.Context, keys []string) (map[string]int, error) {
			mu.Lock()
			requestedKeys = append(requestedKeys, keys...)
			mu.Unlock()
			return map[string]int{"shared-key": 999}, nil
		},
	)

	const numCallers = 10
	var wg sync.WaitGroup
	wg.Add(numCallers)

	results := make([]int, numCallers)
	errs := make([]error, numCallers)

	for i := 0; i < numCallers; i++ {
		idx := i
		go func() {
			defer wg.Done()
			results[idx], errs[idx] = dl.Load(context.Background(), "shared-key")
		}()
	}

	wg.Wait()

	for i := 0; i < numCallers; i++ {
		if errs[i] != nil || results[i] != 999 {
			t.Errorf("caller %d expected (999, nil), got (%d, %v)", i, results[i], errs[i])
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if len(requestedKeys) != 1 {
		t.Errorf("expected batchFn to receive exactly 1 key, got %d (%v)", len(requestedKeys), requestedKeys)
	}
}

// TestDataLoader_NilGuards verifies that calling Load on nil DataLoader, nil batchFn,
// or with nil Context returns appropriate errors without crashing.
func TestDataLoader_NilGuards(t *testing.T) {
	var nilDL *DataLoader[string, int]
	_, err1 := nilDL.Load(context.Background(), "k")
	if !errors.Is(err1, ErrDataLoaderNil) {
		t.Errorf("expected ErrDataLoaderNil, got %v", err1)
	}

	dlNilFn := NewDataLoader[string, int](10*time.Millisecond, nil)
	_, err2 := dlNilFn.Load(context.Background(), "k")
	if !errors.Is(err2, ErrBatchFnNil) {
		t.Errorf("expected ErrBatchFnNil, got %v", err2)
	}

	// Nil context should default to context.Background() and succeed
	dlValid := NewDataLoader[string, int](
		10*time.Millisecond,
		func(ctx context.Context, keys []string) (map[string]int, error) {
			return map[string]int{"ok": 1}, nil
		},
	)
	val, err3 := dlValid.Load(nil, "ok")
	if err3 != nil || val != 1 {
		t.Errorf("expected (1, nil) for nil context, got (%d, %v)", val, err3)
	}
}

// TestDataLoader_HighConcurrencyStress runs 200 concurrent callers mixing successes,
// missing keys, and context cancellations to verify race-free execution under -race.
func TestDataLoader_HighConcurrencyStress(t *testing.T) {
	dl := NewDataLoader[int, int](10*time.Millisecond, func(ctx context.Context, keys []int) (map[int]int, error) {
		resMap := make(map[int]int)
		for _, k := range keys {
			if k%3 != 0 { // Every 3rd key is intentionally missing
				resMap[k] = k * 10
			}
		}
		return resMap, nil
	})

	const numGoroutines = 200
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		key := i
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()

			val, err := dl.Load(ctx, key)
			if key%3 == 0 {
				if !errors.Is(err, ErrKeyNotFound) && !errors.Is(err, context.DeadlineExceeded) {
					t.Errorf("key %d expected ErrKeyNotFound or timeout, got %v", key, err)
				}
			} else {
				if err == nil && val != key*10 {
					t.Errorf("key %d expected %d, got %d", key, key*10, val)
				}
			}
		}()
	}

	wg.Wait()
}
