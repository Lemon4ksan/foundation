// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generic

import (
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"testing"
)

// TestAdversarial_Guarded_ExtremeContention_RMW verifies that Guarded protects
// concurrent read/write and Read-Modify-Write invariants under high contention.
func TestAdversarial_Guarded_ExtremeContention_RMW(t *testing.T) {
	g := NewGuarded[int](0)

	const (
		numUpdaters = 30
		opsUpdater  = 500

		numReaders = 30
		opsReader  = 500
	)

	var wg sync.WaitGroup
	wg.Add(numUpdaters + numReaders)

	// Updaters: atomic increments via Update
	for range numUpdaters {
		go func() {
			defer wg.Done()
			for range opsUpdater {
				g.Update(func(current int) int {
					return current + 1
				})
			}
		}()
	}

	// Readers: concurrent Get and Read
	for range numReaders {
		go func() {
			defer wg.Done()
			for range opsReader {
				val := g.Get()
				_ = val

				g.Read(func(v int) {
					if v < 0 {
						t.Errorf("read negative value: %d", v)
					}
				})
			}
		}()
	}

	wg.Wait()

	expected := numUpdaters * opsUpdater
	finalVal := g.Get()
	if finalVal != expected {
		t.Fatalf("expected final value %d, got %d (lost updates detected!)", expected, finalVal)
	}
}

// TestAdversarial_Guarded_MapProtection_NoPanic verifies that Guarded completely
// prevents concurrent map read and map write crashes under heavy contention.
func TestAdversarial_Guarded_MapProtection_NoPanic(t *testing.T) {
	initialMap := make(map[int]int)
	g := NewGuarded[map[int]int](initialMap)

	const (
		numMutators = 25
		opsMutator  = 300

		numReaders = 25
		opsReader  = 300
	)

	var wg sync.WaitGroup
	wg.Add(numMutators + numReaders)

	// Mutators
	for m := range numMutators {
		go func(mutatorID int) {
			defer wg.Done()
			for i := range opsMutator {
				key := (mutatorID*100 + i) % 50
				g.Mutate(func(mPtr *map[int]int) {
					(*mPtr)[key] = i
				})
			}
		}(m)
	}

	// Readers
	for r := range numReaders {
		go func(readerID int) {
			defer wg.Done()
			for range opsReader {
				key := rand.IntN(50)
				g.Read(func(m map[int]int) {
					_ = m[key]
				})
			}
		}(r)
	}

	wg.Wait()

	// Verify map remains valid
	g.Read(func(m map[int]int) {
		if len(m) == 0 {
			t.Errorf("expected populated map, got 0 keys")
		}
	})
}

// TestAdversarial_Guarded_SwapAtomicity verifies that Swap operations yield strictly
// unique previous values under concurrent execution.
func TestAdversarial_Guarded_SwapAtomicity(t *testing.T) {
	g := NewGuarded[int64](0)

	const (
		numWorkers = 40
		opsWorker  = 100
	)

	var (
		wg         sync.WaitGroup
		seenValues sync.Map
		duplicates atomic.Int64
		idCounter  atomic.Int64
	)

	wg.Add(numWorkers)

	for range numWorkers {
		go func() {
			defer wg.Done()
			for range opsWorker {
				newVal := idCounter.Add(1)
				oldVal := g.Swap(newVal)

				if oldVal != 0 {
					if _, loaded := seenValues.LoadOrStore(oldVal, true); loaded {
						duplicates.Add(1)
					}
				}
			}
		}()
	}

	wg.Wait()

	if duplicates.Load() > 0 {
		t.Fatalf("detected %d duplicate old values returned by Swap (broken atomicity!)", duplicates.Load())
	}
}
