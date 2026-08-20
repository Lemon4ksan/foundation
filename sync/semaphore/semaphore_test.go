// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package semaphore

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestSemaphore_Basic(t *testing.T) {
	sem := New(2)

	// Acquire 1
	if err := sem.Acquire(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Acquire 2
	if err := sem.Acquire(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Try acquire 3 with a timeout (should fail)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := sem.Acquire(ctx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Release one
	sem.Release()

	// Acquire 3 should now succeed
	if err := sem.Acquire(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test blocked acquire that successfully gets slot via Release
	// Currently sem is fully acquired (2 slots active, limit 2)
	var wg sync.WaitGroup
	wg.Add(1)

	var blockedAcquireErr error
	go func() {
		defer wg.Done()

		blockedAcquireErr = sem.Acquire(context.Background())
	}()

	time.Sleep(10 * time.Millisecond) // ensure it blocks
	sem.Release()                     // release slot to unblock it

	wg.Wait()

	if blockedAcquireErr != nil {
		t.Fatalf("expected blocked acquire to succeed, got: %v", blockedAcquireErr)
	}
}

func TestSemaphore_Resize(t *testing.T) {
	sem := New(1)

	// Acquire 1
	if err := sem.Acquire(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Resize to 2
	sem.Resize(2)

	// Acquire 2 should succeed immediately
	if err := sem.Acquire(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Resize back to 1 (this does not force release, but should block further acquires)
	sem.Resize(1)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	if err := sem.Acquire(ctx); err == nil {
		t.Fatal("expected block and timeout, got success")
	}

	// Release two
	sem.Release()
	sem.Release()

	// Now we should be able to acquire 1 (since limit is 1)
	if err := sem.Acquire(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSemaphore_ContextCancelled(t *testing.T) {
	sem := New(1)
	if err := sem.Acquire(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)

	var acquireErr error
	go func() {
		defer wg.Done()

		acquireErr = sem.Acquire(ctx)
	}()

	// Wait a bit to ensure it is waiting
	time.Sleep(10 * time.Millisecond)
	cancel()

	wg.Wait()

	if acquireErr == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
}

func TestSemaphore_AlreadyCancelled(t *testing.T) {
	sem := New(1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := sem.Acquire(ctx)
	if err == nil {
		t.Fatal("expected error on already cancelled context")
	}
}

func TestSemaphore_NegativeLimitsAndEdgeCases(t *testing.T) {
	// Negative limit coerced to 0
	sem := New(-1)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := sem.Acquire(ctx)
	if err == nil {
		t.Fatal("expected acquire to block and fail with timeout on 0-limit semaphore")
	}

	// Resize to negative
	sem.Resize(-5)
	if sem.limit != 0 {
		t.Errorf("expected limit=0, got %d", sem.limit)
	}
}
