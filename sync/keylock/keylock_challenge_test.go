// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package keylock_test

import (
	"fmt"
	"math/rand/v2"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"

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
			for j := 0; j < iterations; j++ {
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
