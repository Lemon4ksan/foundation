// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package limiter_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/sync/limiter"
	"github.com/lemon4ksan/foundation/testkit/assert"
	"github.com/lemon4ksan/foundation/testkit/require"
)

func TestRepro_SlidingWindowLimiter_ZeroLimitPanic(t *testing.T) {
	t.Parallel()

	// Zero-limit limiter must not panic and must return false
	sw := limiter.NewSlidingWindowLimiter(0, time.Minute)
	require.NotNil(t, sw)

	assert.NotPanics(t, func() {
		allowed, waitTime := sw.Allow(time.Now())
		assert.False(t, allowed)
		assert.Greater(t, waitTime, time.Duration(0))
	})
}

func TestRepro_AdaptiveLimiter_CancelRaceSlotLeak(t *testing.T) {
	t.Parallel()

	// Limiter with capacity 1
	l := limiter.NewAdaptiveLimiter(1)

	// Hold the only available slot
	require.NoError(t, l.Acquire(context.Background()))

	// Goroutine 2 waits with a context that gets cancelled
	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)

	var acquireErr error
	go func() {
		defer wg.Done()
		acquireErr = l.Acquire(ctx)
	}()

	// Wait for goroutine to block
	time.Sleep(20 * time.Millisecond)

	// Concurrently cancel context and release the slot
	cancel()
	l.Release(10 * time.Millisecond)

	wg.Wait()

	// If acquire was cancelled, it must report context cancellation error
	if acquireErr != nil {
		assert.ErrorIs(t, acquireErr, context.Canceled)
	}

	// Invariant: After cancellation/release cycle, the limiter slot MUST NOT be leaked.
	// We must be able to acquire the slot again within 100ms.
	acquireCtx, acquireCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer acquireCancel()

	err := l.Acquire(acquireCtx)
	require.NoError(t, err, "slot was permanently leaked after race between context cancel and release!")
	l.Release(10 * time.Millisecond)
}
