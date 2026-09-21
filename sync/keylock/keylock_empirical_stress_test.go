// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package keylock_test

import (
	"fmt"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/sync/keylock"
)

// TestEmpirical_KeyLock_MutualExclusionAndLeak verifies that under heavy
// concurrent contention across shared and distinct keys, mutual exclusion
// is strictly maintained and all internal entries are reclaimed without memory leaks.
func TestEmpirical_KeyLock_MutualExclusionAndLeak(t *testing.T) {
	km := keylock.New[string]()

	const numKeys = 20
	const numWorkers = 50
	const iterationsPerWorker = 200

	inCritical := make([]atomic.Int32, numKeys)
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for w := range numWorkers {
		go func(workerID int) {
			defer wg.Done()
			r := rand.New(rand.NewPCG(uint64(workerID), uint64(time.Now().UnixNano())))

			for range iterationsPerWorker {
				keyIdx := r.IntN(numKeys)
				keyStr := fmt.Sprintf("key-%d", keyIdx)
				opType := r.IntN(3)

				switch opType {
				case 0: // Lock / Unlock
					km.Lock(keyStr)
					val := inCritical[keyIdx].Add(1)
					if val != 1 {
						t.Errorf("violation on %s: %d active holders", keyStr, val)
					}
					// Micro-yield to stress interleaving
					time.Sleep(time.Microsecond)
					inCritical[keyIdx].Add(-1)
					km.Unlock(keyStr)

				case 1: // TryLock / Unlock
					if km.TryLock(keyStr) {
						val := inCritical[keyIdx].Add(1)
						if val != 1 {
							t.Errorf("violation on %s: %d active holders", keyStr, val)
						}
						time.Sleep(time.Microsecond)
						inCritical[keyIdx].Add(-1)
						km.Unlock(keyStr)
					}

				case 2: // WithLock
					keylock.WithLock(km, keyStr, func() {
						val := inCritical[keyIdx].Add(1)
						if val != 1 {
							t.Errorf("violation on %s: %d active holders", keyStr, val)
						}
						time.Sleep(time.Microsecond)
						inCritical[keyIdx].Add(-1)
					})
				}
			}
		}(w)
	}

	wg.Wait()

	// Verify no keys remain locked or leaked in km.locks
	remaining := km.Keys()
	if len(remaining) != 0 {
		t.Fatalf("expected 0 remaining locks in map, found %d: %v", len(remaining), remaining)
	}
}

// TestEmpirical_KeyLock_HighChurnKeys verifies that creating and unlocking
// thousands of unique keys concurrently leaves zero lingering map entries.
func TestEmpirical_KeyLock_HighChurnKeys(t *testing.T) {
	km := keylock.New[int]()

	const totalKeys = 10000
	var wg sync.WaitGroup
	const workers = 20
	chunk := totalKeys / workers

	wg.Add(workers)
	for w := range workers {
		go func(startIdx int) {
			defer wg.Done()
			for i := startIdx; i < startIdx+chunk; i++ {
				km.Lock(i)
				if !km.IsLocked(i) {
					t.Errorf("expected key %d to be locked", i)
				}
				km.Unlock(i)
			}
		}(w * chunk)
	}

	wg.Wait()

	keys := km.Keys()
	if len(keys) != 0 {
		t.Fatalf("expected 0 active keys after high churn, found %d", len(keys))
	}
}

// TestEmpirical_KeyLock_SameKeyHeavyContention tests 50 goroutines competing
// aggressively for the exact same key with mixed Lock and TryLock.
func TestEmpirical_KeyLock_SameKeyHeavyContention(t *testing.T) {
	km := keylock.New[string]()
	const sharedKey = "hot-key"
	const numWorkers = 50
	const iterations = 100

	var inCritical atomic.Int32
	var totalAcquired atomic.Int64
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for range numWorkers {
		go func() {
			defer wg.Done()
			for i := range iterations {
				if i%2 == 0 {
					km.Lock(sharedKey)
					cur := inCritical.Add(1)
					if cur != 1 {
						t.Errorf("expected 1 inside critical section, got %d", cur)
					}
					totalAcquired.Add(1)
					inCritical.Add(-1)
					km.Unlock(sharedKey)
				} else {
					if km.TryLock(sharedKey) {
						cur := inCritical.Add(1)
						if cur != 1 {
							t.Errorf("expected 1 inside critical section, got %d", cur)
						}
						totalAcquired.Add(1)
						inCritical.Add(-1)
						km.Unlock(sharedKey)
					}
				}
			}
		}()
	}

	wg.Wait()

	if km.IsLocked(sharedKey) {
		t.Fatalf("expected hot-key to be unlocked")
	}
	if len(km.Keys()) != 0 {
		t.Fatalf("expected 0 active keys, got %d", len(km.Keys()))
	}
}
