// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package breaker_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/sync/breaker"
)

// TestBreaker_ConcurrentStateTransitions verifies that concurrent callers driving
// high failure rates transition cleanly between Closed -> Open -> HalfOpen -> Closed.
func TestBreaker_ConcurrentStateTransitions(t *testing.T) {
	cb := breaker.New[int](breaker.Config{
		FailureThreshold: 0.5,
		Cooldown:         50 * time.Millisecond,
		MinRequests:      10,
		Window:           1 * time.Second,
	})

	const numWorkers = 32
	ctx := context.Background()

	var (
		wg          sync.WaitGroup
		failMode    atomic.Bool
		successes   atomic.Int64
		errOpenHits atomic.Int64
		stop        atomic.Bool
	)

	failMode.Store(false)

	wg.Add(numWorkers)
	for range numWorkers {
		go func() {
			defer wg.Done()
			for !stop.Load() {
				_, err := cb.Do(ctx, func(ctx context.Context) (int, error) {
					if failMode.Load() {
						return 0, errors.New("injected downstream failure")
					}
					return 1, nil
				})

				if err == nil {
					successes.Add(1)
				} else if errors.Is(err, breaker.ErrOpen) {
					errOpenHits.Add(1)
				}
				time.Sleep(1 * time.Millisecond)
			}
		}()
	}

	// Phase 1: All success for 50ms
	time.Sleep(50 * time.Millisecond)
	if cb.State() != breaker.StateClosed {
		t.Fatalf("expected StateClosed, got %v", cb.State())
	}

	// Phase 2: Inject failures to trip breaker
	failMode.Store(true)
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) && cb.State() != breaker.StateOpen {
		time.Sleep(1 * time.Millisecond)
	}

	if state := cb.State(); state != breaker.StateOpen {
		t.Fatalf("expected StateOpen, got %v", state)
	}

	// Phase 3: Recover downstream and wait for Cooldown
	failMode.Store(false)
	// Wait for Cooldown (50ms) and successful probe to transition back to Closed
	deadline = time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) && cb.State() != breaker.StateClosed {
		time.Sleep(5 * time.Millisecond)
	}

	stop.Store(true)
	wg.Wait()

	if state := cb.State(); state != breaker.StateClosed {
		t.Fatalf("expected Breaker to recover to StateClosed, got %v", state)
	}
	if errOpenHits.Load() == 0 {
		t.Fatal("expected ErrOpen rejections during StateOpen")
	}
}

// TestBreaker_HalfOpen_PanicPermanentWedge verifies whether a panic during the trial
// request in StateHalfOpen permanently wedges the breaker, preventing all future requests.
func TestBreaker_HalfOpen_PanicPermanentWedge(t *testing.T) {
	cb := breaker.New[int](breaker.Config{
		FailureThreshold: 0.5,
		Cooldown:         20 * time.Millisecond,
		MinRequests:      2,
		Window:           1 * time.Second,
	})

	ctx := context.Background()

	// 1. Force into StateOpen by failing MinRequests
	for range 2 {
		_, _ = cb.Do(ctx, func(ctx context.Context) (int, error) {
			return 0, errors.New("force failure")
		})
	}

	if cb.State() != breaker.StateOpen {
		t.Fatalf("expected StateOpen, got %v", cb.State())
	}

	// 2. Wait for Cooldown to elapse
	time.Sleep(30 * time.Millisecond)
	if cb.State() != breaker.StateHalfOpen {
		t.Fatalf("expected StateHalfOpen, got %v", cb.State())
	}

	// 3. The trial request in StateHalfOpen panics and caller recovers
	func() {
		defer func() {
			_ = recover()
		}()
		_, _ = cb.Do(ctx, func(ctx context.Context) (int, error) {
			panic("unexpected downstream panic in trial probe")
		})
	}()

	// 4. Now attempt subsequent healthy requests.
	// Can the breaker ever recover, or is it permanently wedged?
	var err error
	for range 5 {
		_, err = cb.Do(ctx, func(ctx context.Context) (int, error) {
			return 100, nil
		})
		if err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if err != nil {
		t.Fatalf(
			"BUG CONFIRMED: Circuit breaker is permanently wedged after panic in StateHalfOpen! State=%v, err=%v",
			cb.State(),
			err,
		)
	}
}
