// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

func TestMap(t *testing.T) {
	if Map[int, int](nil, func(x int) int { return x }) != nil {
		t.Fatal("expected nil for nil slice")
	}

	in := []int{1, 2, 3}
	out := Map(in, func(x int) int { return x * 2 })

	expected := []int{2, 4, 6}
	for i, v := range out {
		if v != expected[i] {
			t.Errorf("expected %d at index %d, got %d", expected[i], i, v)
		}
	}
}

func TestFilter(t *testing.T) {
	if Filter[int](nil, func(x int) bool { return true }) != nil {
		t.Fatal("expected nil for nil slice")
	}

	in := []int{1, 2, 3, 4, 5}
	evens := Filter(in, func(x int) bool { return x%2 == 0 })
	if len(evens) != 2 || evens[0] != 2 || evens[1] != 4 {
		t.Errorf("expected [2 4], got %v", evens)
	}
}

func TestSet(t *testing.T) {
	s := NewSet(1, 2, 2, 3)
	if len(s) != 3 {
		t.Errorf("expected set size 3, got %d", len(s))
	}

	if !s.Has(1) || !s.Has(2) || !s.Has(3) || s.Has(4) {
		t.Error("unexpected set contents")
	}

	s.Add(4)

	if !s.Has(4) {
		t.Error("expected set to have 4 after Add")
	}

	other := NewSet(3, 4, 5)

	intersection := s.Intersect(other)
	if len(intersection) != 2 || !intersection.Has(3) || !intersection.Has(4) {
		t.Errorf("unexpected intersection: %v", intersection.ToSlice())
	}

	slice := s.ToSlice()
	if len(slice) != 4 {
		t.Errorf("expected slice length 4, got %d", len(slice))
	}
}

func TestCache(t *testing.T) {
	cache := NewCache[string, int]()

	// Miss
	if _, ok := cache.Get("a"); ok {
		t.Fatal("expected cache miss")
	}

	// Hit
	cache.Set("a", 1, 100*time.Millisecond)

	val, ok := cache.Get("a")
	if !ok || val != 1 {
		t.Fatalf("expected cache hit with 1, got %v, %t", val, ok)
	}

	// Expired
	cache.Set("b", 2, 1*time.Millisecond)
	time.Sleep(2 * time.Millisecond)

	if _, ok := cache.Get("b"); ok {
		t.Fatal("expected cache expiration")
	}
}

func TestParallelMap(t *testing.T) {
	// Empty slice
	if ParallelMap[int, int](context.Background(), nil, 2, func(ctx context.Context, x int) int { return x }) != nil {
		t.Fatal("expected nil for empty slice")
	}

	// Negative limit defaults to 1
	in := []int{1, 2, 3}

	out := ParallelMap(context.Background(), in, -1, func(ctx context.Context, x int) int {
		return x * 10
	})
	if len(out) != 3 || out[0] != 10 || out[1] != 20 || out[2] != 30 {
		t.Fatalf("expected ParallelMap with limit -1 to work, got %v", out)
	}

	// Normal ParallelMap
	out2 := ParallelMap(context.Background(), in, 2, func(ctx context.Context, x int) int {
		return x * 10
	})
	if len(out2) != 3 || out2[0] != 10 || out2[1] != 20 || out2[2] != 30 {
		t.Fatalf("expected ParallelMap to work, got %v", out2)
	}

	// Cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	out3 := ParallelMap(ctx, in, 2, func(ctx context.Context, x int) int {
		return x
	})
	if out3 != nil {
		t.Fatal("expected nil result on cancelled context")
	}
}

func TestParallelForEach(t *testing.T) {
	// Empty slice
	if err := ParallelForEach[int](
		context.Background(),
		nil,
		2,
		func(ctx context.Context, x int) error { return nil },
	); err != nil {
		t.Fatalf("unexpected error on empty: %v", err)
	}

	// Negative limit defaults to 1
	in := []int{1, 2, 3}

	var mu sync.Mutex

	sum := 0

	err := ParallelForEach(context.Background(), in, -1, func(ctx context.Context, x int) error {
		mu.Lock()
		sum += x
		mu.Unlock()

		return nil
	})
	if err != nil || sum != 6 {
		t.Fatalf("expected sum 6 and no error, got %d, %v", sum, err)
	}

	// Cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = ParallelForEach(ctx, in, 2, func(ctx context.Context, x int) error {
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	// Return error
	errDummy := errors.New("dummy")

	err = ParallelForEach(context.Background(), in, 2, func(ctx context.Context, x int) error {
		if x == 2 {
			return errDummy
		}

		return nil
	})
	if !errors.Is(err, errDummy) {
		t.Fatalf("expected %v, got %v", errDummy, err)
	}
}

func TestFuture(t *testing.T) {
	expectedErr := errors.New("future err")
	f := NewFutureFunc(func() (int, error) {
		time.Sleep(10 * time.Millisecond)
		return 42, expectedErr
	})

	// Cancelled context before completion
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.Get(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	// Get successfully
	val, err := f.Get(context.Background())
	if val != 42 || !errors.Is(err, expectedErr) {
		t.Fatalf("expected (42, future err), got (%d, %v)", val, err)
	}
}

func TestSingleFlight(t *testing.T) {
	sf := NewSingleFlight[int]()

	var (
		wg    sync.WaitGroup
		calls int32
	)

	numGoroutines := 10

	wg.Add(numGoroutines)

	results := make([]int, numGoroutines)
	for i := range numGoroutines {
		go func(idx int) {
			defer wg.Done()

			val, _ := sf.Do("key", func() (int, error) {
				time.Sleep(10 * time.Millisecond)

				results[idx] = idx // to check who executed it (or that it block-shares)

				return int(atomic.AddInt32(&calls, 1)), nil
			})
			results[idx] = val
		}(i)
	}

	wg.Wait()

	if calls != 1 {
		t.Errorf("expected function to be called exactly once, got %d", calls)
	}

	for i, r := range results {
		if r != 1 {
			t.Errorf("expected result 1 for goroutine %d, got %d", i, r)
		}
	}
}

func TestRetry(t *testing.T) {
	errDummy := errors.New("dummy")

	// Attempts <= 0 defaults to 1
	var calls int

	err := Retry(context.Background(), RetryConfig{Attempts: -1, Delay: 0}, func(ctx context.Context) error {
		calls++
		return errDummy
	})
	if calls != 1 || !errors.Is(err, errDummy) {
		t.Fatalf("expected 1 call and errDummy, got %d and %v", calls, err)
	}

	// Happy path (succeeds on 2nd attempt)
	calls = 0

	err = Retry(
		context.Background(),
		RetryConfig{Attempts: 3, Delay: 1 * time.Millisecond},
		func(ctx context.Context) error {
			calls++
			if calls == 2 {
				return nil
			}

			return errDummy
		},
	)
	if err != nil || calls != 2 {
		t.Fatalf("expected success on 2nd attempt, got error %v and calls %d", err, calls)
	}

	// Context cancellation in loop
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = Retry(ctx, RetryConfig{Attempts: 3, Delay: 0}, func(ctx context.Context) error {
		return errDummy
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	// Context cancellation during delay
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel2()

	err = Retry(ctx2, RetryConfig{Attempts: 3, Delay: 100 * time.Millisecond}, func(ctx context.Context) error {
		return errDummy
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}
}

func TestBackoff(t *testing.T) {
	// Factor <= 0 defaults to 2
	b := NewBackoff(10*time.Millisecond, 100*time.Millisecond, 0.0, 0.0)
	d1 := b.Next()

	d2 := b.Next()
	if d1 != 10*time.Millisecond || d2 != 20*time.Millisecond {
		t.Errorf("expected 10ms then 20ms, got %v, %v", d1, d2)
	}

	// With Jitter (and testing negative bounds, bounds clamp to Max)
	b2 := NewBackoff(10*time.Millisecond, 50*time.Millisecond, 2.0, 0.5)
	for range 20 {
		d := b2.Next()
		if d > 50*time.Millisecond {
			t.Errorf("expected backoff capped to 50ms, got %v", d)
		}
	}

	// Trigger ms < 0 with high jitter
	b3 := NewBackoff(10*time.Millisecond, 50*time.Millisecond, 2.0, 5.0)
	for range 50 {
		_ = b3.Next()
	}

	// Reset
	b2.Reset()

	dFirst := b2.Next()
	if dFirst > 15*time.Millisecond || dFirst < 5*time.Millisecond {
		t.Errorf("expected backoff reset to first range around 10ms, got %v", dFirst)
	}
}

func TestBatchLoader(t *testing.T) {
	var callCount int32

	batchFn := func(ctx context.Context, keys []string) (map[string]int, error) {
		atomic.AddInt32(&callCount, 1)

		res := make(map[string]int)
		for _, k := range keys {
			if k == "error-trigger" {
				return nil, errors.New("batch error")
			}

			if k != "missing-key" {
				res[k] = len(k)
			}
		}

		return res, nil
	}

	dl := NewBatchLoader[string, int](25*time.Millisecond, batchFn)

	var wg sync.WaitGroup
	wg.Add(3)

	var (
		val1, val2       int
		err1, err2, err3 error
	)

	// Normal concurrent loads should be batched

	go func() {
		defer wg.Done()

		val1, err1 = dl.Load(context.Background(), "alice")
	}()

	go func() {
		defer wg.Done()

		val2, err2 = dl.Load(context.Background(), "bob")
	}()

	// Missing key from batch results
	go func() {
		defer wg.Done()

		_, err3 = dl.Load(context.Background(), "missing-key")
	}()

	wg.Wait()

	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("expected exactly 1 batch execution, got %d", callCount)
	}

	if err1 != nil || val1 != 5 {
		t.Errorf("expected alice = 5, got val %d, err %v", val1, err1)
	}

	if err2 != nil || val2 != 3 {
		t.Errorf("expected bob = 3, got val %d, err %v", val2, err2)
	}

	if err3 == nil || err3.Error() != "foundation batchloader: key not found in batch results" {
		t.Errorf("expected key not found error, got %v", err3)
	}

	// Trigger error from batch function
	dl2 := NewBatchLoader[string, int](25*time.Millisecond, batchFn)

	wg.Add(2)

	var errA, errB error
	go func() {
		defer wg.Done()

		_, errA = dl2.Load(context.Background(), "error-trigger")
	}()
	go func() {
		defer wg.Done()

		_, errB = dl2.Load(context.Background(), "other")
	}()

	wg.Wait()

	if errA == nil || errA.Error() != "batch error" || errB == nil || errB.Error() != "batch error" {
		t.Errorf("expected batch error, got errA=%v, errB=%v", errA, errB)
	}

	// Cancelled context on Load
	dl3 := NewBatchLoader[string, int](100*time.Millisecond, batchFn)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, errCancel := dl3.Load(ctx, "cancel-me")
	if !errors.Is(errCancel, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", errCancel)
	}

	// Test executeBatch on empty queue
	dl4 := NewBatchLoader[string, int](5*time.Millisecond, batchFn)
	dl4.executeBatch() // should return immediately without panic/executing batchFn
}

func TestIndexBy(t *testing.T) {
	if IndexBy[string, int](nil, func(x int) string { return "" }) != nil {
		t.Fatal("expected nil for nil slice")
	}

	in := []int{1, 2, 3}

	res := IndexBy(in, func(x int) int { return x * 10 })
	if len(res) != 3 || res[10] != 1 || res[20] != 2 || res[30] != 3 {
		t.Fatalf("unexpected IndexBy result: %v", res)
	}
}

func TestUnique(t *testing.T) {
	if len(Unique[int](nil)) != 0 {
		t.Fatal("expected empty slice")
	}

	in := []int{1, 2, 2, 3, 1, 4}
	res := Unique(in)
	expected := []int{1, 2, 3, 4}

	if len(res) != 4 {
		t.Fatalf("expected length 4, got %d", len(res))
	}

	for i, v := range res {
		if v != expected[i] {
			t.Errorf("expected %d at %d, got %d", expected[i], i, v)
		}
	}
}

func TestGroupBy(t *testing.T) {
	if GroupBy[string, int](nil, func(x int) string { return "" }) != nil {
		t.Fatal("expected nil for nil slice")
	}

	in := []int{1, 2, 3, 4, 5}
	res := GroupBy(in, func(x int) string {
		if x%2 == 0 {
			return "even"
		}

		return "odd"
	})

	if len(res["even"]) != 2 || len(res["odd"]) != 3 {
		t.Fatalf("unexpected GroupBy: %v", res)
	}
}

func TestAnyAllFind(t *testing.T) {
	in := []int{1, 2, 3, 4}

	if !Any(in, func(x int) bool { return x == 3 }) {
		t.Error("expected Any to return true")
	}

	if Any(in, func(x int) bool { return x == 5 }) {
		t.Error("expected Any to return false")
	}

	if !All(in, func(x int) bool { return x > 0 }) {
		t.Error("expected All to return true")
	}

	if All(in, func(x int) bool { return x < 4 }) {
		t.Error("expected All to return false")
	}

	val, found := Find(in, func(x int) bool { return x%3 == 0 })
	if !found || val != 3 {
		t.Errorf("expected (3, true), got (%d, %t)", val, found)
	}

	val2, found2 := Find(in, func(x int) bool { return x == 10 })
	if found2 || val2 != 0 {
		t.Errorf("expected (0, false), got (%d, %t)", val2, found2)
	}
}

func TestFilterInPlace(t *testing.T) {
	in := []int{1, 2, 3, 4, 5}
	res := FilterInPlace(in, func(x int) bool {
		return x%2 == 0
	})

	if len(res) != 2 || res[0] != 2 || res[1] != 4 {
		t.Fatalf("unexpected FilterInPlace output: %v", res)
	}

	// Verify clear() zeroed out the tail of the backing array
	backing := in[:5]
	if backing[2] != 0 || backing[3] != 0 || backing[4] != 0 {
		t.Errorf("expected backing array tail to be cleared, got %v", backing)
	}
}

func TestApplyOptions(t *testing.T) {
	type config struct {
		name string
		val  int
	}

	var cfg config

	optName := func(c *config) { c.name = "opt" }
	optVal := func(c *config) { c.val = 42 }

	ApplyOptions(&cfg, optName, nil, optVal)

	if cfg.name != "opt" || cfg.val != 42 {
		t.Errorf("expected options to be applied, got %v", cfg)
	}
}

func TestPtr(t *testing.T) {
	v := 42

	p := new(v)
	if p == nil || *p != 42 {
		t.Fatalf("expected pointer to 42, got %v", p)
	}
}

func TestPtrOrNil(t *testing.T) {
	p1 := PtrOrNil(0)
	if p1 != nil {
		t.Fatal("expected nil pointer for zero value")
	}

	p2 := PtrOrNil(42)
	if p2 == nil || *p2 != 42 {
		t.Fatalf("expected pointer to 42, got %v", p2)
	}
}

func TestZeroIsZero(t *testing.T) {
	z := Zero[int]()
	if z != 0 {
		t.Errorf("expected 0, got %d", z)
	}

	if !IsZero(0) {
		t.Error("expected 0 to be zero")
	}

	if IsZero(42) {
		t.Error("expected 42 to not be zero")
	}
}

func TestDeref(t *testing.T) {
	if Deref[int](nil) != 0 {
		t.Error("expected 0 for nil pointer")
	}

	val := 42
	if Deref(&val) != 42 {
		t.Error("expected 42")
	}
}

func TestDerefOr(t *testing.T) {
	if DerefOr[int](nil, 100) != 100 {
		t.Error("expected default value 100 for nil pointer")
	}

	val := 42
	if DerefOr(&val, 100) != 42 {
		t.Error("expected 42, not default 100")
	}
}

func TestCoalesce(t *testing.T) {
	if Coalesce[int](0, 0, 0) != 0 {
		t.Error("expected zero value")
	}

	if Coalesce(0, 42, 100) != 42 {
		t.Error("expected first non-zero value 42")
	}
}

func TestTernary(t *testing.T) {
	if Ternary(true, "yes", "no") != "yes" {
		t.Error("expected 'yes'")
	}

	if Ternary(false, "yes", "no") != "no" {
		t.Error("expected 'no'")
	}
}

func TestKeysAndValues(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2}
	keys := Keys(m)
	values := Values(m)

	if len(keys) != 2 || len(values) != 2 {
		t.Errorf("expected 2 keys and 2 values, got %v, %v", keys, values)
	}

	// Chunked edge cases
	if Chunked([]int{1, 2, 3}, 0) != nil {
		t.Error("expected nil for Chunked with size <= 0")
	}
	chunks := Chunked([]int{1, 2, 3, 4, 5}, 2)
	if len(chunks) != 3 || len(chunks[0]) != 2 || len(chunks[2]) != 1 {
		t.Errorf("unexpected chunks: %v", chunks)
	}
}

func TestCoalesceNil(t *testing.T) {
	var a *int
	b := new(10)
	c := new(20)

	res := CoalesceNil(a, b, c)
	if res == nil || *res != 10 {
		t.Errorf("expected *res=10, got %v", res)
	}

	var x, y *int
	if CoalesceNil(x, y) != nil {
		t.Error("expected nil for all nil inputs")
	}
}

func TestShardedMap_All(t *testing.T) {
	sm := NewShardedMap[uint64, int]()
	sm.Set(1, 10)
	sm.Set(2, 20)
	sm.Set(3, 30)

	all := sm.All()
	if len(all) != 3 {
		t.Errorf("expected 3 items, got %d", len(all))
	}

	sm.Delete(2)
	all = sm.All()
	if len(all) != 2 {
		t.Errorf("expected 2 items after delete, got %d", len(all))
	}

	var nilSm *ShardedMap[uint64, int]
	if nilSm.All() != nil {
		t.Error("expected nil for nil ShardedMap.All")
	}
	if _, ok := nilSm.Get(1); ok {
		t.Error("expected false for nil ShardedMap.Get")
	}
	nilSm.Set(1, 10)
	nilSm.Delete(1)
}

func TestFuture_ContextAndTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	f := NewFutureContext[string](ctx, func(c context.Context) (string, error) {
		<-c.Done()
		return "", c.Err()
	})

	_, err := f.Get(context.Background())
	if err == nil {
		t.Error("expected context error on timeout")
	}

	// Double set (idempotent)
	fManual := NewFuture[int]()
	fManual.Set(10, nil)
	fManual.Set(20, nil)

	val, err := fManual.Get(context.Background())
	if err != nil || val != 10 {
		t.Errorf("expected val=10, got val=%d, err=%v", val, err)
	}

	var nilFuture *Future[int]
	nilFuture.Set(1, nil)
	_, err = nilFuture.Get(context.Background())
	if err == nil {
		t.Error("expected error for nil Future.Get")
	}
}

func TestLazy_TakeDropToSeq(t *testing.T) {
	seq := func(yield func(int) bool) {
		for i := 1; i <= 5; i++ {
			if !yield(i) {
				return
			}
		}
	}

	taken := Take(seq, 2)
	slice := ToSlice(taken)
	if len(slice) != 2 || slice[0] != 1 || slice[1] != 2 {
		t.Errorf("expected [1 2], got %v", slice)
	}

	if len(ToSlice(Take(seq, 0))) != 0 {
		t.Error("expected empty slice for Take <= 0")
	}

	dropped := Drop(seq, 3)
	sliceDropped := ToSlice(dropped)
	if len(sliceDropped) != 2 || sliceDropped[0] != 4 || sliceDropped[1] != 5 {
		t.Errorf("expected [4 5], got %v", sliceDropped)
	}

	if len(ToSlice(Drop(seq, 0))) != 5 {
		t.Error("expected 5 elements for Drop <= 0")
	}

	// Stride step <= 0
	r := NewRange(1, 5)
	r.Stride(0, func(v int) bool { return true })
}

func TestRaceFirstSuccess(t *testing.T) {
	t.Run("first_success_wins", func(t *testing.T) {
		task1 := func(ctx context.Context) Result[string] {
			time.Sleep(100 * time.Millisecond)
			return Success("slow")
		}
		task2 := func(ctx context.Context) Result[string] {
			time.Sleep(10 * time.Millisecond)
			return Success("fast")
		}
		task3 := func(ctx context.Context) Result[string] {
			return Failure[string](errors.New("immediate fail"))
		}

		res := RaceFirstSuccess(context.Background(), task1, task2, task3)
		if !res.IsSuccess() {
			t.Fatal("expected success")
		}
		val, _ := res.Unwrap()
		if val != "fast" {
			t.Errorf("expected 'fast', got %q", val)
		}
	})

	t.Run("all_fail", func(t *testing.T) {
		task1 := func(ctx context.Context) Result[int] {
			return Failure[int](errors.New("fail1"))
		}
		task2 := func(ctx context.Context) Result[int] {
			return Failure[int](errors.New("fail2"))
		}

		res := RaceFirstSuccess(context.Background(), task1, task2)
		if res.IsSuccess() {
			t.Fatal("expected failure")
		}
	})

	t.Run("empty_tasks", func(t *testing.T) {
		res := RaceFirstSuccess[int](context.Background())
		if res.IsSuccess() {
			t.Fatal("expected failure on empty tasks")
		}
	})
}

func TestMutexAndLockGuards(t *testing.T) {
	t.Parallel()

	m := NewMutex(10)
	if m.Get() != 10 {
		t.Fatalf("expected 10, got %d", m.Get())
	}

	m.Set(20)
	if m.Get() != 20 {
		t.Fatalf("expected 20, got %d", m.Get())
	}

	old := m.Swap(30)
	if old != 20 || m.Get() != 30 {
		t.Fatalf("expected old 20, current 30, got old %d, current %d", old, m.Get())
	}

	res := m.Update(func(curr int) int { return curr + 5 })
	if res != 35 || m.Get() != 35 {
		t.Fatalf("expected 35, got %d", res)
	}

	m.With(func(val *int) {
		*val = 100
	})
	if m.Get() != 100 {
		t.Fatalf("expected 100, got %d", m.Get())
	}

	if !m.TryWith(func(val *int) { *val = 200 }) || m.Get() != 200 {
		t.Fatal("expected TryWith to succeed")
	}

	var mu sync.Mutex
	val := WithGuard(&mu, func() int { return 42 })
	if val != 42 {
		t.Fatalf("expected 42, got %d", val)
	}

	var rw sync.RWMutex
	rVal := WithRGuard(&rw, func() string { return "apple" })
	if rVal != "apple" {
		t.Fatalf("expected apple, got %s", rVal)
	}
}

func TestFindAndFindOptional(t *testing.T) {
	items := []int{10, 20, 30, 40}

	// Find
	v, ok := Find(items, func(x int) bool { return x == 30 })
	if !ok || v != 30 {
		t.Fatalf("expected to find 30, got %d (ok=%v)", v, ok)
	}

	_, ok = Find(items, func(x int) bool { return x == 99 })
	if ok {
		t.Fatal("expected not found for 99")
	}

	// FindOptional
	opt := FindOptional(items, func(x int) bool { return x == 20 })
	if !opt.IsPresent() {
		t.Fatal("expected 20 to be present in Optional")
	}
	if val, ok := opt.Value(); !ok || val != 20 {
		t.Fatalf("expected 20, got %d", val)
	}

	optNone := FindOptional(items, func(x int) bool { return x == 99 })
	if optNone.IsPresent() {
		t.Fatal("expected None for 99")
	}

	// Nil predicate
	optNilPred := FindOptional(items, nil)
	if optNilPred.IsPresent() {
		t.Fatal("expected None for nil predicate")
	}
}
// TestAdversarial_BatchLoader_MassiveKeyDeduplication_Stress verifies that BatchLoader
// deduplicates batch keys and unblocks all waiters under heavy concurrency.
func TestAdversarial_BatchLoader_MassiveKeyDeduplication_Stress(t *testing.T) {
	var (
		batchCount atomic.Int32
		totalKeys  atomic.Int32
	)

	dl := NewBatchLoader[string, int](
		15*time.Millisecond,
		func(ctx context.Context, keys []string) (map[string]int, error) {
			batchCount.Add(1)
			totalKeys.Add(int32(len(keys)))

			res := make(map[string]int, len(keys))
			for _, k := range keys {
				res[k] = len(k) * 10
			}
			return res, nil
		},
	)

	const numCallers = 200
	var wg sync.WaitGroup
	wg.Add(numCallers)

	errs := make([]error, numCallers)
	results := make([]int, numCallers)

	// Callers only request one of 5 keys
	targetKeys := []string{"alpha", "bravo", "charlie", "delta", "echo"}

	for i := range numCallers {
		idx := i
		k := targetKeys[i%len(targetKeys)]
		go func(key string) {
			defer wg.Done()
			results[idx], errs[idx] = dl.Load(context.Background(), key)
		}(k)
	}

	wg.Wait()

	for i := range numCallers {
		require.NoError(t, errs[i])
		k := targetKeys[i%len(targetKeys)]
		assert.Equal(t, len(k)*10, results[i])
	}

	// Because keys were deduplicated, total keys batched must be significantly less than numCallers
	assert.LessOrEqual(t, int(totalKeys.Load()), len(targetKeys)*int(batchCount.Load()))
}

// TestAdversarial_BatchLoader_CancellationStorm_Deregistration tests high-concurrency
// context cancellations racing against batch executions.
func TestAdversarial_BatchLoader_CancellationStorm_Deregistration(t *testing.T) {
	var batchExecutions atomic.Int32

	dl := NewBatchLoader[int, int](
		25*time.Millisecond,
		func(ctx context.Context, keys []int) (map[int]int, error) {
			batchExecutions.Add(1)
			time.Sleep(10 * time.Millisecond) // simulate I/O
			res := make(map[int]int, len(keys))
			for _, k := range keys {
				res[k] = k * 100
			}
			return res, nil
		},
	)

	const numCallers = 200
	var wg sync.WaitGroup
	wg.Add(numCallers)

	var (
		cancelledCount atomic.Int32
		succeededCount atomic.Int32
	)

	for i := range numCallers {
		key := i
		go func() {
			defer wg.Done()

			// Half of callers have very short timeouts that expire during the window
			var (
				ctx    context.Context
				cancel context.CancelFunc
			)

			if key%2 == 0 {
				ctx, cancel = context.WithTimeout(context.Background(), time.Duration(2+rand.Intn(15))*time.Millisecond)
			} else {
				ctx, cancel = context.WithTimeout(context.Background(), 200*time.Millisecond)
			}
			defer cancel()

			val, err := dl.Load(ctx, key)
			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
					cancelledCount.Add(1)
				} else {
					t.Errorf("unexpected error for key %d: %v", key, err)
				}
			} else {
				succeededCount.Add(1)
				if val != key*100 {
					t.Errorf("key %d: expected %d, got %d", key, key*100, val)
				}
			}
		}()
	}

	wg.Wait()

	assert.Equal(t, int32(numCallers), cancelledCount.Load()+succeededCount.Load())
	assert.Greater(t, succeededCount.Load(), int32(0))
}

// TestAdversarial_BatchLoader_ZeroDelay_ContinuousThroughput verifies that
// delay=0 does not deadlock and processes continuous streams of requests.
func TestAdversarial_BatchLoader_ZeroDelay_ContinuousThroughput(t *testing.T) {
	dl := NewBatchLoader[int, int](
		0, // zero delay window: immediate scheduling
		func(ctx context.Context, keys []int) (map[int]int, error) {
			res := make(map[int]int, len(keys))
			for _, k := range keys {
				res[k] = k + 1
			}
			return res, nil
		},
	)

	const (
		numWorkers = 20
		opsWorker  = 50
	)

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for w := range numWorkers {
		go func(workerID int) {
			defer wg.Done()
			for i := range opsWorker {
				key := workerID*1000 + i
				val, err := dl.Load(context.Background(), key)
				if err != nil {
					t.Errorf("key %d returned error: %v", key, err)
					return
				}
				if val != key+1 {
					t.Errorf("key %d: expected %d, got %d", key, key+1, val)
					return
				}
			}
		}(w)
	}

	wg.Wait()
}

// TestAdversarial_BatchLoader_IntermittentPanic_Recovery verifies that callers
// unblock and BatchLoader recovers when batchFn panics intermittently.
func TestAdversarial_BatchLoader_IntermittentPanic_Recovery(t *testing.T) {
	var batchExec atomic.Int32

	dl := NewBatchLoader[int, int](
		10*time.Millisecond,
		func(ctx context.Context, keys []int) (map[int]int, error) {
			execNum := batchExec.Add(1)
			if execNum%2 == 1 {
				panic(fmt.Sprintf("deliberate intermittent panic on batch #%d", execNum))
			}
			res := make(map[int]int, len(keys))
			for _, k := range keys {
				res[k] = k * 2
			}
			return res, nil
		},
	)

	// Batch 1: will panic
	var wg1 sync.WaitGroup
	const batch1Callers = 10
	wg1.Add(batch1Callers)
	b1Errs := make([]error, batch1Callers)

	for i := range batch1Callers {
		idx := i
		go func() {
			defer wg1.Done()
			_, b1Errs[idx] = dl.Load(context.Background(), idx)
		}()
	}
	wg1.Wait()

	for i, err := range b1Errs {
		if !errors.Is(err, ErrBatchPanicked) {
			t.Fatalf("caller %d expected ErrBatchPanicked, got: %v", i, err)
		}
	}

	// Batch 2: must succeed normally on the same instance
	var wg2 sync.WaitGroup
	const batch2Callers = 10
	wg2.Add(batch2Callers)
	b2Errs := make([]error, batch2Callers)
	b2Vals := make([]int, batch2Callers)

	for i := range batch2Callers {
		idx := i
		go func() {
			defer wg2.Done()
			b2Vals[idx], b2Errs[idx] = dl.Load(context.Background(), idx)
		}()
	}
	wg2.Wait()

	for i := range batch2Callers {
		if b2Errs[i] != nil {
			t.Fatalf("caller %d in batch 2 expected success, got error: %v", i, b2Errs[i])
		}
		if b2Vals[i] != i*2 {
			t.Fatalf("caller %d in batch 2: expected %d, got %d", i, i*2, b2Vals[i])
		}
	}
}
// 1. Stress Test: Highly concurrent readers, writers, removers, and clearers under race detector
func TestCache_AdversarialConcurrentStress(t *testing.T) {
	c := NewCacheWithJanitor[string, int](5 * time.Millisecond)
	defer c.Close()

	const duration = 2 * time.Second
	stopCh := make(chan struct{})
	time.AfterFunc(duration, func() { close(stopCh) })

	var (
		wg        sync.WaitGroup
		opsRead   atomic.Int64
		opsWrite  atomic.Int64
		opsDelete atomic.Int64
		opsPurge  atomic.Int64
		opsLen    atomic.Int64
		opsClear  atomic.Int64
	)

	const numKeys = 200

	// 20 Readers
	for r := range 20 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(id * 1000)))
			for {
				select {
				case <-stopCh:
					return
				default:
					key := fmt.Sprintf("key_%d", rng.Intn(numKeys))
					_, _ = c.Get(key)
					opsRead.Add(1)
				}
			}
		}(r)
	}

	// 20 Writers with mixed TTLs
	for w := range 20 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(id * 2000)))
			for {
				select {
				case <-stopCh:
					return
				default:
					key := fmt.Sprintf("key_%d", rng.Intn(numKeys))
					val := rng.Int()
					var ttl time.Duration
					switch rng.Intn(5) {
					case 0:
						ttl = 0 // immediate expiration
					case 1:
						ttl = -10 * time.Millisecond // past
					case 2:
						ttl = time.Duration(rng.Intn(10)+1) * time.Millisecond // short
					case 3:
						ttl = 50 * time.Millisecond // medium
					case 4:
						ttl = 10 * time.Second // long
					}
					c.Set(key, val, ttl)
					opsWrite.Add(1)
				}
			}
		}(w)
	}

	// 10 Deleters
	for d := range 10 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(id * 3000)))
			for {
				select {
				case <-stopCh:
					return
				default:
					key := fmt.Sprintf("key_%d", rng.Intn(numKeys))
					_ = c.Delete(key)
					opsDelete.Add(1)
				}
			}
		}(d)
	}

	// 5 Purgers
	for range 5 {
		wg.Go(func() {
			for {
				select {
				case <-stopCh:
					return
				default:
					_ = c.PurgeExpired()
					opsPurge.Add(1)
					time.Sleep(10 * time.Millisecond)
				}
			}
		})
	}

	// 5 Len checkers
	for range 5 {
		wg.Go(func() {
			for {
				select {
				case <-stopCh:
					return
				default:
					_ = c.Len()
					opsLen.Add(1)
					time.Sleep(10 * time.Millisecond)
				}
			}
		})
	}

	// 2 Clearers
	for range 2 {
		wg.Go(func() {
			for {
				select {
				case <-stopCh:
					return
				default:
					time.Sleep(200 * time.Millisecond)
					c.Clear()
					opsClear.Add(1)
				}
			}
		})
	}

	wg.Wait()

	t.Logf("Stress test complete. Ops: Read=%d, Write=%d, Delete=%d, Purge=%d, Len=%d, Clear=%d",
		opsRead.Load(), opsWrite.Load(), opsDelete.Load(), opsPurge.Load(), opsLen.Load(), opsClear.Load())

	if opsRead.Load() == 0 || opsWrite.Load() == 0 {
		t.Fatal("stress test did not execute sufficient operations")
	}
}

// 2. Janitor Goroutine Leak Verification: Explicit Close and Garbage Collection Finalizer
func TestCache_JanitorGoroutineLeak(t *testing.T) {
	// Step A: Verify explicit Close terminates janitor goroutines immediately
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	initialGoroutines := runtime.NumGoroutine()

	const count = 100
	caches := make([]*Cache[int, int], count)
	for i := range count {
		caches[i] = NewCacheWithJanitor[int, int](10 * time.Millisecond)
	}

	afterCreateGoroutines := runtime.NumGoroutine()
	if afterCreateGoroutines < initialGoroutines+count-5 {
		t.Fatalf("expected at least %d goroutines, got %d (initial: %d)",
			initialGoroutines+count, afterCreateGoroutines, initialGoroutines)
	}

	// Explicitly close all
	for i := range count {
		caches[i].Close()
	}

	// Wait for goroutines to exit
	time.Sleep(50 * time.Millisecond)
	afterCloseGoroutines := runtime.NumGoroutine()
	if afterCloseGoroutines > initialGoroutines+5 {
		t.Fatalf("goroutines leaked after Close(): initial=%d, afterClose=%d",
			initialGoroutines, afterCloseGoroutines)
	}

	// Step B: Verify finalizer terminates janitors when caches are abandoned without Close()
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	baseGoroutines := runtime.NumGoroutine()

	func() {
		localCaches := make([]*Cache[int, int], count)
		for i := range count {
			localCaches[i] = NewCacheWithJanitor[int, int](10 * time.Millisecond)
			localCaches[i].Set(i, i, time.Hour)
		}
		// localCaches goes out of scope here
		_ = localCaches
	}()

	// Force GC and wait for finalizers
	for range 5 {
		runtime.GC()
		time.Sleep(50 * time.Millisecond)
		if runtime.NumGoroutine() <= baseGoroutines+5 {
			break
		}
	}

	finalGoroutines := runtime.NumGoroutine()
	if finalGoroutines > baseGoroutines+5 {
		t.Fatalf("goroutines leaked after GC finalization: base=%d, final=%d",
			baseGoroutines, finalGoroutines)
	}

	// Step C: Concurrent Close() calls must not panic
	c := NewCacheWithJanitor[string, string](10 * time.Millisecond)
	var closeWg sync.WaitGroup
	for range 50 {
		closeWg.Go(func() {
			c.Close()
		})
	}
	closeWg.Wait()
}

// 3. Key Churn and Memory Reclamation Verification
func TestCache_KeyChurnMemoryReclamation(t *testing.T) {
	c := NewCacheWithJanitor[string, []byte](10 * time.Millisecond)
	defer c.Close()

	// 100 persistent items
	for i := range 100 {
		c.Set(fmt.Sprintf("perm_%d", i), make([]byte, 1024), 1*time.Hour)
	}

	// Measure baseline memory
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Churn 100,000 unique keys with 5ms TTL
	const churnTotal = 100000
	payload := make([]byte, 256)

	for i := range churnTotal {
		key := fmt.Sprintf("churn_%d", i)
		c.Set(key, payload, 5*time.Millisecond)
		if i%10000 == 0 {
			time.Sleep(10 * time.Millisecond)
			c.PurgeExpired()
		}
	}

	// Allow remaining churn keys to expire and purge
	time.Sleep(30 * time.Millisecond)
	c.PurgeExpired()

	// Verify only the 100 persistent items remain
	if c.Len() != 100 {
		t.Fatalf("expected exactly 100 items remaining, got %d", c.Len())
	}

	// Measure memory after churn and GC
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	t.Logf("HeapAlloc before churn: %d KB, after churn: %d KB, diff: %d KB",
		m1.HeapAlloc/1024, m2.HeapAlloc/1024, int64(m2.HeapAlloc-m1.HeapAlloc)/1024)

	// Verify that heap does not retain 100,000 payload objects
	// 100,000 * 256 bytes payload = ~25MB. Heap growth should be well below 15MB.
	heapDiffKB := int64(m2.HeapAlloc-m1.HeapAlloc) / 1024
	if heapDiffKB > 20000 {
		t.Fatalf("excessive heap growth after churn: %d KB", heapDiffKB)
	}
}

// 4. Eviction Accuracy and Sub-millisecond Expiration
func TestCache_EvictionAccuracyAdversarial(t *testing.T) {
	c := NewCacheWithJanitor[string, int](0) // disable janitor for exact timing
	defer c.Close()

	// Boundary 1: Immediate and negative TTL
	c.Set("zero_ttl", 100, 0)
	c.Set("negative_ttl", 200, -1*time.Second)

	if val, ok := c.Get("zero_ttl"); ok || val != 0 {
		t.Fatalf("expected immediate miss on zero TTL, got val=%v, ok=%t", val, ok)
	}
	if val, ok := c.Get("negative_ttl"); ok || val != 0 {
		t.Fatalf("expected immediate miss on negative TTL, got val=%v, ok=%t", val, ok)
	}

	// Boundary 2: Nanosecond accuracy
	c.Set("fast", 1, 15*time.Millisecond)
	c.Set("slow", 2, 80*time.Millisecond)

	// Immediately: both present
	if v, ok := c.Get("fast"); !ok || v != 1 {
		t.Fatalf("expected fast present immediately")
	}
	if v, ok := c.Get("slow"); !ok || v != 2 {
		t.Fatalf("expected slow present immediately")
	}

	// Sleep past fast TTL but before slow TTL
	time.Sleep(30 * time.Millisecond)

	// fast must miss, slow must hit
	if v, ok := c.Get("fast"); ok {
		t.Fatalf("expected fast expired, got %v", v)
	}
	if v, ok := c.Get("slow"); !ok || v != 2 {
		t.Fatalf("expected slow still active, got %v (ok=%t)", v, ok)
	}

	// Boundary 3: Delete return value
	// Delete on active item returns true
	if !c.Delete("slow") {
		t.Fatal("expected Delete on active slow key to return true")
	}
	// Delete on already deleted item returns false
	if c.Delete("slow") {
		t.Fatal("expected Delete on already deleted key to return false")
	}

	// Boundary 4: PurgeExpired accuracy
	c.Set("p1", 1, 10*time.Millisecond)
	c.Set("p2", 2, 10*time.Millisecond)
	c.Set("p3", 3, 200*time.Millisecond)
	c.Set("p4", 4, 200*time.Millisecond)

	time.Sleep(25 * time.Millisecond)

	purged := c.PurgeExpired()
	if purged != 2 {
		t.Fatalf("expected exactly 2 purged, got %d", purged)
	}
	if c.Len() != 2 {
		t.Fatalf("expected Len() == 2, got %d", c.Len())
	}
	if _, ok := c.Get("p3"); !ok {
		t.Fatal("expected p3 to survive purge")
	}
	if _, ok := c.Get("p4"); !ok {
		t.Fatal("expected p4 to survive purge")
	}
}

// 5. Zero-value struct handling
func TestCache_ZeroValueHandling(t *testing.T) {
	var zero Cache[string, int]

	// Read operations on zero value Cache
	if _, ok := zero.Get("missing"); ok {
		t.Fatal("expected false from zero-value Get")
	}
	if zero.Len() != 0 {
		t.Fatal("expected 0 from zero-value Len")
	}
	if zero.Delete("missing") {
		t.Fatal("expected false from zero-value Delete")
	}
	if zero.PurgeExpired() != 0 {
		t.Fatal("expected 0 from zero-value PurgeExpired")
	}
	zero.Clear()
	zero.Close()

	// Initialized via Set
	zero.Set("k", 42, time.Hour)
	if val, ok := zero.Get("k"); !ok || val != 42 {
		t.Fatalf("expected 42 after lazy init via Set, got %v, ok=%t", val, ok)
	}
}

// 6. Benchmark concurrent read/write throughput
func BenchmarkCache_Concurrent90Read10Write(b *testing.B) {
	c := NewCacheWithJanitor[int, int](50 * time.Millisecond)
	defer c.Close()

	for i := range 1000 {
		c.Set(i, i, 5*time.Second)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		for pb.Next() {
			k := rng.Intn(1000)
			if rng.Intn(10) == 0 {
				c.Set(k, k, 5*time.Second)
			} else {
				_, _ = c.Get(k)
			}
		}
	})
}
// TestAdversarial_Guarded_ExtremeContention_RMW verifies that Guarded protects
// concurrent read/write and Read-Modify-Write invariants under high contention.
func TestAdversarial_Guarded_ExtremeContention_RMW(t *testing.T) {
	g := NewGuarded[int](0)

	const (
		numUpdaters = 30
		opsUpdater  = 500

		numReaders = 30
		opsReader  = 500
	)

	var wg sync.WaitGroup
	wg.Add(numUpdaters + numReaders)

	// Updaters: atomic increments via Update
	for range numUpdaters {
		go func() {
			defer wg.Done()
			for range opsUpdater {
				g.Update(func(current int) int {
					return current + 1
				})
			}
		}()
	}

	// Readers: concurrent Get and Read
	for range numReaders {
		go func() {
			defer wg.Done()
			for range opsReader {
				val := g.Get()
				_ = val

				g.Read(func(v int) {
					if v < 0 {
						t.Errorf("read negative value: %d", v)
					}
				})
			}
		}()
	}

	wg.Wait()

	expected := numUpdaters * opsUpdater
	finalVal := g.Get()
	if finalVal != expected {
		t.Fatalf("expected final value %d, got %d (lost updates detected!)", expected, finalVal)
	}
}

// TestAdversarial_Guarded_MapProtection_NoPanic verifies that Guarded completely
// prevents concurrent map read and map write crashes under heavy contention.
func TestAdversarial_Guarded_MapProtection_NoPanic(t *testing.T) {
	initialMap := make(map[int]int)
	g := NewGuarded[map[int]int](initialMap)

	const (
		numMutators = 25
		opsMutator  = 300

		numReaders = 25
		opsReader  = 300
	)

	var wg sync.WaitGroup
	wg.Add(numMutators + numReaders)

	// Mutators
	for m := range numMutators {
		go func(mutatorID int) {
			defer wg.Done()
			for i := range opsMutator {
				key := (mutatorID*100 + i) % 50
				g.Mutate(func(mPtr *map[int]int) {
					(*mPtr)[key] = i
				})
			}
		}(m)
	}

	// Readers
	for r := range numReaders {
		go func(readerID int) {
			defer wg.Done()
			for range opsReader {
				key := rand.Intn(50)
				g.Read(func(m map[int]int) {
					_ = m[key]
				})
			}
		}(r)
	}

	wg.Wait()

	// Verify map remains valid
	g.Read(func(m map[int]int) {
		if len(m) == 0 {
			t.Errorf("expected populated map, got 0 keys")
		}
	})
}

// TestAdversarial_Guarded_SwapAtomicity verifies that Swap operations yield strictly
// unique previous values under concurrent execution.
func TestAdversarial_Guarded_SwapAtomicity(t *testing.T) {
	g := NewGuarded[int64](0)

	const (
		numWorkers = 40
		opsWorker  = 100
	)

	var (
		wg         sync.WaitGroup
		seenValues sync.Map
		duplicates atomic.Int64
		idCounter  atomic.Int64
	)

	wg.Add(numWorkers)

	for range numWorkers {
		go func() {
			defer wg.Done()
			for range opsWorker {
				newVal := idCounter.Add(1)
				oldVal := g.Swap(newVal)

				if oldVal != 0 {
					if _, loaded := seenValues.LoadOrStore(oldVal, true); loaded {
						duplicates.Add(1)
					}
				}
			}
		}()
	}

	wg.Wait()

	if duplicates.Load() > 0 {
		t.Fatalf("detected %d duplicate old values returned by Swap (broken atomicity!)", duplicates.Load())
	}
}
type AdversarialChild struct {
	Tag  Optional[string] `json:"tag,omitzero"`
	Code Optional[int]    `json:"code,omitzero"`
}

type AdversarialParent struct {
	ID       Optional[string]           `json:"id,omitzero"`
	Child    Optional[AdversarialChild] `json:"child,omitzero"`
	Flag     Optional[bool]             `json:"flag,omitzero"`
	RawBytes Optional[[]byte]           `json:"bytes,omitzero"`
}

func TestOptional_Adversarial_NilPointerReceiver(t *testing.T) {
	var nilOpt *Optional[string]
	err := nilOpt.UnmarshalJSON([]byte(`"fail"`))
	assert.NotNil(t, err)
	assert.True(t, errors.Is(err, ErrNilOptional))
}

func TestOptional_Adversarial_CorruptedData(t *testing.T) {
	corruptedInputs := [][]byte{
		[]byte(`{`),
		[]byte(`}`),
		[]byte(`{"key":`),
		[]byte(`{"key": "val"`),
		[]byte(`[1, 2,`),
		[]byte(`"unterminated`),
		[]byte(`123a`),
		[]byte(`truefalse`),
		[]byte(`nulll`),
		[]byte(`{"nested": {"broken": `),
		[]byte("\x00\x01\x02\xff"),
		[]byte(`{"tag": 123}`), // Type mismatch: int instead of string
	}

	for i, input := range corruptedInputs {
		t.Run(fmt.Sprintf("Input_%d", i), func(t *testing.T) {
			// Test with Optional[string]
			sOpt := Some("original_string")
			err := json.Unmarshal(input, &sOpt)
			assert.NotNil(t, err)
			assert.True(t, sOpt.IsPresent())
			assert.Equal(t, "original_string", sOpt.MustValue())

			// Test with Optional[int]
			iOpt := Some(9999)
			err = json.Unmarshal(input, &iOpt)
			assert.NotNil(t, err)
			assert.True(t, iOpt.IsPresent())
			assert.Equal(t, 9999, iOpt.MustValue())

			// Test with Optional[AdversarialChild]
			cOpt := Some(AdversarialChild{Tag: Some("safe")})
			err = json.Unmarshal(input, &cOpt)
			assert.NotNil(t, err)
			assert.True(t, cOpt.IsPresent())
			assert.Equal(t, "safe", cOpt.MustValue().Tag.MustValue())
		})
	}
}

func TestOptional_Adversarial_WhitespaceAndEmpty(t *testing.T) {
	emptyInputs := [][]byte{
		nil,
		{},
		[]byte(""),
		[]byte(" "),
		[]byte("\t\t"),
		[]byte("\n\r\n"),
		[]byte("   \t  \n  "),
		[]byte("null"),
		[]byte("  null  "),
		[]byte("\n\tnull\r\n"),
		[]byte("\t  null \t\n"),
	}

	for i, input := range emptyInputs {
		t.Run(fmt.Sprintf("Empty_%d", i), func(t *testing.T) {
			var opt Optional[string]
			opt = Some("should_be_cleared")
			err := opt.UnmarshalJSON(input)
			assert.Nil(t, err)
			assert.False(t, opt.IsPresent())
			v, ok := opt.Value()
			assert.False(t, ok)
			assert.Equal(t, "", v)
			assert.True(t, opt.IsZero())
		})
	}
}

func TestOptional_Adversarial_NestedStructsRoundtrip(t *testing.T) {
	// 1. All fields present
	p1 := AdversarialParent{
		ID: Some("root-101"),
		Child: Some(AdversarialChild{
			Tag:  Some("alpha"),
			Code: Some(404),
		}),
		Flag:     Some(false), // Explicit zero-value boolean
		RawBytes: Some([]byte("binary-payload")),
	}

	data1, err := json.Marshal(p1)
	assert.Nil(t, err)

	var p1Decoded AdversarialParent
	err = json.Unmarshal(data1, &p1Decoded)
	assert.Nil(t, err)
	assert.True(t, p1Decoded.ID.IsPresent())
	assert.Equal(t, "root-101", p1Decoded.ID.MustValue())
	assert.True(t, p1Decoded.Child.IsPresent())
	assert.Equal(t, "alpha", p1Decoded.Child.MustValue().Tag.MustValue())
	assert.Equal(t, 404, p1Decoded.Child.MustValue().Code.MustValue())
	assert.True(t, p1Decoded.Flag.IsPresent())
	assert.Equal(t, false, p1Decoded.Flag.MustValue())
	assert.True(t, p1Decoded.RawBytes.IsPresent())
	assert.Equal(t, []byte("binary-payload"), p1Decoded.RawBytes.MustValue())

	// 2. Inner child partially present
	p2 := AdversarialParent{
		ID: Some("root-202"),
		Child: Some(AdversarialChild{
			Tag:  Some(""), // Explicit empty string
			Code: None[int](),
		}),
		Flag:     None[bool](),
		RawBytes: None[[]byte](),
	}

	data2, err := json.Marshal(p2)
	assert.Nil(t, err)
	// Code, Flag, RawBytes should be omitted due to omitzero!
	assert.Equal(t, `{"id":"root-202","child":{"tag":""}}`, string(data2))

	var p2Decoded AdversarialParent
	err = json.Unmarshal(data2, &p2Decoded)
	assert.Nil(t, err)
	assert.True(t, p2Decoded.ID.IsPresent())
	assert.True(t, p2Decoded.Child.IsPresent())
	assert.True(t, p2Decoded.Child.MustValue().Tag.IsPresent())
	assert.Equal(t, "", p2Decoded.Child.MustValue().Tag.MustValue())
	assert.False(t, p2Decoded.Child.MustValue().Code.IsPresent())
	assert.False(t, p2Decoded.Flag.IsPresent())
	assert.False(t, p2Decoded.RawBytes.IsPresent())

	// 3. All fields None() -> should serialize to {}
	p3 := AdversarialParent{
		ID:       None[string](),
		Child:    None[AdversarialChild](),
		Flag:     None[bool](),
		RawBytes: None[[]byte](),
	}
	data3, err := json.Marshal(p3)
	assert.Nil(t, err)
	assert.Equal(t, `{}`, string(data3))

	var p3Decoded AdversarialParent
	// Initialize with Some values to verify unmarshaling into empty JSON clears/keeps fields
	p3Decoded = AdversarialParent{ID: Some("preset")}
	err = json.Unmarshal(data3, &p3Decoded)
	assert.Nil(t, err)
	// Note: in standard Go json.Unmarshal into existing struct, missing keys in JSON do not overwrite existing fields.
	assert.Equal(t, "preset", p3Decoded.ID.MustValue())
}

func TestOptional_Adversarial_NestedOptional(t *testing.T) {
	// Optional[Optional[string]]
	nestedSome := Some(Some("inner_value"))
	data, err := json.Marshal(nestedSome)
	assert.Nil(t, err)
	assert.Equal(t, `"inner_value"`, string(data))

	var decoded Optional[Optional[string]]
	err = json.Unmarshal(data, &decoded)
	assert.Nil(t, err)
	assert.True(t, decoded.IsPresent())
	assert.True(t, decoded.MustValue().IsPresent())
	assert.Equal(t, "inner_value", decoded.MustValue().MustValue())

	// Optional[Optional[string]] where outer is Some(None())
	nestedSomeNone := Some(None[string]())
	data, err = json.Marshal(nestedSomeNone)
	assert.Nil(t, err)
	assert.Equal(t, `null`, string(data))

	// When outer is None
	nestedNone := None[Optional[string]]()
	data, err = json.Marshal(nestedNone)
	assert.Nil(t, err)
	assert.Equal(t, `null`, string(data))
}

func TestOptional_Adversarial_OmitZeroExhaustive(t *testing.T) {
	type BoundaryModel struct {
		EmptyStr  Optional[string]         `json:"empty_str,omitzero"`
		ZeroInt   Optional[int]            `json:"zero_int,omitzero"`
		ZeroFloat Optional[float64]        `json:"zero_float,omitzero"`
		FalseBool Optional[bool]           `json:"false_bool,omitzero"`
		ZeroTime  Optional[time.Time]      `json:"zero_time,omitzero"`
		EmptyMap  Optional[map[string]int] `json:"empty_map,omitzero"`
		EmptyList Optional[[]string]       `json:"empty_list,omitzero"`
		NilOpt    Optional[*int]           `json:"nil_opt,omitzero"`
	}

	// 1. All explicitly Some(zero-value)
	bm := BoundaryModel{
		EmptyStr:  Some(""),
		ZeroInt:   Some(0),
		ZeroFloat: Some(0.0),
		FalseBool: Some(false),
		ZeroTime:  Some(time.Time{}),
		EmptyMap:  Some(map[string]int{}),
		EmptyList: Some([]string{}),
		NilOpt:    Some[*int](nil),
	}

	data, err := json.Marshal(bm)
	assert.Nil(t, err)

	// Since all are Some(...), IsZero() returns false, meaning none are omitted!
	var rawMap map[string]any
	err = json.Unmarshal(data, &rawMap)
	assert.Nil(t, err)

	assert.True(t, rawMap["empty_str"] != nil)
	assert.Equal(t, "", rawMap["empty_str"])

	assert.True(t, rawMap["zero_int"] != nil)
	assert.Equal(t, float64(0), rawMap["zero_int"])

	assert.True(t, rawMap["zero_float"] != nil)
	assert.Equal(t, float64(0), rawMap["zero_float"])

	assert.True(t, rawMap["false_bool"] != nil)
	assert.Equal(t, false, rawMap["false_bool"])

	// 2. All explicitly None()
	bmNone := BoundaryModel{}
	dataNone, err := json.Marshal(bmNone)
	assert.Nil(t, err)
	assert.Equal(t, `{}`, string(dataNone))
}

func TestOptional_Adversarial_ConcurrencyStress(t *testing.T) {
	const goroutines = 100
	const iterations = 500

	sharedValue := Some("thread_safe_shared_payload")
	var wg sync.WaitGroup

	// 1. Concurrent readers of the same Optional instance
	for range goroutines {
		wg.Go(func() {
			for range iterations {
				assert.True(t, sharedValue.IsPresent())
				assert.Equal(t, "thread_safe_shared_payload", sharedValue.MustValue())
				assert.False(t, sharedValue.IsZero())

				b, err := sharedValue.MarshalJSON()
				assert.Nil(t, err)
				assert.True(t, bytes.Equal(b, []byte(`"thread_safe_shared_payload"`)))
			}
		})
	}

	// 2. Concurrent workers unmarshaling into independent Optional instances
	for g := range goroutines {
		wg.Add(1)
		go func(gid int) {
			defer wg.Done()
			for i := range iterations {
				var opt Optional[int]
				payload := fmt.Sprintf("%d", gid*1000+i)
				err := json.Unmarshal([]byte(payload), &opt)
				assert.Nil(t, err)
				assert.True(t, opt.IsPresent())
				assert.Equal(t, gid*1000+i, opt.MustValue())

				// Reset with null
				err = json.Unmarshal([]byte("null"), &opt)
				assert.Nil(t, err)
				assert.False(t, opt.IsPresent())
				assert.True(t, opt.IsZero())
			}
		}(g)
	}

	wg.Wait()
}

func TestOptional_Adversarial_PrimitiveBoundariesRoundtrip(t *testing.T) {
	// 1. Extreme Int64 Boundaries
	int64Cases := []int64{
		math.MinInt64,
		-9223372036854775807,
		-1000000000,
		-1,
		0,
		1,
		1000000000,
		math.MaxInt64,
	}
	for _, val := range int64Cases {
		opt := Some(val)
		b, err := json.Marshal(opt)
		assert.Nil(t, err)
		assert.Equal(t, fmt.Sprintf("%d", val), string(b))

		var decoded Optional[int64]
		err = json.Unmarshal(b, &decoded)
		assert.Nil(t, err)
		assert.True(t, decoded.IsPresent())
		assert.Equal(t, val, decoded.MustValue())
		assert.False(t, decoded.IsZero())
	}

	// 2. Extreme Uint64 Boundaries
	uint64Cases := []uint64{
		0,
		1,
		math.MaxUint32,
		math.MaxUint64,
	}
	for _, val := range uint64Cases {
		opt := Some(val)
		b, err := json.Marshal(opt)
		assert.Nil(t, err)
		assert.Equal(t, fmt.Sprintf("%d", val), string(b))

		var decoded Optional[uint64]
		err = json.Unmarshal(b, &decoded)
		assert.Nil(t, err)
		assert.True(t, decoded.IsPresent())
		assert.Equal(t, val, decoded.MustValue())
		assert.False(t, decoded.IsZero())
	}

	// 3. Float64 Extreme Boundaries
	float64Cases := []float64{
		-12345.6789,
		-1.0,
		-0.0,
		0.0,
		1.0,
		3.141592653589793,
		math.MaxFloat64,
		math.SmallestNonzeroFloat64,
	}
	for _, val := range float64Cases {
		opt := Some(val)
		b, err := json.Marshal(opt)
		assert.Nil(t, err)

		var decoded Optional[float64]
		err = json.Unmarshal(b, &decoded)
		assert.Nil(t, err)
		assert.True(t, decoded.IsPresent())
		assert.Equal(t, val, decoded.MustValue())
		assert.False(t, decoded.IsZero())
	}

	// 4. Unicode, Emojis, and Escape Strings
	strCases := []string{
		"",
		"simple",
		"spaces in between",
		"alpha & beta = gamma ? # 100% / test+case",
		"日本語のクエリ",
		"Привет мир",
		"Größe Überprüfung",
		"مرحبا بالعالم",
		"🚀🔥💻🎉",
		"line1\nline2\r\nline3\ttab",
		"\"quoted\" and \\backslash\\",
	}
	for _, val := range strCases {
		opt := Some(val)
		b, err := json.Marshal(opt)
		assert.Nil(t, err)

		var decoded Optional[string]
		err = json.Unmarshal(b, &decoded)
		assert.Nil(t, err)
		assert.True(t, decoded.IsPresent())
		assert.Equal(t, val, decoded.MustValue())
		assert.False(t, decoded.IsZero())
	}

	// 5. Booleans
	for _, val := range []bool{true, false} {
		opt := Some(val)
		b, err := json.Marshal(opt)
		assert.Nil(t, err)
		assert.Equal(t, fmt.Sprintf("%t", val), string(b))

		var decoded Optional[bool]
		err = json.Unmarshal(b, &decoded)
		assert.Nil(t, err)
		assert.True(t, decoded.IsPresent())
		assert.Equal(t, val, decoded.MustValue())
		assert.False(t, decoded.IsZero())
	}
}

func TestOptional_Adversarial_CollectionsAndPointersRoundtrip(t *testing.T) {
	// 1. Slices
	optSlicePopulated := Some([]int{10, -20, 30})
	b, err := json.Marshal(optSlicePopulated)
	assert.Nil(t, err)
	assert.Equal(t, "[10,-20,30]", string(b))

	var decodedSlice Optional[[]int]
	err = json.Unmarshal(b, &decodedSlice)
	assert.Nil(t, err)
	assert.True(t, decodedSlice.IsPresent())
	assert.Equal(t, []int{10, -20, 30}, decodedSlice.MustValue())
	assert.False(t, decodedSlice.IsZero())

	// Empty slice
	optSliceEmpty := Some([]int{})
	b, err = json.Marshal(optSliceEmpty)
	assert.Nil(t, err)
	assert.Equal(t, "[]", string(b))

	var decodedEmptySlice Optional[[]int]
	err = json.Unmarshal(b, &decodedEmptySlice)
	assert.Nil(t, err)
	assert.True(t, decodedEmptySlice.IsPresent())
	assert.Equal(t, 0, len(decodedEmptySlice.MustValue()))
	assert.False(t, decodedEmptySlice.IsZero())

	// None slice
	optSliceNone := None[[]int]()
	b, err = json.Marshal(optSliceNone)
	assert.Nil(t, err)
	assert.Equal(t, "null", string(b))
	assert.True(t, optSliceNone.IsZero())

	// 2. Maps
	optMap := Some(map[string]int{"a": 1, "b": 2})
	b, err = json.Marshal(optMap)
	assert.Nil(t, err)

	var decodedMap Optional[map[string]int]
	err = json.Unmarshal(b, &decodedMap)
	assert.Nil(t, err)
	assert.True(t, decodedMap.IsPresent())
	assert.Equal(t, 2, len(decodedMap.MustValue()))
	assert.Equal(t, 1, decodedMap.MustValue()["a"])
	assert.Equal(t, 2, decodedMap.MustValue()["b"])
	assert.False(t, decodedMap.IsZero())

	// None map
	optMapNone := None[map[string]int]()
	b, err = json.Marshal(optMapNone)
	assert.Nil(t, err)
	assert.Equal(t, "null", string(b))
	assert.True(t, optMapNone.IsZero())

	// 3. Pointers
	x := 42
	optPtr := Some(&x)
	b, err = json.Marshal(optPtr)
	assert.Nil(t, err)
	assert.Equal(t, "42", string(b))
	assert.False(t, optPtr.IsZero())

	var decodedPtr Optional[*int]
	err = json.Unmarshal(b, &decodedPtr)
	assert.Nil(t, err)
	assert.True(t, decodedPtr.IsPresent())
	assert.Equal(t, 42, *decodedPtr.MustValue())

	// Nil pointer inside Some
	optNilPtr := Some[*int](nil)
	b, err = json.Marshal(optNilPtr)
	assert.Nil(t, err)
	assert.Equal(t, "null", string(b))
	assert.True(t, optNilPtr.IsPresent())
	assert.False(t, optNilPtr.IsZero())
}

func TestOptional_Adversarial_DirectIsZeroMatrix(t *testing.T) {
	// Explicit Some(zeroValue) MUST report IsZero() == false
	assert.False(t, Some("").IsZero())
	assert.False(t, Some(0).IsZero())
	assert.False(t, Some(int64(0)).IsZero())
	assert.False(t, Some(uint(0)).IsZero())
	assert.False(t, Some(uint64(0)).IsZero())
	assert.False(t, Some(0.0).IsZero())
	assert.False(t, Some(false).IsZero())
	assert.False(t, Some(time.Time{}).IsZero())
	assert.False(t, Some([]int{}).IsZero())
	assert.False(t, Some(map[string]int{}).IsZero())
	assert.False(t, Some[*int](nil).IsZero())

	// Unset None[T]() MUST report IsZero() == true
	assert.True(t, None[string]().IsZero())
	assert.True(t, None[int]().IsZero())
	assert.True(t, None[int64]().IsZero())
	assert.True(t, None[uint]().IsZero())
	assert.True(t, None[uint64]().IsZero())
	assert.True(t, None[float64]().IsZero())
	assert.True(t, None[bool]().IsZero())
	assert.True(t, None[time.Time]().IsZero())
	assert.True(t, None[[]int]().IsZero())
	assert.True(t, None[map[string]int]().IsZero())
	assert.True(t, None[*int]().IsZero())
}

func TestGeneric_Allocations(t *testing.T) {
	items := []string{"a", "b", "c", "d"}

	allocs := testing.AllocsPerRun(100, func() {
		_ = Chunked(items, 2)
	})

	// Chunked shares the backing slice, making only 1 allocation for the outer slice of slices.
	if allocs > 1 {
		t.Errorf("expected 1 allocation for Chunked, got %v", allocs)
	}
}
