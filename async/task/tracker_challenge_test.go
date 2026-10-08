// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package task

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

// TestAdversarial_Tracker_ConcurrentResolveCancelRemove tests aggressive concurrent
// operations on Tracker under the race detector.
func TestAdversarial_Tracker_ConcurrentResolveCancelRemove(t *testing.T) {
	tracker := NewTracker[uint64, string](0)
	defer tracker.Close()

	const (
		numGoroutines = 40
		opsPerWorker  = 50
	)

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	var (
		resolvedCount atomic.Int64
		removedCount  atomic.Int64
		addedCount    atomic.Int64
	)

	for g := range numGoroutines {
		go func(workerID int) {
			defer wg.Done()

			for i := range opsPerWorker {
				id := tracker.NextID()
				strategy := AsyncStrategy
				if i%2 == 0 {
					strategy = SyncStrategy
				}

				addErr := tracker.Add(id, func(ctx context.Context, resp string, err error) {
					// callback should handle any outcome gracefully
				}, WithWait[string](), WithCallbackStrategy[string](strategy), WithTimeout[string](50*time.Millisecond))

				if addErr != nil {
					continue
				}
				addedCount.Add(1)

				// Interleave operations based on workerID and iteration
				switch (workerID + i) % 4 {
				case 0:
					// Manual resolution
					if tracker.Resolve(context.Background(), id, fmt.Sprintf("val-%d", id), nil) {
						resolvedCount.Add(1)
					}
				case 1:
					// Synchronous wait with short timeout
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
					_, _ = tracker.WaitFor(ctx, id)
					cancel()
				case 2:
					// Manual removal
					if tracker.Remove(context.Background(), id) {
						removedCount.Add(1)
					}
				case 3:
					// Periodic bulk cancellation
					if i%10 == 0 {
						tracker.CancelAll(context.Background(), errors.New("bulk cancelled"))
					}
				}
			}
		}(g)
	}

	wg.Wait()

	// Drain any remaining tasks
	tracker.CancelAll(context.Background(), errors.New("cleanup"))
	assert.Equal(t, 0, tracker.Count())
}

// TestAdversarial_Tracker_ThreeWayResolutionRace tests race conditions between
// timer timeout, context cancellation, and manual Resolve for the exact same task.
func TestAdversarial_Tracker_ThreeWayResolutionRace(t *testing.T) {
	tracker := NewTracker[uint64, int](0)
	defer tracker.Close()

	const numTasks = 300
	var wg sync.WaitGroup
	wg.Add(numTasks)

	var (
		callbackInvoked [numTasks]atomic.Int32
		waitForReturned [numTasks]atomic.Int32
	)

	for i := range numTasks {
		idx := i
		id := tracker.NextID()

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(5+rand.IntN(15))*time.Millisecond)

		err := tracker.Add(id, func(c context.Context, resp int, err error) {
			callbackInvoked[idx].Add(1)
		}, WithWait[int](), WithContext[int](ctx), WithTimeout[int](time.Duration(5+rand.IntN(15))*time.Millisecond))
		require.NoError(t, err)

		// Goroutine 1: Manual resolve with jitter
		go func() {
			time.Sleep(time.Duration(rand.IntN(15)) * time.Millisecond)
			tracker.Resolve(context.Background(), id, idx*10, nil)
		}()

		// Goroutine 2: Context cancellation trigger with jitter
		go func() {
			time.Sleep(time.Duration(rand.IntN(15)) * time.Millisecond)
			cancel()
		}()

		// Goroutine 3: WaitFor
		go func() {
			defer wg.Done()
			waitCtx, waitCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer waitCancel()

			_, _ = tracker.WaitFor(waitCtx, id)
			waitForReturned[idx].Add(1)
		}()
	}

	wg.Wait()

	// Give any trailing async callbacks a moment to finish
	time.Sleep(50 * time.Millisecond)

	for i := range numTasks {
		invoked := callbackInvoked[i].Load()
		if invoked > 1 {
			t.Fatalf("task %d: callback invoked %d times (must be <= 1)", i, invoked)
		}
		if waitForReturned[i].Load() != 1 {
			t.Fatalf("task %d: WaitFor returned %d times (must be exactly 1)", i, waitForReturned[i].Load())
		}
	}
}

// TestAdversarial_Tracker_KeepAliveStreamingRace tests concurrent chunk resolution
// on streaming (KeepAlive) tasks.
func TestAdversarial_Tracker_KeepAliveStreamingRace(t *testing.T) {
	tracker := NewTracker[string, int](0)
	defer tracker.Close()

	const (
		numStreams   = 20
		chunksStream = 30
	)

	var receivedChunks [numStreams]atomic.Int32

	for s := range numStreams {
		streamID := fmt.Sprintf("stream-%d", s)
		streamIdx := s

		err := tracker.Add(streamID, func(ctx context.Context, resp int, err error) {
			if err == nil {
				receivedChunks[streamIdx].Add(1)
			}
		}, WithKeepAlive[int](true), WithCallbackStrategy[int](SyncStrategy))
		require.NoError(t, err)
	}

	var wg sync.WaitGroup
	for s := range numStreams {
		streamID := fmt.Sprintf("stream-%d", s)
		for c := range chunksStream {
			wg.Add(1)
			go func(sid string, chunkVal int) {
				defer wg.Done()
				tracker.Resolve(context.Background(), sid, chunkVal, nil)
			}(streamID, c)
		}
	}

	wg.Wait()

	for s := range numStreams {
		assert.Equal(t, int32(chunksStream), receivedChunks[s].Load())
		streamID := fmt.Sprintf("stream-%d", s)
		assert.True(t, tracker.Remove(context.Background(), streamID))
	}

	assert.Equal(t, 0, tracker.Count())
}

// TestAdversarial_Tracker_CapacityBoundsUnderStorm verifies strict capacity
// enforcement under high concurrency.
func TestAdversarial_Tracker_CapacityBoundsUnderStorm(t *testing.T) {
	const capacity = 25
	tracker := NewTracker[uint64, struct{}](capacity)
	defer tracker.Close()

	const numWorkers = 100
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	var rejected atomic.Int64
	var accepted atomic.Int64

	for range numWorkers {
		go func() {
			defer wg.Done()
			id := tracker.NextID()
			err := tracker.Add(id, nil)
			if err != nil {
				rejected.Add(1)
			} else {
				accepted.Add(1)
			}
		}()
	}

	wg.Wait()

	count := tracker.Count()
	assert.LessOrEqual(t, count, capacity)
	assert.Equal(t, int64(numWorkers), rejected.Load()+accepted.Load())
}

// TestAdversarial_Tracker_NextID_MonotonicityAndTypes tests ID uniqueness across goroutines.
func TestAdversarial_Tracker_NextID_MonotonicityAndTypes(t *testing.T) {
	t.Run("Uint64", func(t *testing.T) {
		tr := NewTracker[uint64, int](0)
		const count = 5000
		var (
			wg sync.WaitGroup
			mu sync.Mutex
			m  = make(map[uint64]bool, count)
		)

		wg.Add(50)
		for range 50 {
			go func() {
				defer wg.Done()
				for range 100 {
					id := tr.NextID()
					mu.Lock()
					m[id] = true
					mu.Unlock()
				}
			}()
		}
		wg.Wait()
		assert.Equal(t, count, len(m))
	})

	t.Run("String", func(t *testing.T) {
		tr := NewTracker[string, int](0)
		const count = 2000
		var (
			wg sync.WaitGroup
			mu sync.Mutex
			m  = make(map[string]bool, count)
		)

		wg.Add(20)
		for range 20 {
			go func() {
				defer wg.Done()
				for range 100 {
					id := tr.NextID()
					mu.Lock()
					m[id] = true
					mu.Unlock()
				}
			}()
		}
		wg.Wait()
		assert.Equal(t, count, len(m))
	})
}

// TestAdversarial_Tracker_WaitForCancelledContext_NoLeak tests that cancelling WaitFor
// does not destroy or leak the underlying task entry before its eventual resolution.
func TestAdversarial_Tracker_WaitForCancelledContext_NoLeak(t *testing.T) {
	tracker := NewTracker[uint64, string](0)
	defer tracker.Close()

	id := tracker.NextID()
	var cbCalled atomic.Bool

	err := tracker.Add(id, func(ctx context.Context, resp string, err error) {
		cbCalled.Store(true)
	}, WithWait[string]())
	require.NoError(t, err)

	waitCtx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel context

	_, waitErr := tracker.WaitFor(waitCtx, id)
	assert.ErrorIs(t, waitErr, context.Canceled)

	// Task must still exist in tracker
	assert.Equal(t, 1, tracker.Count())

	// Resolve task manually later
	ok := tracker.Resolve(context.Background(), id, "late_result", nil)
	assert.True(t, ok)
	assert.Equal(t, 0, tracker.Count())

	time.Sleep(20 * time.Millisecond)
	assert.True(t, cbCalled.Load())
}
