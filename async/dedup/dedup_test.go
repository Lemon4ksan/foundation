// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dedup

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

func TestGroup_Do_SuppressedExecution(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		group := &Group[string, string]{}

		var (
			runCount atomic.Int32
			mu       sync.Mutex
			results  []string
		)

		fn := func(_ context.Context) (string, error) {
			runCount.Add(1)
			time.Sleep(100 * time.Millisecond)
			return "shared-value", nil
		}

		var wg sync.WaitGroup

		const waiters = 5

		for range waiters {
			wg.Go(func() {
				val, err := group.Do(t.Context(), "key-1", fn)
				if err != nil {
					t.Errorf("Do() failed: %v", err)
					return
				}

				mu.Lock()

				results = append(results, val)
				mu.Unlock()
			})
		}

		// Let all goroutines enter Do and block
		synctest.Wait()

		// Advance virtual time to complete the worker execution
		time.Sleep(150 * time.Millisecond)
		wg.Wait()

		got := runCount.Load()
		if got != 1 {
			t.Errorf("worker execution count = %d, want 1", got)
		}

		mu.Lock()
		defer mu.Unlock()

		if len(results) != waiters {
			t.Errorf("results count = %d, want %d", len(results), waiters)
		}

		for i, res := range results {
			if res != "shared-value" {
				t.Errorf("results[%d] = %q, want %q", i, res, "shared-value")
			}
		}
	})
}

func TestGroup_Do_PartialContextCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := &Group[string, string]{}

		ctxA, cancelA := context.WithCancel(t.Context())
		t.Cleanup(cancelA)
		ctxB := t.Context()

		var (
			runCount atomic.Int32
			mu       sync.Mutex
			resultB  string
			errB     error
			errA     error
		)

		fn := func(_ context.Context) (string, error) {
			runCount.Add(1)
			time.Sleep(100 * time.Millisecond)
			return "success", nil
		}

		var wg sync.WaitGroup

		// Goroutine A: Will cancel while waiting
		wg.Go(func() {
			_, err := g.Do(ctxA, "key-2", fn)

			mu.Lock()
			errA = err
			mu.Unlock()
		})

		// Goroutine B: Will wait and successfully complete
		wg.Go(func() {
			r, err := g.Do(ctxB, "key-2", fn)

			mu.Lock()
			resultB = r
			errB = err
			mu.Unlock()
		})

		// Let both goroutines enter Do and block
		synctest.Wait()
		cancelA()
		synctest.Wait()

		mu.Lock()
		eA := errA
		mu.Unlock()

		if eA == nil || !errors.Is(eA, context.Canceled) {
			t.Errorf("caller A error = %v, want context.Canceled", eA)
		}

		// Check that B is still waiting on the execution (under mutex protection)
		mu.Lock()
		resB := resultB
		eB := errB
		mu.Unlock()

		if resB != "" || eB != nil {
			t.Errorf("caller B completed prematurely: val = %q, err = %v", resB, eB)
		}

		// Advance virtual time past worker execution threshold
		time.Sleep(150 * time.Millisecond)
		wg.Wait()

		gotCount := runCount.Load()
		if gotCount != 1 {
			t.Errorf("worker execution count = %d, want 1", gotCount)
		}

		mu.Lock()
		resBFinal := resultB
		eBFinal := errB
		mu.Unlock()

		if eBFinal != nil {
			t.Errorf("caller B error = %v, want nil", eBFinal)
		}

		if resBFinal != "success" {
			t.Errorf("caller B result = %q, want %q", resBFinal, "success")
		}
	})
}

func TestGroup_Do_WorkerContextCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := &Group[string, string]{}

		ctxA, cancelA := context.WithCancel(t.Context())
		t.Cleanup(cancelA)
		ctxB, cancelB := context.WithCancel(t.Context())
		t.Cleanup(cancelB)

		var workerCancelled atomic.Bool

		fn := func(workerCtx context.Context) (string, error) {
			select {
			case <-workerCtx.Done():
				workerCancelled.Store(true)
				return "", workerCtx.Err()
			case <-time.After(100 * time.Millisecond):
				return "completed", nil
			}
		}

		var wg sync.WaitGroup

		wg.Go(func() { g.Do(ctxA, "key-3", fn) })
		wg.Go(func() { g.Do(ctxB, "key-3", fn) })

		// Let them start and block inside worker selection
		synctest.Wait()
		cancelA()
		cancelB()
		synctest.Wait()

		if !workerCancelled.Load() {
			t.Error("worker context was not cancelled after all waiters cancelled their contexts")
		}

		wg.Wait()
	})
}

func TestGroup_Do_PanicIsolation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := &Group[string, string]{}
		ctx := t.Context()

		fn := func(workerCtx context.Context) (string, error) {
			time.Sleep(50 * time.Millisecond)
			panic("simulated-panic")
		}

		var wg sync.WaitGroup

		var (
			recoveredPanic any
			secondErr      error
		)

		// Goroutine 1: Initiator (will receive and propagate panic)
		wg.Go(func() {
			defer func() {
				if r := recover(); r != nil {
					recoveredPanic = r
				}
			}()

			g.Do(ctx, "key-4", fn)
		})

		// Ensure the initiator has successfully registered first
		synctest.Wait()

		// Goroutine 2: Secondary waiter (will receive ErrWorkerPanicked)
		wg.Go(func() {
			_, secondErr = g.Do(ctx, "key-4", fn)
		})

		// Advance virtual time past the panic point
		time.Sleep(100 * time.Millisecond)
		wg.Wait()

		if recoveredPanic != "simulated-panic" {
			t.Errorf("recovered panic = %v, want %q", recoveredPanic, "simulated-panic")
		}

		if secondErr == nil || !errors.Is(secondErr, ErrWorkerPanicked) {
			t.Errorf("second caller error = %v, want ErrWorkerPanicked", secondErr)
		}
	})
}

func TestGroup_Do_ExpiredContext(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := &Group[string, string]{}

		ctx, cancel := context.WithCancel(t.Context())
		cancel() // Cancel immediately

		var runCount atomic.Int32

		fn := func(workerCtx context.Context) (string, error) {
			runCount.Add(1)
			return "data", nil
		}

		_, err := g.Do(ctx, "key-5", fn)
		if err == nil || !errors.Is(err, context.Canceled) {
			t.Errorf("execution error = %v, want context.Canceled", err)
		}

		if got := runCount.Load(); got != 0 {
			t.Errorf("worker execution count = %d, want 0", got)
		}
	})
}

func TestGroup_Do_KeyReutilization(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := &Group[string, string]{}
		ctx := t.Context()

		var runCount atomic.Int32

		fn := func(workerCtx context.Context) (string, error) {
			runCount.Add(1)
			return "value", nil
		}

		// First execution
		val1, err1 := g.Do(ctx, "key-reused", fn)
		if err1 != nil {
			t.Fatalf("first execution failed: %v", err1)
		}

		if val1 != "value" {
			t.Errorf("first result = %q, want 'value'", val1)
		}

		// Second execution (should trigger fn again because the first is done)
		val2, err2 := g.Do(ctx, "key-reused", fn)
		if err2 != nil {
			t.Fatalf("second execution failed: %v", err2)
		}

		if val2 != "value" {
			t.Errorf("second result = %q, want 'value'", val2)
		}

		if got := runCount.Load(); got != 2 {
			t.Errorf("total executions = %d, want 2", got)
		}
	})
}

func TestGroup_Do_MultiKeyParallelism(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := &Group[string, string]{}

		var (
			runCount atomic.Int32
			mu       sync.Mutex
			results  = make(map[string]string)
		)

		fn := func(workerCtx context.Context) (string, error) {
			runCount.Add(1)
			time.Sleep(100 * time.Millisecond)
			return "val", nil
		}

		var wg sync.WaitGroup

		wg.Go(func() {
			val, _ := g.Do(t.Context(), "key-a", fn)

			mu.Lock()
			results["a"] = val
			mu.Unlock()
		})

		wg.Go(func() {
			val, _ := g.Do(t.Context(), "key-b", fn)

			mu.Lock()
			results["b"] = val
			mu.Unlock()
		})

		// Let both start and block inside virtual time sleep
		synctest.Wait()

		// Advance virtual time to complete both executions
		time.Sleep(150 * time.Millisecond)
		wg.Wait()

		gotCount := runCount.Load()
		if gotCount != 2 {
			t.Errorf("total executions = %d, want 2 (no suppression for different keys)", gotCount)
		}

		mu.Lock()
		defer mu.Unlock()

		if results["a"] != "val" || results["b"] != "val" {
			t.Errorf("results mismatch: %v", results)
		}
	})
}
