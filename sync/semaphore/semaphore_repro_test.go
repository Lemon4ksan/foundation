// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package semaphore_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/sync/semaphore"
	"github.com/lemon4ksan/foundation/testkit/assert"
	"github.com/lemon4ksan/foundation/testkit/require"
)

func TestRepro_Semaphore_CancelRaceSlotLeak(t *testing.T) {
	t.Parallel()

	sem := semaphore.New(1)

	// Acquire slot 1
	require.NoError(t, sem.Acquire(context.Background()))

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)

	var acquireErr error
	go func() {
		defer wg.Done()
		acquireErr = sem.Acquire(ctx)
	}()

	time.Sleep(20 * time.Millisecond)

	// Cancel context while simultaneously releasing the slot
	cancel()
	sem.Release()

	wg.Wait()

	if acquireErr != nil {
		assert.ErrorIs(t, acquireErr, context.Canceled)
	}

	// Invariant: The slot must not be permanently leaked.
	acquireCtx, acquireCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer acquireCancel()

	err := sem.Acquire(acquireCtx)
	require.NoError(t, err, "semaphore slot was permanently leaked on cancelled acquire race!")
	sem.Release()
}

func TestRepro_Semaphore_ReleaseUnderflowGuard(t *testing.T) {
	t.Parallel()

	sem := semaphore.New(1)

	// Calling Release on an unacquired semaphore must not underflow active slots
	sem.Release()
	sem.Release()

	// Capacity was 1, so only 1 Acquire should succeed
	require.NoError(t, sem.Acquire(context.Background()))

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	// Second acquire must block and time out (not succeed due to negative active count)
	err := sem.Acquire(ctx)
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	sem.Release()
}
