// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package keylock_test

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/sync/keylock"
)

// TestKeyLock_HeavySameKeyContention tests 64 goroutines contending on the same key
// modifying an unbuffered non-thread-safe slice.
func TestKeyLock_HeavySameKeyContention(t *testing.T) {
	km := keylock.New[string]()
	const (
		key           = "shared-resource"
		numGoroutines = 64
		iterations    = 500
	)

	var (
		wg    sync.WaitGroup
		slice []int
	)

	wg.Add(numGoroutines)
	for i := range numGoroutines {
		go func(workerID int) {
			defer wg.Done()
			for j := range iterations {
				km.Lock(key)
				slice = append(slice, workerID*1000+j)
				km.Unlock(key)
			}
		}(i)
	}

	wg.Wait()

	expectedLen := numGoroutines * iterations
	if len(slice) != expectedLen {
		t.Fatalf("slice length mismatch: expected %d, got %d", expectedLen, len(slice))
	}
	if km.IsLocked(key) {
		t.Fatal("key reported locked after all workers finished")
	}
}

// TestKeyLock_MultiKeyIndependentParallelism tests that locks on distinct keys
// do not serialize each other.
func TestKeyLock_MultiKeyIndependentParallelism(t *testing.T) {
	km := keylock.New[int]()
	const numKeys = 32
	const iterations = 200

	counters := make([]int, numKeys)
	var wg sync.WaitGroup

	wg.Add(numKeys)
	for k := range numKeys {
		go func(keyIndex int) {
			defer wg.Done()
			for range iterations {
				km.Lock(keyIndex)
				counters[keyIndex]++
				km.Unlock(keyIndex)
			}
		}(k)
	}

	wg.Wait()

	for k, c := range counters {
		if c != iterations {
			t.Fatalf("key %d counter mismatch: expected %d, got %d", k, iterations, c)
		}
	}
}

// TestKeyLock_ChurnAndPoolRecycling hammers the reference pool by creating, locking,
// unlocking thousands of ephemeral keys.
func TestKeyLock_ChurnAndPoolRecycling(t *testing.T) {
	km := keylock.New[string]()
	const numWorkers = 32
	const opsPerWorker = 1000

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for w := range numWorkers {
		go func(workerID int) {
			defer wg.Done()
			for i := range opsPerWorker {
				k := fmt.Sprintf("k-%d-%d", workerID, i%10) // Key reuse to trigger pool churn
				km.Lock(k)
				runtime.Gosched()
				km.Unlock(k)
			}
		}(w)
	}

	wg.Wait()

	keys := km.Keys()
	if len(keys) != 0 {
		t.Fatalf("expected 0 remaining keys in map after all unlocked, got %d: %v", len(keys), keys)
	}
}

// TestKeyLock_TryLock_Contention tests TryLock under concurrent contention.
func TestKeyLock_TryLock_Contention(t *testing.T) {
	km := keylock.New[string]()
	const key = "trylock-key"

	var (
		wg        sync.WaitGroup
		holds     atomic.Int64
		conflicts atomic.Int64
		stop      atomic.Bool
	)

	const numWorkers = 32
	wg.Add(numWorkers)

	for range numWorkers {
		go func() {
			defer wg.Done()
			for !stop.Load() {
				if km.TryLock(key) {
					holds.Add(1)
					runtime.Gosched()
					km.Unlock(key)
				} else {
					conflicts.Add(1)
					runtime.Gosched()
				}
			}
		}()
	}

	for holds.Load() < 2000 {
		runtime.Gosched()
	}
	stop.Store(true)
	wg.Wait()

	if km.IsLocked(key) {
		t.Fatal("key still locked after workers terminated")
	}
	if conflicts.Load() == 0 {
		t.Fatal("expected TryLock conflicts under heavy contention")
	}
}

// TestKeyLock_RandomWalkKeys performs random walks of Lock, TryLock, Unlock, IsLocked
// across multiple keys to uncover race conditions or deadlocks.
func TestKeyLock_RandomWalkKeys(t *testing.T) {
	km := keylock.New[int]()
	const numKeys = 8
	const numWorkers = 24
	const iterations = 500

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for range numWorkers {
		go func() {
			defer wg.Done()
			for range iterations {
				k := rand.IntN(numKeys)
				if rand.Float64() < 0.5 {
					km.Lock(k)
					runtime.Gosched()
					km.Unlock(k)
				} else {
					if km.TryLock(k) {
						runtime.Gosched()
						km.Unlock(k)
					}
				}
			}
		}()
	}

	wg.Wait()

	keys := km.Keys()
	if len(keys) != 0 {
		t.Fatalf("expected 0 active keys remaining, got %d", len(keys))
	}
}

// TestKeyLock_ZeroValue verifies that zero-value Mutex initializes properly on first Lock.
func TestKeyLock_ZeroValue(t *testing.T) {
	var km keylock.Mutex[string]

	km.Lock("alpha")
	if !km.IsLocked("alpha") {
		t.Fatal("expected alpha to be locked")
	}
	km.Unlock("alpha")

	if km.IsLocked("alpha") {
		t.Fatal("expected alpha to be unlocked")
	}
}

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

func TestEmpirical_Keylock_WithLockResult_MutualExclusion(t *testing.T) {
	km := keylock.New[string]()
	const key = "shared-key"
	const numWorkers = 20
	const iterations = 50

	var inCritical atomic.Int32
	var totalExec atomic.Int64
	var wg sync.WaitGroup

	for range numWorkers {
		wg.Go(func() {
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
		})
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

	b.ReportAllocs()

	for b.Loop() {
		_, _ = keylock.WithLockResult(km, key, func() (int, error) {
			return 42, nil
		})
	}
}
