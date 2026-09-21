// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package keylock_test

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/sync/keylock"
)

func TestEmpirical_Keylock_WithLockResult_MutualExclusion(t *testing.T) {
	km := keylock.New[string]()
	const key = "shared-key"
	const numWorkers = 20
	const iterations = 50

	var inCritical atomic.Int32
	var totalExec atomic.Int64
	var wg sync.WaitGroup

	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range iterations {
				val, err := keylock.WithLockResult(km, key, func() (int, error) {
					cur := inCritical.Add(1)
					if cur != 1 {
						t.Errorf("violation: %d active holders inside WithLockResult", cur)
					}
					time.Sleep(10 * time.Microsecond)
					inCritical.Add(-1)
					return 100, nil
				})
				if err != nil || val != 100 {
					t.Errorf("unexpected WithLockResult output: val=%d, err=%v", val, err)
				}
				totalExec.Add(1)
			}
		}()
	}

	wg.Wait()

	if km.IsLocked(key) {
		t.Fatalf("expected key to be unlocked after all workers finished")
	}
	if totalExec.Load() != int64(numWorkers*iterations) {
		t.Fatalf("expected %d executions, got %d", numWorkers*iterations, totalExec.Load())
	}
}

func TestEmpirical_Keylock_WithLockResult_ErrorAndPanicSafety(t *testing.T) {
	km := keylock.New[string]()
	const key = "err-key"

	// 1. Error returned from fn: lock must be released
	errExpected := errors.New("expected error")
	res, err := keylock.WithLockResult(km, key, func() (int, error) {
		return 0, errExpected
	})
	if !errors.Is(err, errExpected) {
		t.Fatalf("expected %v, got %v", errExpected, err)
	}
	if res != 0 {
		t.Fatalf("expected 0, got %d", res)
	}
	if km.IsLocked(key) {
		t.Fatalf("key was not unlocked after error")
	}

	// 2. Panic inside fn: defer must unlock the key upon recovery
	const panicKey = "panic-key"
	func() {
		defer func() {
			_ = recover()
		}()
		_, _ = keylock.WithLockResult(km, panicKey, func() (int, error) {
			panic("deliberate panic")
		})
	}()

	if km.IsLocked(panicKey) {
		t.Fatalf("key was not unlocked after panic in fn")
	}

	// Verify the key can immediately be acquired again
	reacquired := km.TryLock(panicKey)
	if !reacquired {
		t.Fatalf("failed to reacquire key after panic recovery")
	}
	km.Unlock(panicKey)
}

func BenchmarkWithLockResult(b *testing.B) {
	km := keylock.New[string]()
	const key = "bench-key"

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_, _ = keylock.WithLockResult(km, key, func() (int, error) {
			return 42, nil
		})
	}
}
