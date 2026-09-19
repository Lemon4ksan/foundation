// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generic

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

// 1. Basic Hit, Miss, and Existing Expiration behavior
func TestCache_BasicOperations(t *testing.T) {
	c := NewCache[string, int]()
	defer c.Close()

	if _, ok := c.Get("missing"); ok {
		t.Fatal("expected miss")
	}

	c.Set("k1", 42, 100*time.Millisecond)
	if v, ok := c.Get("k1"); !ok || v != 42 {
		t.Fatalf("expected 42, got %v (ok: %t)", v, ok)
	}
}

// 2. Passive Eviction on Get: Verifies that accessing an expired item removes it from internal storage
func TestCache_PassiveEvictionOnGet(t *testing.T) {
	c := NewCacheWithJanitor[string, string](0) // disable janitor to isolate passive test
	defer c.Close()

	c.Set("temp", "val", 10*time.Millisecond)
	time.Sleep(20 * time.Millisecond)

	// Before Get: item is logically expired but still physically stored in map
	// Calling Get should return miss AND evict the item
	val, ok := c.Get("temp")
	if ok || val != "" {
		t.Fatalf("expected miss on expired item, got %q, %t", val, ok)
	}

	// PurgeExpired should now report 0 because Get already purged it
	if purged := c.PurgeExpired(); purged != 0 {
		t.Fatalf("expected 0 purged (already lazily evicted by Get), got %d", purged)
	}
}

// 3. Passive Eviction on Len(): Verifies that Len() purges expired items and reports true active count
func TestCache_PassiveEvictionOnLen(t *testing.T) {
	c := NewCacheWithJanitor[string, int](0) // disable janitor
	defer c.Close()

	for i := 0; i < 10; i++ {
		c.Set(fmt.Sprintf("key_%d", i), i, 15*time.Millisecond)
	}
	c.Set("permanent", 999, 1*time.Hour)

	time.Sleep(25 * time.Millisecond)

	// Len must report 1 (only "permanent" remains) and physically purge the 10 expired keys
	length := c.Len()
	if length != 1 {
		t.Fatalf("expected Len() == 1, got %d", length)
	}

	if purged := c.PurgeExpired(); purged != 0 {
		t.Fatalf("expected 0 remaining expired items, got %d", purged)
	}
}

// 4. Deterministic Purge: Verifies PurgeExpired removes un-accessed keys
func TestCache_PurgeExpired(t *testing.T) {
	c := NewCacheWithJanitor[int, int](0) // disable janitor
	defer c.Close()

	// Add 5 short-lived items, 5 long-lived items
	for i := 0; i < 5; i++ {
		c.Set(i, i, 10*time.Millisecond)
	}
	for i := 5; i < 10; i++ {
		c.Set(i, i, 1*time.Hour)
	}

	time.Sleep(20 * time.Millisecond)

	// Purge without calling Get
	evicted := c.PurgeExpired()
	if evicted != 5 {
		t.Fatalf("expected 5 items evicted, got %d", evicted)
	}

	if c.Len() != 5 {
		t.Fatalf("expected 5 remaining items, got %d", c.Len())
	}

	// Verify long-lived items are intact
	for i := 5; i < 10; i++ {
		if val, ok := c.Get(i); !ok || val != i {
			t.Fatalf("expected key %d intact, got %v, %t", i, val, ok)
		}
	}
}

// 5. Active Background Janitor: Verifies automatic background sweep of un-accessed keys
func TestCache_BackgroundJanitor(t *testing.T) {
	// Janitor runs every 20ms
	c := NewCacheWithJanitor[string, int](20 * time.Millisecond)
	defer c.Close()

	// Set items with 15ms TTL
	for i := 0; i < 20; i++ {
		c.Set(fmt.Sprintf("auto_%d", i), i, 15*time.Millisecond)
	}

	// Do NOT call Get or Len. Wait 50ms for janitor tick to fire
	time.Sleep(50 * time.Millisecond)

	// All items must have been cleaned by janitor
	if purged := c.PurgeExpired(); purged != 0 {
		t.Fatalf("expected janitor to have purged all items, but PurgeExpired found %d", purged)
	}

	if c.Len() != 0 {
		t.Fatalf("expected Len() == 0 after janitor sweep, got %d", c.Len())
	}
}

// 6. Delete Method: Verifies deletion of active vs expired keys
func TestCache_Delete(t *testing.T) {
	c := NewCacheWithJanitor[string, string](0)
	defer c.Close()

	c.Set("active", "val1", 1*time.Hour)
	c.Set("expiring", "val2", 10*time.Millisecond)

	// Delete active key
	if !c.Delete("active") {
		t.Fatal("expected true when deleting active key")
	}
	if _, ok := c.Get("active"); ok {
		t.Fatal("key should have been deleted")
	}

	// Delete missing key
	if c.Delete("non_existent") {
		t.Fatal("expected false when deleting missing key")
	}

	// Delete expired key
	time.Sleep(20 * time.Millisecond)
	if c.Delete("expiring") {
		t.Fatal("expected false when deleting expired key")
	}
}

// 7. Clear Method: Verifies complete memory wipe
func TestCache_Clear(t *testing.T) {
	c := NewCache[int, int]()
	defer c.Close()

	for i := 0; i < 100; i++ {
		c.Set(i, i, 1*time.Hour)
	}

	c.Clear()

	if c.Len() != 0 {
		t.Fatalf("expected Len == 0 after Clear, got %d", c.Len())
	}
	if _, ok := c.Get(1); ok {
		t.Fatal("expected miss after Clear")
	}
}

// 8. Nil Cache Safety: Regression check for fuzz tests
func TestCache_NilSafety(t *testing.T) {
	var nilCache *Cache[string, string]

	nilCache.Set("k", "v", time.Hour)
	if _, ok := nilCache.Get("k"); ok {
		t.Fatal("expected false on nil cache Get")
	}
	if nilCache.Delete("k") {
		t.Fatal("expected false on nil cache Delete")
	}
	if nilCache.Len() != 0 {
		t.Fatal("expected 0 on nil cache Len")
	}
	if nilCache.PurgeExpired() != 0 {
		t.Fatal("expected 0 on nil cache PurgeExpired")
	}
	nilCache.Clear()
	nilCache.Close()
}

// 9. Concurrency & Race Detector Stress Test: Run with `go test -race`
func TestCache_ConcurrentStressRace(t *testing.T) {
	c := NewCacheWithJanitor[int, int](15 * time.Millisecond)
	defer c.Close()

	const workers = 20
	const iterations = 500

	var wg sync.WaitGroup
	wg.Add(workers)

	for w := 0; w < workers; w++ {
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				key := (workerID*iterations + i) % 50
				ttl := time.Duration((i%10)+1) * time.Millisecond

				switch i % 5 {
				case 0, 1:
					c.Set(key, i, ttl)
				case 2, 3:
					_, _ = c.Get(key)
				case 4:
					c.Delete(key)
				}

				if i%50 == 0 {
					_ = c.Len()
				}
				if i%100 == 0 {
					_ = c.PurgeExpired()
				}
			}
		}(w)
	}

	wg.Wait()
}

// 10. Double-Checked Locking Race Test
func TestCache_DoubleCheckedLockingRace(t *testing.T) {
	c := NewCacheWithJanitor[string, string](0)
	defer c.Close()

	// Initial expired item
	c.Set("race_key", "old_val", 1*time.Millisecond)
	time.Sleep(2 * time.Millisecond)

	var wg sync.WaitGroup
	wg.Add(2)

	var (
		getReturnedVal string
		getHit         bool
	)

	// Goroutine 1: calls Get()
	go func() {
		defer wg.Done()
		getReturnedVal, getHit = c.Get("race_key")
	}()

	// Goroutine 2: concurrently renews "race_key" with 1 hour TTL
	go func() {
		defer wg.Done()
		c.Set("race_key", "renewed_val", 1*time.Hour)
	}()

	wg.Wait()

	// After both finish, "race_key" must NOT be destroyed by stale eviction
	val, ok := c.Get("race_key")
	if !ok || val != "renewed_val" {
		t.Fatalf("expected renewed_val to persist, got %q (ok: %t)", val, ok)
	}
	_ = getReturnedVal
	_ = getHit
}

// 11. Finalizer and Goroutine Cleanup Verification
func TestCache_FinalizerCleanup(t *testing.T) {
	for i := 0; i < 50; i++ {
		func() {
			c := NewCacheWithJanitor[int, int](10 * time.Millisecond)
			c.Set(1, 1, 10*time.Millisecond)
			// c goes out of scope here without explicit Close()
		}()
	}

	// Trigger GC to run finalizers
	runtime.GC()
	time.Sleep(30 * time.Millisecond)
	runtime.GC()
}
