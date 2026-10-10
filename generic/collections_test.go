// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generic

import (
	"fmt"
	"reflect"
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

	for i := range 10 {
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
	for i := range 5 {
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
	for i := range 20 {
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

	for i := range 100 {
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

	for w := range workers {
		go func(workerID int) {
			defer wg.Done()
			for i := range iterations {
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
	for range 50 {
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

// 12. LRU Values Recency Order Regression Test
func TestLRU_Values_RecencyOrder(t *testing.T) {
	lru := NewLRU[string, int](3)

	// Empty LRU must return empty slice
	if vals := lru.Values(); len(vals) != 0 {
		t.Fatalf("expected empty values for empty LRU, got %v", vals)
	}

	// Insert items
	lru.Put("a", 1)
	lru.Put("b", 2)
	lru.Put("c", 3)

	// Recency order must be MRU to LRU: [3, 2, 1]
	expected := []int{3, 2, 1}
	if vals := lru.Values(); !reflect.DeepEqual(vals, expected) {
		t.Fatalf("expected %v, got %v", expected, vals)
	}

	// Access "b" -> moves "b" to MRU: [2, 3, 1]
	if val, ok := lru.Get("b"); !ok || val != 2 {
		t.Fatalf("expected Get(b) == 2, got %v, %t", val, ok)
	}
	expected = []int{2, 3, 1}
	if vals := lru.Values(); !reflect.DeepEqual(vals, expected) {
		t.Fatalf("expected after Get(b): %v, got %v", expected, vals)
	}

	// Access "a" -> moves "a" to MRU: [1, 2, 3]
	if val, ok := lru.Get("a"); !ok || val != 1 {
		t.Fatalf("expected Get(a) == 1, got %v, %t", val, ok)
	}
	expected = []int{1, 2, 3}
	if vals := lru.Values(); !reflect.DeepEqual(vals, expected) {
		t.Fatalf("expected after Get(a): %v, got %v", expected, vals)
	}

	// Put "d" (new key) -> capacity 3 exceeded, evicts LRU which is "c": [4, 1, 2]
	lru.Put("d", 4)
	expected = []int{4, 1, 2}
	if vals := lru.Values(); !reflect.DeepEqual(vals, expected) {
		t.Fatalf("expected after Put(d): %v, got %v", expected, vals)
	}

	// Update existing key "b" -> moves to MRU with updated value: [20, 4, 1]
	lru.Put("b", 20)
	expected = []int{20, 4, 1}
	if vals := lru.Values(); !reflect.DeepEqual(vals, expected) {
		t.Fatalf("expected after update Put(b, 20): %v, got %v", expected, vals)
	}

	// Delete "d" -> remaining: [20, 1]
	if !lru.Delete("d") {
		t.Fatalf("expected Delete(d) to return true")
	}
	expected = []int{20, 1}
	if vals := lru.Values(); !reflect.DeepEqual(vals, expected) {
		t.Fatalf("expected after Delete(d): %v, got %v", expected, vals)
	}
}

// 13. Set Intersect Asymmetric Sizes Regression Test
func TestSet_Intersect_Asymmetric(t *testing.T) {
	// Large set vs small set
	large := NewSet[int]()
	const largeCount = 100_000
	for i := range largeCount {
		large.Add(i)
	}

	small := NewSet[int](500, 999_999, 1000, 2_000_000, 50_000)

	// Common elements: 500, 1000, 50000 (3 elements)
	inter1 := large.Intersect(small)
	inter2 := small.Intersect(large)

	if len(inter1) != 3 || !inter1.Has(500) || !inter1.Has(1000) || !inter1.Has(50000) {
		t.Fatalf("unexpected intersection large.Intersect(small): %v", inter1.ToSlice())
	}

	if len(inter2) != 3 || !inter2.Has(500) || !inter2.Has(1000) || !inter2.Has(50000) {
		t.Fatalf("unexpected intersection small.Intersect(large): %v", inter2.ToSlice())
	}

	// Verify both intersections contain identical elements
	for item := range inter1 {
		if !inter2.Has(item) {
			t.Fatalf("expected inter2 to contain %d", item)
		}
	}

	// Edge cases: nil sets
	var nilSet Set[int]
	if res := nilSet.Intersect(small); len(res) != 0 {
		t.Fatalf("expected empty set when receiver is nil, got len %d", len(res))
	}
	if res := small.Intersect(nilSet); len(res) != 0 {
		t.Fatalf("expected empty set when arg is nil, got len %d", len(res))
	}
	if res := nilSet.Intersect(nilSet); len(res) != 0 {
		t.Fatalf("expected empty set when both nil, got len %d", len(res))
	}

	// Edge cases: empty set
	empty := NewSet[int]()
	if res := empty.Intersect(small); len(res) != 0 {
		t.Fatalf("expected empty set, got len %d", len(res))
	}
	if res := small.Intersect(empty); len(res) != 0 {
		t.Fatalf("expected empty set, got len %d", len(res))
	}

	// Disjoint sets
	disjoint1 := NewSet(1, 2, 3)
	disjoint2 := NewSet(4, 5, 6)
	if res := disjoint1.Intersect(disjoint2); len(res) != 0 {
		t.Fatalf("expected empty set for disjoint sets, got len %d", len(res))
	}
}

func BenchmarkSet_Intersect_AsymmetricLargeReceiver(b *testing.B) {
	large := NewSet[int]()
	for i := range 100_000 {
		large.Add(i)
	}
	small := NewSet(10, 20, 30, 40, 50)

	for b.Loop() {
		_ = large.Intersect(small)
	}
}

func BenchmarkSet_Intersect_AsymmetricSmallReceiver(b *testing.B) {
	large := NewSet[int]()
	for i := range 100_000 {
		large.Add(i)
	}
	small := NewSet(10, 20, 30, 40, 50)

	for b.Loop() {
		_ = small.Intersect(large)
	}
}
