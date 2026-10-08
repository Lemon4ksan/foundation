// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generic

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

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
				ctx, cancel = context.WithTimeout(context.Background(), time.Duration(2+rand.IntN(15))*time.Millisecond)
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
