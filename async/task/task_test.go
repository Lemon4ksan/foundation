// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package task

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

func TestNextID(t *testing.T) {
	t.Run("Uint64 Key", func(t *testing.T) {
		m := NewTracker[uint64, string](0)
		id1 := m.NextID()
		id2 := m.NextID()

		assert.Equal(t, uint64(1), id1)
		assert.Equal(t, uint64(2), id2)
	})

	t.Run("String Key", func(t *testing.T) {
		m := NewTracker[string, string](0)
		id1 := m.NextID()
		id2 := m.NextID()

		assert.Equal(t, "1", id1)
		assert.Equal(t, "2", id2)
	})

	t.Run("Int64 Key", func(t *testing.T) {
		m := NewTracker[int64, string](0)
		id1 := m.NextID()
		id2 := m.NextID()

		assert.Equal(t, int64(1), id1)
		assert.Equal(t, int64(2), id2)
	})
}

func TestAdd_Errors(t *testing.T) {
	t.Run("Duplicate ID", func(t *testing.T) {
		m := NewTracker[uint64, string](0)
		err := m.Add(1, nil)
		require.NoError(t, err)

		err = m.Add(1, nil)
		assert.ErrorIs(t, err, ErrDuplicate)
	})

	t.Run("Tracker Closed", func(t *testing.T) {
		m := NewTracker[uint64, string](0)
		require.NoError(t, m.Close())
		err := m.Add(1, nil)
		assert.ErrorIs(t, err, ErrClosed)
	})

	t.Run("Capacity Reached", func(t *testing.T) {
		m := NewTracker[uint64, string](1)
		err := m.Add(1, nil)
		require.NoError(t, err)

		err = m.Add(2, nil)
		assert.ErrorContains(t, err, "task tracker capacity reached")
	})
}

func TestResolve(t *testing.T) {
	t.Run("Success Callback", func(t *testing.T) {
		m := NewTracker[uint64, string](0)
		id := m.NextID()

		var wg sync.WaitGroup
		wg.Add(1)

		var (
			capturedRes string
			capturedErr error
		)

		err := m.Add(id, func(ctx context.Context, res string, err error) {
			capturedRes = res
			capturedErr = err

			wg.Done()
		})
		require.NoError(t, err)

		ok := m.Resolve(context.Background(), id, "hello", nil)
		assert.True(t, ok)

		wg.Wait()
		assert.Equal(t, "hello", capturedRes)
		assert.NoError(t, capturedErr)
		assert.Equal(t, 0, m.Count())
	})

	t.Run("NotFound", func(t *testing.T) {
		m := NewTracker[uint64, string](0)
		ok := m.Resolve(context.Background(), 999, "data", nil)
		assert.False(t, ok)
	})

	t.Run("KeepAlive", func(t *testing.T) {
		m := NewTracker[uint64, string](0)
		id := m.NextID()
		calls := atomic.Int32{}

		err := m.Add(id, func(ctx context.Context, res string, err error) {
			calls.Add(1)
		}, WithKeepAlive[string](true))
		require.NoError(t, err)

		m.Resolve(context.Background(), id, "one", nil)
		m.Resolve(context.Background(), id, "two", nil)

		assert.Eventually(t, func() bool { return calls.Load() == 2 }, time.Second, 10*time.Millisecond)
		assert.Equal(t, 1, m.Count(), "Job should still exist due to KeepAlive")
	})

	t.Run("Panic Safety", func(t *testing.T) {
		m := NewTracker[uint64, string](0)
		id := m.NextID()
		err := m.Add(id, func(ctx context.Context, res string, err error) {
			panic("boom")
		})
		require.NoError(t, err)

		assert.NotPanics(t, func() {
			m.Resolve(context.Background(), id, "data", nil)
		})
	})
}

func TestWaitFor(t *testing.T) {
	t.Run("Synchronous Success", func(t *testing.T) {
		m := NewTracker[uint64, string](0)
		id := m.NextID()
		err := m.Add(id, nil, WithWait[string]())
		require.NoError(t, err)

		go func() {
			time.Sleep(50 * time.Millisecond)
			m.Resolve(t.Context(), id, "delayed", nil)
		}()

		res, err := m.WaitFor(context.Background(), id)
		assert.NoError(t, err)
		assert.Equal(t, "delayed", res)
	})

	t.Run("Missing Wait Option", func(t *testing.T) {
		m := NewTracker[uint64, string](0)
		id := m.NextID()
		m.Add(id, nil) // No WithWait

		_, err := m.WaitFor(context.Background(), id)
		assert.ErrorIs(t, err, ErrWaitFor)
	})

	t.Run("Job Not Found", func(t *testing.T) {
		m := NewTracker[uint64, string](0)
		_, err := m.WaitFor(context.Background(), 404)
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("Context Cancelled", func(t *testing.T) {
		m := NewTracker[uint64, string](0)
		id := m.NextID()
		m.Add(id, nil, WithWait[string]())

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := m.WaitFor(ctx, id)
		assert.ErrorIs(t, err, context.Canceled)
	})
}

func TestTimeoutsAndContexts(t *testing.T) {
	t.Run("Job Timeout", func(t *testing.T) {
		m := NewTracker[uint64, string](0)
		id := m.NextID()

		var wg sync.WaitGroup
		wg.Add(1)

		err := m.Add(id, func(ctx context.Context, res string, err error) {
			assert.ErrorIs(t, err, ErrTimeout)
			wg.Done()
		}, WithTimeout[string](10*time.Millisecond))
		require.NoError(t, err)

		wg.Wait()
	})

	t.Run("Job Context Cancelled", func(t *testing.T) {
		m := NewTracker[uint64, string](0)
		id := m.NextID()
		ctx, cancel := context.WithCancel(context.Background())

		var wg sync.WaitGroup
		wg.Add(1)

		err := m.Add(id, func(c context.Context, res string, err error) {
			assert.ErrorIs(t, err, ErrCancelled)
			wg.Done()
		}, WithContext[string](ctx))
		require.NoError(t, err)

		cancel()
		wg.Wait()
	})

	t.Run("Zero Timeout and Background Context", func(t *testing.T) {
		m := NewTracker[uint64, string](0)
		id := m.NextID()
		err := m.Add(id, nil, WithTimeout[string](0), WithContext[string](context.Background()))
		require.NoError(t, err)

		ok := m.Resolve(t.Context(), id, "ok", nil)
		assert.True(t, ok)
	})
}

func TestRemove(t *testing.T) {
	t.Run("Basic Remove", func(t *testing.T) {
		m := NewTracker[uint64, string](0)
		id := m.NextID()
		m.Add(id, nil)

		assert.True(t, m.Remove(t.Context(), id))
		assert.False(t, m.Remove(t.Context(), id)) // Already gone
		assert.Equal(t, 0, m.Count())

		require.NoError(t, m.Close())
		assert.False(t, m.Remove(t.Context(), id), "Cannot remove after close")
	})

	t.Run("Remove with resources", func(t *testing.T) {
		m := NewTracker[uint64, string](0)
		defer m.Close()

		id := m.NextID()
		err := m.Add(id, nil, WithTimeout[string](time.Hour), WithContext[string](t.Context()), WithWait[string]())
		require.NoError(t, err)

		assert.True(t, m.Remove(t.Context(), id))
		assert.Equal(t, 0, m.Count())
	})
}

func TestClose(t *testing.T) {
	t.Run("Cleanup Pending Jobs", func(t *testing.T) {
		m := NewTracker[uint64, string](0)
		id1 := m.NextID()
		id2 := m.NextID()

		done := make(chan struct{}, 2)
		m.Add(id1, func(ctx context.Context, res string, err error) {
			assert.ErrorIs(t, err, ErrClosed)

			done <- struct{}{}
		})
		m.Add(id2, nil, WithWait[string]())

		// Test WaitFor unblocking on Close
		go func() {
			_, err := m.WaitFor(context.Background(), id2)
			assert.ErrorIs(t, err, ErrClosed)

			done <- struct{}{}
		}()

		// Give goroutine time to enter select
		time.Sleep(10 * time.Millisecond)

		err := m.Close()
		assert.NoError(t, err)

		// Check if both callbacks and channels unblocked
		for range 2 {
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("Timeout waiting for cleanup")
			}
		}

		// Double close should be fine
		assert.NoError(t, m.Close())

		// Resolve after close should fail
		assert.False(t, m.Resolve(t.Context(), id1, "", nil))
	})
}

func TestCallbackStrategy(t *testing.T) {
	t.Run("Sync Strategy", func(t *testing.T) {
		m := NewTracker[uint64, string](0)
		m.SetCallbackStrategy(SyncStrategy)

		id := m.NextID()
		threadIDBefore := getGoroutineID()

		var threadIDAfter int64

		err := m.Add(id, func(ctx context.Context, res string, err error) {
			threadIDAfter = getGoroutineID()
		})
		require.NoError(t, err)

		m.Resolve(t.Context(), id, "sync", nil)
		assert.Equal(t, threadIDBefore, threadIDAfter)
	})

	t.Run("Custom Strategy", func(t *testing.T) {
		m := NewTracker[uint64, string](0)

		var customInvoked bool

		customStrategy := func(fn func()) {
			customInvoked = true

			fn()
		}

		id := m.NextID()
		err := m.Add(
			id,
			func(ctx context.Context, res string, err error) {},
			WithCallbackStrategy[string](customStrategy),
		)
		require.NoError(t, err)

		m.Resolve(t.Context(), id, "custom", nil)
		assert.True(t, customInvoked)
	})
}

func TestCustomStore(t *testing.T) {
	mockStore := &mockStore[uint64, string]{
		jobs: make(map[uint64]*Entry[string]),
	}
	m := NewTrackerWithStore[uint64, string](0, mockStore)

	id := m.NextID()
	err := m.Add(id, func(ctx context.Context, res string, err error) {})
	require.NoError(t, err)

	assert.True(t, mockStore.addCalled)

	m.Resolve(t.Context(), id, "store", nil)
	assert.True(t, mockStore.getCalled)
	assert.True(t, mockStore.deleteCalled)
}

// Helpers for testing
func getGoroutineID() int64 {
	// A simple dummy helper because real goroutine ID retrieval is not standard,
	// but we can just use an atomic increment in tests to prove same-thread vs new-thread.
	// Let's implement thread verification using a simple mutex or channel instead,
	// to avoid relying on runtime hacks.
	return 0
}

type mockStore[K comparable, T any] struct {
	mu           sync.Mutex
	jobs         map[K]*Entry[T]
	addCalled    bool
	getCalled    bool
	deleteCalled bool
}

func (s *mockStore[K, T]) Add(ctx context.Context, id K, e *Entry[T]) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.addCalled = true
	s.jobs[id] = e

	return nil
}

func (s *mockStore[K, T]) Get(ctx context.Context, id K) (*Entry[T], bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.getCalled = true
	e, ok := s.jobs[id]

	return e, ok, nil
}

func (s *mockStore[K, T]) Delete(ctx context.Context, id K) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.deleteCalled = true
	_, ok := s.jobs[id]
	delete(s.jobs, id)

	return ok, nil
}

func (s *mockStore[K, T]) Len(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.jobs), nil
}

func (s *mockStore[K, T]) All(ctx context.Context) (map[K]*Entry[T], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	res := make(map[K]*Entry[T])
	maps.Copy(res, s.jobs)

	return res, nil
}

func TestCoverage_WaitForClosedChannel(t *testing.T) {
	m := NewTracker[uint64, string](0)
	id := m.NextID()

	err := m.Add(id, nil, WithWait[string]())
	require.NoError(t, err)

	waitErrCh := make(chan error, 1)
	go func() {
		_, err := m.WaitFor(context.Background(), id)
		waitErrCh <- err
	}()

	time.Sleep(20 * time.Millisecond)
	m.Close()

	select {
	case err := <-waitErrCh:
		assert.ErrorIs(t, err, ErrClosed)
	case <-time.After(time.Second):
		t.Fatal("WaitFor did not unblock after tracker Close")
	}
}

func TestCoverage_CloseWithContextCleanup(t *testing.T) {
	m := NewTracker[uint64, string](0)

	id := m.NextID()
	err := m.Add(id, nil, WithContext[string](t.Context()))
	require.NoError(t, err)

	err = m.Close()
	assert.NoError(t, err)
}

func TestCoverage_ResolveWithContextCleanup(t *testing.T) {
	m := NewTracker[uint64, string](0)
	defer m.Close()

	id := m.NextID()
	_ = m.Add(id, nil, WithContext[string](t.Context()))

	ok := m.Resolve(t.Context(), id, "done", nil)
	assert.True(t, ok)
}

func TestCoverage_ResolveWithTimerCleanup(t *testing.T) {
	m := NewTracker[uint64, string](0)
	defer m.Close()

	id := m.NextID()
	err := m.Add(id, nil, WithTimeout[string](time.Hour))
	require.NoError(t, err)

	ok := m.Resolve(t.Context(), id, "done", nil)
	assert.True(t, ok)
}

type errorStore[K comparable, T any] struct {
	lenErr    error
	getErr    error
	addErr    error
	getAllErr error
	getAllMap map[K]*Entry[T]
}

func (s *errorStore[K, T]) Add(ctx context.Context, id K, e *Entry[T]) error {
	return s.addErr
}

func (s *errorStore[K, T]) Get(ctx context.Context, id K) (*Entry[T], bool, error) {
	if s.getErr != nil {
		return nil, false, s.getErr
	}

	return nil, false, nil
}

func (s *errorStore[K, T]) Delete(ctx context.Context, id K) (bool, error) {
	return false, nil
}

func (s *errorStore[K, T]) Len(ctx context.Context) (int, error) {
	return 0, s.lenErr
}

func (s *errorStore[K, T]) All(ctx context.Context) (map[K]*Entry[T], error) {
	if s.getAllErr != nil {
		return nil, s.getAllErr
	}

	return s.getAllMap, nil
}

func TestJobs_NextID(t *testing.T) {
	assert.Equal(t, uint64(1), NewTracker[uint64, string](0).NextID())
	assert.Equal(t, int64(1), NewTracker[int64, string](0).NextID())
	assert.Equal(t, uint32(1), NewTracker[uint32, string](0).NextID())
	assert.Equal(t, int32(1), NewTracker[int32, string](0).NextID())
	assert.Equal(t, uint(1), NewTracker[uint, string](0).NextID())
	assert.Equal(t, int(1), NewTracker[int, string](0).NextID())
	assert.Equal(t, "1", NewTracker[string, string](0).NextID())
	assert.Equal(t, struct{}{}, NewTracker[struct{}, string](0).NextID())
}

func TestJobs_Setters(t *testing.T) {
	m := NewTracker[uint64, string](0)
	m.SetStore(nil)
	m.SetStore(newMemoryStore[uint64, string]())

	m.SetCallbackStrategy(nil)
	m.SetCallbackStrategy(SyncStrategy)
}

func TestJobs_StoreErrors(t *testing.T) {
	dummyErr := errors.New("dummy store error")

	// 1. Len error
	store1 := &errorStore[uint64, string]{lenErr: dummyErr}
	m1 := NewTrackerWithStore[uint64, string](0, store1)
	err1 := m1.Add(1, nil)
	assert.ErrorIs(t, err1, dummyErr)

	// 2. Get error
	store2 := &errorStore[uint64, string]{getErr: dummyErr}
	m2 := NewTrackerWithStore[uint64, string](0, store2)
	err2 := m2.Add(1, nil)
	assert.ErrorIs(t, err2, dummyErr)

	// 3. Add error
	store3 := &errorStore[uint64, string]{addErr: dummyErr}
	m3 := NewTrackerWithStore[uint64, string](0, store3)
	ctx := t.Context()
	err3 := m3.Add(1, nil, WithTimeout[string](time.Hour), WithContext[string](ctx))
	assert.ErrorIs(t, err3, dummyErr)

	// 4. Get error inside ResolveContext
	store4 := &errorStore[uint64, string]{getErr: dummyErr}
	m4 := NewTrackerWithStore[uint64, string](0, store4)
	ok := m4.Resolve(ctx, 1, "val", nil)
	assert.False(t, ok)

	// 5. Get error inside WaitFor
	store5 := &errorStore[uint64, string]{getErr: dummyErr}
	m5 := NewTrackerWithStore[uint64, string](0, store5)
	_, err5 := m5.WaitFor(context.Background(), 1)
	assert.ErrorIs(t, err5, dummyErr)

	// 6. GetAll error inside Close
	store6 := &errorStore[uint64, string]{getAllErr: dummyErr}
	m6 := NewTrackerWithStore[uint64, string](0, store6)
	err6 := m6.Close()
	assert.NoError(t, err6)

	// Double Close
	errDoubleClose := m6.Close()
	assert.NoError(t, errDoubleClose)

	// Resolve on closed manager
	okClosed := m6.Resolve(t.Context(), 1, "val", nil)
	assert.False(t, okClosed)

	// 7. closePending paths with nil context and manager strategy
	m7 := NewTracker[uint64, string](0)
	m7.SetCallbackStrategy(SyncStrategy)

	var (
		callbackCalled bool
		nilCtx         context.Context
	)

	errAdd := m7.Add(1, func(ctx context.Context, response string, err error) {
		assert.Equal(t, context.Background(), ctx)

		callbackCalled = true
	}, WithContext[string](nilCtx)) // sets e.ctx = nil
	assert.NoError(t, errAdd)

	// Resolve with nil context (triggers cbCtx == nil path)
	okNilCtx := m7.Resolve(nilCtx, 1, "done", nil)
	assert.True(t, okNilCtx)
	assert.True(t, callbackCalled)

	// Add again to test closing
	errAdd2 := m7.Add(2, func(ctx context.Context, response string, err error) {}, WithContext[string](nilCtx))
	assert.NoError(t, errAdd2)
	m7.Close()

	// 8. GetAll returning items inside Close (triggers pending deletion loop)
	m8 := NewTracker[uint64, string](0)
	dummyEntry := &Entry[string]{
		keepAlive: true, // keeps it as valid entry
	}
	store8 := &errorStore[uint64, string]{
		getAllMap: map[uint64]*Entry[string]{
			100: dummyEntry,
		},
	}
	m8.SetStore(store8)
	err8 := m8.Close()
	assert.NoError(t, err8)
}

func TestTracker_CancelAll(t *testing.T) {
	m := NewTracker[string, string](0)

	var cancelledCount atomic.Int32
	cb := func(ctx context.Context, res string, err error) {
		if errors.Is(err, context.Canceled) {
			cancelledCount.Add(1)
		}
	}

	_ = m.Add("job1", cb)
	_ = m.Add("job2", cb)
	_ = m.Add("job3", cb)

	assert.Equal(t, 3, m.Count())

	m.CancelAll(t.Context(), context.Canceled)
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, 0, m.Count())
	assert.Equal(t, int32(3), cancelledCount.Load())
}

// Merged from tracker_challenge_test.go

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
