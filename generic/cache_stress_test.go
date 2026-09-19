// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generic

import (
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// 1. Stress Test: Highly concurrent readers, writers, removers, and clearers under race detector
func TestCache_AdversarialConcurrentStress(t *testing.T) {
	c := NewCacheWithJanitor[string, int](5 * time.Millisecond)
	defer c.Close()

	const duration = 2 * time.Second
	stopCh := make(chan struct{})
	time.AfterFunc(duration, func() { close(stopCh) })

	var (
		wg        sync.WaitGroup
		opsRead   atomic.Int64
		opsWrite  atomic.Int64
		opsDelete atomic.Int64
		opsPurge  atomic.Int64
		opsLen    atomic.Int64
		opsClear  atomic.Int64
	)

	const numKeys = 200

	// 20 Readers
	for r := 0; r < 20; r++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(id * 1000)))
			for {
				select {
				case <-stopCh:
					return
				default:
					key := fmt.Sprintf("key_%d", rng.Intn(numKeys))
					_, _ = c.Get(key)
					opsRead.Add(1)
				}
			}
		}(r)
	}

	// 20 Writers with mixed TTLs
	for w := 0; w < 20; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(id * 2000)))
			for {
				select {
				case <-stopCh:
					return
				default:
					key := fmt.Sprintf("key_%d", rng.Intn(numKeys))
					val := rng.Int()
					var ttl time.Duration
					switch rng.Intn(5) {
					case 0:
						ttl = 0 // immediate expiration
					case 1:
						ttl = -10 * time.Millisecond // past
					case 2:
						ttl = time.Duration(rng.Intn(10)+1) * time.Millisecond // short
					case 3:
						ttl = 50 * time.Millisecond // medium
					case 4:
						ttl = 10 * time.Second // long
					}
					c.Set(key, val, ttl)
					opsWrite.Add(1)
				}
			}
		}(w)
	}

	// 10 Deleters
	for d := 0; d < 10; d++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(id * 3000)))
			for {
				select {
				case <-stopCh:
					return
				default:
					key := fmt.Sprintf("key_%d", rng.Intn(numKeys))
					_ = c.Delete(key)
					opsDelete.Add(1)
				}
			}
		}(d)
	}

	// 5 Purgers
	for p := 0; p < 5; p++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stopCh:
					return
				default:
					_ = c.PurgeExpired()
					opsPurge.Add(1)
					time.Sleep(10 * time.Millisecond)
				}
			}
		}()
	}

	// 5 Len checkers
	for l := 0; l < 5; l++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stopCh:
					return
				default:
					_ = c.Len()
					opsLen.Add(1)
					time.Sleep(10 * time.Millisecond)
				}
			}
		}()
	}

	// 2 Clearers
	for cl := 0; cl < 2; cl++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stopCh:
					return
				default:
					time.Sleep(200 * time.Millisecond)
					c.Clear()
					opsClear.Add(1)
				}
			}
		}()
	}

	wg.Wait()

	t.Logf("Stress test complete. Ops: Read=%d, Write=%d, Delete=%d, Purge=%d, Len=%d, Clear=%d",
		opsRead.Load(), opsWrite.Load(), opsDelete.Load(), opsPurge.Load(), opsLen.Load(), opsClear.Load())

	if opsRead.Load() == 0 || opsWrite.Load() == 0 {
		t.Fatal("stress test did not execute sufficient operations")
	}
}

// 2. Janitor Goroutine Leak Verification: Explicit Close and Garbage Collection Finalizer
func TestCache_JanitorGoroutineLeak(t *testing.T) {
	// Step A: Verify explicit Close terminates janitor goroutines immediately
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	initialGoroutines := runtime.NumGoroutine()

	const count = 100
	caches := make([]*Cache[int, int], count)
	for i := 0; i < count; i++ {
		caches[i] = NewCacheWithJanitor[int, int](10 * time.Millisecond)
	}

	afterCreateGoroutines := runtime.NumGoroutine()
	if afterCreateGoroutines < initialGoroutines+count-5 {
		t.Fatalf("expected at least %d goroutines, got %d (initial: %d)",
			initialGoroutines+count, afterCreateGoroutines, initialGoroutines)
	}

	// Explicitly close all
	for i := 0; i < count; i++ {
		caches[i].Close()
	}

	// Wait for goroutines to exit
	time.Sleep(50 * time.Millisecond)
	afterCloseGoroutines := runtime.NumGoroutine()
	if afterCloseGoroutines > initialGoroutines+5 {
		t.Fatalf("goroutines leaked after Close(): initial=%d, afterClose=%d",
			initialGoroutines, afterCloseGoroutines)
	}

	// Step B: Verify finalizer terminates janitors when caches are abandoned without Close()
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	baseGoroutines := runtime.NumGoroutine()

	func() {
		localCaches := make([]*Cache[int, int], count)
		for i := 0; i < count; i++ {
			localCaches[i] = NewCacheWithJanitor[int, int](10 * time.Millisecond)
			localCaches[i].Set(i, i, time.Hour)
		}
		// localCaches goes out of scope here
		_ = localCaches
	}()

	// Force GC and wait for finalizers
	for retry := 0; retry < 5; retry++ {
		runtime.GC()
		time.Sleep(50 * time.Millisecond)
		if runtime.NumGoroutine() <= baseGoroutines+5 {
			break
		}
	}

	finalGoroutines := runtime.NumGoroutine()
	if finalGoroutines > baseGoroutines+5 {
		t.Fatalf("goroutines leaked after GC finalization: base=%d, final=%d",
			baseGoroutines, finalGoroutines)
	}

	// Step C: Concurrent Close() calls must not panic
	c := NewCacheWithJanitor[string, string](10 * time.Millisecond)
	var closeWg sync.WaitGroup
	for i := 0; i < 50; i++ {
		closeWg.Add(1)
		go func() {
			defer closeWg.Done()
			c.Close()
		}()
	}
	closeWg.Wait()
}

// 3. Key Churn and Memory Reclamation Verification
func TestCache_KeyChurnMemoryReclamation(t *testing.T) {
	c := NewCacheWithJanitor[string, []byte](10 * time.Millisecond)
	defer c.Close()

	// 100 persistent items
	for i := 0; i < 100; i++ {
		c.Set(fmt.Sprintf("perm_%d", i), make([]byte, 1024), 1*time.Hour)
	}

	// Measure baseline memory
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Churn 100,000 unique keys with 5ms TTL
	const churnTotal = 100000
	payload := make([]byte, 256)

	for i := 0; i < churnTotal; i++ {
		key := fmt.Sprintf("churn_%d", i)
		c.Set(key, payload, 5*time.Millisecond)
		if i%10000 == 0 {
			time.Sleep(10 * time.Millisecond)
			c.PurgeExpired()
		}
	}

	// Allow remaining churn keys to expire and purge
	time.Sleep(30 * time.Millisecond)
	c.PurgeExpired()

	// Verify only the 100 persistent items remain
	if c.Len() != 100 {
		t.Fatalf("expected exactly 100 items remaining, got %d", c.Len())
	}

	// Measure memory after churn and GC
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	t.Logf("HeapAlloc before churn: %d KB, after churn: %d KB, diff: %d KB",
		m1.HeapAlloc/1024, m2.HeapAlloc/1024, int64(m2.HeapAlloc-m1.HeapAlloc)/1024)

	// Verify that heap does not retain 100,000 payload objects
	// 100,000 * 256 bytes payload = ~25MB. Heap growth should be well below 15MB.
	heapDiffKB := int64(m2.HeapAlloc-m1.HeapAlloc) / 1024
	if heapDiffKB > 20000 {
		t.Fatalf("excessive heap growth after churn: %d KB", heapDiffKB)
	}
}

// 4. Eviction Accuracy and Sub-millisecond Expiration
func TestCache_EvictionAccuracyAdversarial(t *testing.T) {
	c := NewCacheWithJanitor[string, int](0) // disable janitor for exact timing
	defer c.Close()

	// Boundary 1: Immediate and negative TTL
	c.Set("zero_ttl", 100, 0)
	c.Set("negative_ttl", 200, -1*time.Second)

	if val, ok := c.Get("zero_ttl"); ok || val != 0 {
		t.Fatalf("expected immediate miss on zero TTL, got val=%v, ok=%t", val, ok)
	}
	if val, ok := c.Get("negative_ttl"); ok || val != 0 {
		t.Fatalf("expected immediate miss on negative TTL, got val=%v, ok=%t", val, ok)
	}

	// Boundary 2: Nanosecond accuracy
	c.Set("fast", 1, 15*time.Millisecond)
	c.Set("slow", 2, 80*time.Millisecond)

	// Immediately: both present
	if v, ok := c.Get("fast"); !ok || v != 1 {
		t.Fatalf("expected fast present immediately")
	}
	if v, ok := c.Get("slow"); !ok || v != 2 {
		t.Fatalf("expected slow present immediately")
	}

	// Sleep past fast TTL but before slow TTL
	time.Sleep(30 * time.Millisecond)

	// fast must miss, slow must hit
	if v, ok := c.Get("fast"); ok {
		t.Fatalf("expected fast expired, got %v", v)
	}
	if v, ok := c.Get("slow"); !ok || v != 2 {
		t.Fatalf("expected slow still active, got %v (ok=%t)", v, ok)
	}

	// Boundary 3: Delete return value
	// Delete on active item returns true
	if !c.Delete("slow") {
		t.Fatal("expected Delete on active slow key to return true")
	}
	// Delete on already deleted item returns false
	if c.Delete("slow") {
		t.Fatal("expected Delete on already deleted key to return false")
	}

	// Boundary 4: PurgeExpired accuracy
	c.Set("p1", 1, 10*time.Millisecond)
	c.Set("p2", 2, 10*time.Millisecond)
	c.Set("p3", 3, 200*time.Millisecond)
	c.Set("p4", 4, 200*time.Millisecond)

	time.Sleep(25 * time.Millisecond)

	purged := c.PurgeExpired()
	if purged != 2 {
		t.Fatalf("expected exactly 2 purged, got %d", purged)
	}
	if c.Len() != 2 {
		t.Fatalf("expected Len() == 2, got %d", c.Len())
	}
	if _, ok := c.Get("p3"); !ok {
		t.Fatal("expected p3 to survive purge")
	}
	if _, ok := c.Get("p4"); !ok {
		t.Fatal("expected p4 to survive purge")
	}
}

// 5. Zero-value struct handling
func TestCache_ZeroValueHandling(t *testing.T) {
	var zero Cache[string, int]

	// Read operations on zero value Cache
	if _, ok := zero.Get("missing"); ok {
		t.Fatal("expected false from zero-value Get")
	}
	if zero.Len() != 0 {
		t.Fatal("expected 0 from zero-value Len")
	}
	if zero.Delete("missing") {
		t.Fatal("expected false from zero-value Delete")
	}
	if zero.PurgeExpired() != 0 {
		t.Fatal("expected 0 from zero-value PurgeExpired")
	}
	zero.Clear()
	zero.Close()

	// Initialized via Set
	zero.Set("k", 42, time.Hour)
	if val, ok := zero.Get("k"); !ok || val != 42 {
		t.Fatalf("expected 42 after lazy init via Set, got %v, ok=%t", val, ok)
	}
}

// 6. Benchmark concurrent read/write throughput
func BenchmarkCache_Concurrent90Read10Write(b *testing.B) {
	c := NewCacheWithJanitor[int, int](50 * time.Millisecond)
	defer c.Close()

	for i := 0; i < 1000; i++ {
		c.Set(i, i, 5*time.Second)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		for pb.Next() {
			k := rng.Intn(1000)
			if rng.Intn(10) == 0 {
				c.Set(k, k, 5*time.Second)
			} else {
				_, _ = c.Get(k)
			}
		}
	})
}
