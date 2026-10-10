// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package limiter_test

import (
	"context"
	"fmt"
	"math/rand/v2"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/async/rate"
	"github.com/lemon4ksan/foundation/sync/limiter"
)

// TestLimiter_Adaptive_CancelStormStress hammers Adaptive with concurrent acquisitions
// where many goroutines timeout and cancel their context.
func TestLimiter_Adaptive_CancelStormStress(t *testing.T) {
	l := limiter.NewAdaptive(10)
	const (
		numWorkers = 64
		iterations = 100
	)

	var (
		wg        sync.WaitGroup
		acquired  atomic.Int64
		cancelled atomic.Int64
	)

	wg.Add(numWorkers)
	for i := range numWorkers {
		go func(workerID int) {
			defer wg.Done()
			for range iterations {
				// Vary timeout between 0ms and 10ms
				timeout := time.Duration(rand.IntN(10)) * time.Millisecond
				ctx, cancel := context.WithTimeout(context.Background(), timeout)

				err := l.Acquire(ctx)
				cancel()

				if err == nil {
					acquired.Add(1)
					// Simulate some RTT between 1ms and 5ms
					time.Sleep(time.Duration(rand.IntN(5)+1) * time.Millisecond)
					l.Release(10 * time.Millisecond)
				} else {
					cancelled.Add(1)
				}
			}
		}(i)
	}

	wg.Wait()

	if acquired.Load() == 0 {
		t.Fatal("no requests acquired slot")
	}
	if cancelled.Load() == 0 {
		t.Fatal("no requests cancelled under stress")
	}

	// Verify that the limiter didn't leak all slots: we should still be able to acquire
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	if err := l.Acquire(ctx); err != nil {
		t.Fatalf("failed to acquire slot after cancel storm: %v (possible slot leak!)", err)
	}
	l.Release(10 * time.Millisecond)
}

// TestLimiter_Keyed_ConcurrentSweepAndClose tests Keyed limiter under simultaneous
// high key churn, background sweeping, and concurrent Close calls.
func TestLimiter_Keyed_ConcurrentSweepAndClose(t *testing.T) {
	kl := limiter.NewKeyed[string](rate.Limit(100), 10, 10*time.Millisecond)
	const numWorkers = 32

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	stop := make(chan struct{})

	for w := range numWorkers {
		go func(id int) {
			defer wg.Done()
			i := 0
			for {
				select {
				case <-stop:
					return
				default:
					key := fmt.Sprintf("key-%d-%d", id, i%20)
					_, err := kl.Allow(key)
					if err != nil {
						// ErrClosed is valid when closed
						return
					}
					i++
					if i%10 == 0 {
						runtime.Gosched()
					}
				}
			}
		}(w)
	}

	time.Sleep(50 * time.Millisecond)

	// Close concurrently while workers are running
	if err := kl.Close(); err != nil {
		t.Fatalf("unexpected error on Close: %v", err)
	}

	close(stop)
	wg.Wait()

	// Subsequent Allow calls must return ErrClosed
	_, err := kl.Allow("any-key")
	if err != limiter.ErrClosed {
		t.Fatalf("expected ErrClosed after Close(), got %v", err)
	}
}

// TestLimiter_SlidingWindow_StrictRateLimit verifies that SlidingWindow never
// allows more than `limit` events within the sliding window duration under concurrent requests.
func TestLimiter_SlidingWindow_StrictRateLimit(t *testing.T) {
	const limit = 20
	window := 100 * time.Millisecond
	sw := limiter.NewSlidingWindow(limit, window)

	const numWorkers = 50
	var (
		wg      sync.WaitGroup
		allowed atomic.Int64
		start   = time.Now()
	)

	wg.Add(numWorkers)
	for range numWorkers {
		go func() {
			defer wg.Done()
			now := time.Now()
			ok, _ := sw.Allow(now)
			if ok {
				allowed.Add(1)
			}
		}()
	}

	wg.Wait()

	elapsed := time.Since(start)
	if elapsed < window && allowed.Load() > limit {
		t.Fatalf(
			"SlidingWindow rate violation: allowed %d requests, limit is %d within %v",
			allowed.Load(),
			limit,
			window,
		)
	}
}
