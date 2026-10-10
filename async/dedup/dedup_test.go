// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dedup

import (
	"context"
	"errors"
	"fmt"
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

		// Caller A (Initiator)
		wg.Go(func() {
			_, errA = g.Do(ctxA, "key-2", fn)
		})

		// Caller B (Secondary Waiter)
		wg.Go(func() {
			val, err := g.Do(ctxB, "key-2", fn)
			mu.Lock()
			resultB = val
			errB = err
			mu.Unlock()
		})

		// Let both enter and wait
		synctest.Wait()

		// Cancel Caller A while execution is in progress
		cancelA()
		synctest.Wait()

		if errA == nil || !errors.Is(errA, context.Canceled) {
			t.Errorf("caller A error = %v, want context.Canceled", errA)
		}

		// Advance virtual time so worker finishes for Caller B
		time.Sleep(150 * time.Millisecond)
		wg.Wait()

		mu.Lock()
		defer mu.Unlock()

		if errB != nil {
			t.Errorf("caller B failed: %v", errB)
		}
		if resultB != "success" {
			t.Errorf("caller B result = %q, want %q", resultB, "success")
		}

		if got := runCount.Load(); got != 1 {
			t.Errorf("total executions = %d, want 1", got)
		}
	})
}

func TestGroup_Do_AllWaitersCancelled(t *testing.T) {
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
				return "done", nil
			}
		}

		var wg sync.WaitGroup

		wg.Go(func() {
			_, _ = g.Do(ctxA, "key-3", fn)
		})

		wg.Go(func() {
			_, _ = g.Do(ctxB, "key-3", fn)
		})

		// Let both enter and block
		synctest.Wait()

		// Cancel all waiting callers
		cancelA()
		cancelB()

		// Allow cancellation propagation into the worker's internal context
		synctest.Wait()
		wg.Wait()

		if !workerCancelled.Load() {
			t.Errorf("worker was not cancelled after all waiters aborted")
		}
	})
}

func TestGroup_Do_PanicIsolation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := &Group[string, string]{}

		fnStarted := make(chan struct{})
		fn := func(_ context.Context) (string, error) {
			close(fnStarted)
			time.Sleep(50 * time.Millisecond)
			panic("simulated-panic")
		}

		var (
			recoveredPanic any
			secondErr      error
			wg             sync.WaitGroup
		)

		wg.Add(2)

		// Caller 1 (Initiator)
		go func() {
			defer wg.Done()
			defer func() {
				recoveredPanic = recover()
			}()
			_, _ = g.Do(t.Context(), "key-panic", fn)
		}()

		<-fnStarted

		// Caller 2 (Secondary waiter)
		go func() {
			defer wg.Done()
			_, secondErr = g.Do(t.Context(), "key-panic", fn)
		}()

		synctest.Wait()
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

		fn := func(_ context.Context) (string, error) {
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

		fn := func(_ context.Context) (string, error) {
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

		fn := func(_ context.Context) (string, error) {
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

		synctest.Wait()
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

func TestGroup_EdgeCases_NilAndInvalidInputs(t *testing.T) {
	var nilGroup *Group[string, int]
	ctx := context.Background()

	_, err := nilGroup.Do(ctx, "key", func(_ context.Context) (int, error) { return 1, nil })
	if err == nil {
		t.Error("expected error for nil group.Do")
	}

	g := &Group[string, int]{}
	_, err = g.Do(ctx, "key", nil)
	if err == nil {
		t.Error("expected error for nil callFn in Do")
	}

	// DoChan nil checks
	ch := nilGroup.DoChan(ctx, "key", func(_ context.Context) (int, error) { return 1, nil })
	res := <-ch
	if res.Err == nil {
		t.Error("expected error in DoChan for nil group")
	}

	ch = g.DoChan(ctx, "key", nil)
	res = <-ch
	if res.Err == nil {
		t.Error("expected error in DoChan for nil callFn")
	}

	// Expired context in DoChan
	cancCtx, cancel := context.WithCancel(context.Background())
	cancel()
	ch = g.DoChan(cancCtx, "key", func(_ context.Context) (int, error) { return 1, nil })
	res = <-ch
	if res.Err == nil {
		t.Error("expected error in DoChan for expired context")
	}
}

func TestGroup_DoChan_SuccessAndCoalesce(t *testing.T) {
	g := &Group[string, string]{}
	ctx := context.Background()

	ch1 := g.DoChan(ctx, "chan-key", func(_ context.Context) (string, error) {
		time.Sleep(20 * time.Millisecond)
		return "chan-val", nil
	})

	ch2 := g.DoChan(ctx, "chan-key", func(_ context.Context) (string, error) {
		return "should-not-run", nil
	})

	res1 := <-ch1
	res2 := <-ch2

	if res1.Val != "chan-val" || res1.Err != nil {
		t.Errorf("res1 = %+v, want 'chan-val'", res1)
	}
	if res2.Val != "chan-val" || res2.Err != nil {
		t.Errorf("res2 = %+v, want 'chan-val'", res2)
	}
}

func TestGroup_CallAlreadyDoneBranches(t *testing.T) {
	g := &Group[string, string]{
		calls: make(map[string]*call[string]),
	}

	// Manually inject a completed call
	g.calls["done-key"] = &call[string]{
		done: true,
		val:  "done-value",
	}

	val, err := g.Do(context.Background(), "done-key", func(_ context.Context) (string, error) {
		return "unexpected", nil
	})
	if err != nil || val != "done-value" {
		t.Errorf("Do returned val=%q, err=%v, want 'done-value'", val, err)
	}

	// Inject a completed panic call
	g.calls["panic-key"] = &call[string]{
		done:     true,
		panicVal: "some-panic",
	}
	_, err = g.Do(context.Background(), "panic-key", func(_ context.Context) (string, error) {
		return "unexpected", nil
	})
	if !errors.Is(err, ErrWorkerPanicked) {
		t.Errorf("Do returned err=%v, want ErrWorkerPanicked", err)
	}

	// DoChan on already completed call
	ch := g.DoChan(context.Background(), "done-key", func(_ context.Context) (string, error) {
		return "unexpected", nil
	})
	res := <-ch
	if res.Val != "done-value" || res.Err != nil {
		t.Errorf("DoChan on done call returned %+v, want 'done-value'", res)
	}

	// DoChan on already panicked call
	ch = g.DoChan(context.Background(), "panic-key", func(_ context.Context) (string, error) {
		return "unexpected", nil
	})
	res = <-ch
	if !errors.Is(res.Err, ErrWorkerPanicked) {
		t.Errorf("DoChan on panic call returned %+v, want ErrWorkerPanicked", res)
	}
}

func TestGroup_Wait_CancellationAfterDone(t *testing.T) {
	g := &Group[string, string]{}

	for range 30 {
		c := &call[string]{
			done: true,
		}
		resCh := make(chan Result[string], 1)
		resCh <- Result[string]{Val: "cancelled-but-done"}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		val, err := g.wait(ctx, "test-key", c, resCh)
		if err != nil || val != "cancelled-but-done" {
			t.Errorf("wait with cancelled ctx but done=true returned val=%q, err=%v", val, err)
		}
	}

	for range 30 {
		func() {
			c := &call[string]{
				done: true,
			}
			resCh := make(chan Result[string], 1)
			resCh <- Result[string]{PanicVal: "wait-panic"}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			defer func() {
				r := recover()
				if r != "wait-panic" {
					t.Errorf("recovered = %v, want 'wait-panic'", r)
				}
			}()

			_, _ = g.wait(ctx, "test-key-panic", c, resCh)
		}()
	}
}

func TestEmpirical_Dedup_KeysSeq_Empty(t *testing.T) {
	g := &Group[string, int]{}

	if g.InFlight() != 0 || g.Len() != 0 {
		t.Fatalf("expected 0 in flight, got %d", g.InFlight())
	}

	for k := range g.KeysSeq() {
		t.Fatalf("expected no keys, got %s", k)
	}
}

func TestEmpirical_Dedup_KeysSeq_InFlightAndEarlyTermination(t *testing.T) {
	g := &Group[string, int]{}

	release := make(chan struct{})
	var started sync.WaitGroup

	const numKeys = 5
	for i := range numKeys {
		started.Add(1)
		key := fmt.Sprintf("key-%d", i)
		go func(k string) {
			_, _ = g.Do(context.Background(), k, func(_ context.Context) (int, error) {
				started.Done()
				<-release
				return 42, nil
			})
		}(key)
	}

	started.Wait()

	if g.InFlight() != numKeys || g.Len() != numKeys {
		t.Fatalf("expected %d in flight, got %d", numKeys, g.InFlight())
	}

	// 1. Full traversal
	keys := make(map[string]bool)
	for k := range g.KeysSeq() {
		keys[k] = true
	}
	if len(keys) != numKeys {
		t.Fatalf("expected %d in-flight keys, got %d", numKeys, len(keys))
	}

	// 2. Early termination: break after 0 iterations (first item returns false)
	count := 0
	for range g.KeysSeq() {
		count++
		break
	}
	if count != 1 {
		t.Fatalf("expected count 1 on immediate break, got %d", count)
	}

	// 3. Early termination: break after 2 iterations
	count = 0
	for range g.KeysSeq() {
		count++
		if count == 2 {
			break
		}
	}
	if count != 2 {
		t.Fatalf("expected count 2, got %d", count)
	}

	close(release)
	time.Sleep(20 * time.Millisecond)

	emptyCount := 0
	for range g.KeysSeq() {
		emptyCount++
	}
	if emptyCount != 0 {
		t.Fatalf("expected 0 in-flight keys after completion, got %d", emptyCount)
	}
	if g.InFlight() != 0 {
		t.Fatalf("expected 0 in flight, got %d", g.InFlight())
	}
	if g.Len() != 0 {
		t.Fatalf("expected 0 len, got %d", g.Len())
	}
}

func TestGroup_NilReceiver(t *testing.T) {
	var g *Group[string, int]

	count := 0
	for range g.KeysSeq() {
		count++
	}
	if count != 0 {
		t.Fatalf("expected 0 from nil KeysSeq, got %d", count)
	}
	if n := g.InFlight(); n != 0 {
		t.Fatalf("expected 0 from nil InFlight, got %d", n)
	}
	if n := g.Len(); n != 0 {
		t.Fatalf("expected 0 from nil Len, got %d", n)
	}
}

func BenchmarkKeysSeq_Empty(b *testing.B) {
	g := &Group[string, int]{}

	b.ReportAllocs()

	for b.Loop() {
		for k := range g.KeysSeq() {
			_ = k
		}
	}
}
