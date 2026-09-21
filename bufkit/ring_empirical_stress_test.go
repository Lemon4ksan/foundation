// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bufkit_test

import (
	"sync"
	"testing"

	"github.com/lemon4ksan/foundation/bufkit"
)

// TestEmpirical_Ring_SPSC_HighThroughputWrapAround stresses the ring buffer
// across millions of items with small capacity (e.g. 4 and 16) to force continuous
// head/tail wrap-around under concurrent producer-consumer execution.
func TestEmpirical_Ring_SPSC_HighThroughputWrapAround(t *testing.T) {
	capacities := []int{2, 4, 16, 128}

	for _, capVal := range capacities {
		t.Run(t.Name(), func(t *testing.T) {
			ring := bufkit.NewRing[int](capVal)
			const totalItems = 500000

			var wg sync.WaitGroup
			wg.Add(2)

			// Producer
			go func() {
				defer wg.Done()
				for i := range totalItems {
					for !ring.Push(i) {
						// Spin-wait when full
					}
				}
			}()

			// Consumer
			go func() {
				defer wg.Done()
				for expected := range totalItems {
					for {
						val, ok := ring.Pop()
						if ok {
							if val != expected {
								t.Errorf("capacity %d: expected %d, got %d", capVal, expected, val)
							}
							break
						}
					}
				}
			}()

			wg.Wait()

			if ring.Len() != 0 {
				t.Fatalf("expected ring to be empty after consuming all items, got len=%d", ring.Len())
			}
		})
	}
}

// TestEmpirical_Ring_ResetStress verifies Reset correctness under cyclic usage.
func TestEmpirical_Ring_ResetStress(t *testing.T) {
	ring := bufkit.NewRing[int](8)

	for cycle := range 100 {
		for i := range 8 {
			if !ring.Push(cycle*100 + i) {
				t.Fatalf("cycle %d: push failed at item %d", cycle, i)
			}
		}
		if ring.Len() != 8 {
			t.Fatalf("cycle %d: expected len 8, got %d", cycle, ring.Len())
		}
		// Reset midway
		ring.Reset()
		if ring.Len() != 0 {
			t.Fatalf("cycle %d: expected len 0 after reset, got %d", cycle, ring.Len())
		}
		if _, ok := ring.Pop(); ok {
			t.Fatalf("cycle %d: pop on empty ring after reset succeeded", cycle)
		}
	}
}
