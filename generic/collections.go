// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generic

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Set represents an unordered collection of unique elements.
// Set is not safe for concurrent use by multiple goroutines without external synchronization.
type Set[T comparable] map[T]struct{}

// NewSet creates a new [Set] initialized with the provided items.
func NewSet[T comparable](items ...T) Set[T] {
	s := make(Set[T], len(items))
	for _, item := range items {
		s[item] = struct{}{}
	}

	return s
}

// Add inserts an item into the set.
func (s Set[T]) Add(item T) {
	if s == nil {
		return
	}

	s[item] = struct{}{}
}

// Has reports whether the set contains the specified item.
func (s Set[T]) Has(item T) bool {
	if s == nil {
		return false
	}

	_, ok := s[item]

	return ok
}

// Intersect returns a new set containing the intersection of two sets.
func (s Set[T]) Intersect(other Set[T]) Set[T] {
	res := make(Set[T])
	if s == nil || other == nil {
		return res
	}

	for k := range s {
		if other.Has(k) {
			res.Add(k)
		}
	}

	return res
}

// ToSlice converts the set back into a flat slice.
// The order of the elements in the returned slice is undefined.
func (s Set[T]) ToSlice() []T {
	if s == nil {
		return nil
	}

	res := make([]T, 0, len(s))
	for k := range s {
		res = append(res, k)
	}

	return res
}

type cacheItem[V any] struct {
	value     V
	expiresAt time.Time
}

// cacheImpl contains the underlying cache storage and synchronization primitives.
type cacheImpl[K comparable, V any] struct {
	mu          sync.RWMutex
	data        map[K]cacheItem[V]
	stopJanitor chan struct{}
	closed      atomic.Bool
}

// Cache implements a thread-safe, in-memory key-value store with Time-To-Live (TTL) expiration
// and automatic background memory reclamation.
// It is safe for concurrent use by multiple goroutines.
type Cache[K comparable, V any] struct {
	impl *cacheImpl[K, V]
}

// DefaultCleanupInterval is the default frequency for background TTL reclamation.
const DefaultCleanupInterval = 30 * time.Second

// NewCache creates and initializes a new thread-safe [Cache] instance with automatic
// background memory reclamation running every 30 seconds.
func NewCache[K comparable, V any]() *Cache[K, V] {
	return NewCacheWithJanitor[K, V](DefaultCleanupInterval)
}

// NewCacheWithJanitor creates a new [Cache] instance with a custom background cleanup interval.
// If cleanupInterval <= 0, background cleanup is disabled (passive eviction and PurgeExpired only).
func NewCacheWithJanitor[K comparable, V any](cleanupInterval time.Duration) *Cache[K, V] {
	impl := &cacheImpl[K, V]{
		data:        make(map[K]cacheItem[V]),
		stopJanitor: make(chan struct{}),
	}
	c := &Cache[K, V]{impl: impl}

	if cleanupInterval > 0 {
		go impl.runJanitor(cleanupInterval)
		runtime.SetFinalizer(c, func(outer *Cache[K, V]) {
			outer.Close()
		})
	}

	return c
}

func (impl *cacheImpl[K, V]) runJanitor(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			impl.PurgeExpired()
		case <-impl.stopJanitor:
			return
		}
	}
}

// Close stops the background janitor goroutine and unregisters the finalizer.
// It is safe to call multiple times.
func (c *Cache[K, V]) Close() {
	if c == nil || c.impl == nil {
		return
	}
	if c.impl.closed.CompareAndSwap(false, true) {
		close(c.impl.stopJanitor)
		runtime.SetFinalizer(c, nil)
	}
}

func (c *Cache[K, V]) getOrInitImpl() *cacheImpl[K, V] {
	if c == nil {
		return nil
	}
	if c.impl == nil {
		c.impl = &cacheImpl[K, V]{
			data:        make(map[K]cacheItem[V]),
			stopJanitor: make(chan struct{}),
		}
	}
	return c.impl
}

// Set stores a value in the cache under the specified key, with an associated TTL duration.
func (c *Cache[K, V]) Set(key K, val V, ttl time.Duration) {
	impl := c.getOrInitImpl()
	if impl == nil {
		return
	}

	impl.mu.Lock()
	defer impl.mu.Unlock()

	if impl.data == nil {
		impl.data = make(map[K]cacheItem[V])
	}

	impl.data[key] = cacheItem[V]{
		value:     val,
		expiresAt: time.Now().Add(ttl),
	}
}

// Get retrieves a value from the cache by key.
//
// If the key is not found, or if the associated TTL has expired, Get returns
// the zero value of type V and false. Expired items are lazily deleted from memory.
func (c *Cache[K, V]) Get(key K) (V, bool) {
	if c == nil || c.impl == nil {
		var zero V
		return zero, false
	}

	now := time.Now()

	// 1. Fast path: Optimistic read under RLock
	c.impl.mu.RLock()
	if c.impl.data == nil {
		c.impl.mu.RUnlock()
		var zero V
		return zero, false
	}

	item, ok := c.impl.data[key]
	if !ok {
		c.impl.mu.RUnlock()
		var zero V
		return zero, false
	}

	if now.Before(item.expiresAt) {
		val := item.value
		c.impl.mu.RUnlock()
		return val, true
	}
	c.impl.mu.RUnlock()

	// 2. Slow path: Expired key. Acquire write lock to lazily evict.
	c.impl.mu.Lock()
	defer c.impl.mu.Unlock()

	// Double-check under write lock in case another goroutine updated key between RUnlock and Lock
	if item, ok = c.impl.data[key]; ok && !time.Now().Before(item.expiresAt) {
		delete(c.impl.data, key)
	}

	var zero V
	return zero, false
}

// Delete removes a key from the cache.
// It returns true if the key existed and was not expired at the time of deletion.
func (c *Cache[K, V]) Delete(key K) bool {
	if c == nil || c.impl == nil {
		return false
	}

	c.impl.mu.Lock()
	defer c.impl.mu.Unlock()

	if c.impl.data == nil {
		return false
	}

	item, ok := c.impl.data[key]
	if !ok {
		return false
	}

	delete(c.impl.data, key)
	return time.Now().Before(item.expiresAt)
}

// Len returns the number of active, unexpired items currently in the cache.
// Any expired items discovered during the check are purged.
func (c *Cache[K, V]) Len() int {
	if c == nil || c.impl == nil {
		return 0
	}

	now := time.Now()

	// 1. Optimistic read check
	c.impl.mu.RLock()
	if c.impl.data == nil {
		c.impl.mu.RUnlock()
		return 0
	}

	hasExpired := false
	activeCount := 0
	for _, item := range c.impl.data {
		if now.Before(item.expiresAt) {
			activeCount++
		} else {
			hasExpired = true
		}
	}
	c.impl.mu.RUnlock()

	if !hasExpired {
		return activeCount
	}

	// 2. If expired items exist, acquire write lock and purge
	c.impl.mu.Lock()
	defer c.impl.mu.Unlock()
	c.impl.purgeExpiredLocked(now)
	return len(c.impl.data)
}

// Clear removes all items from the cache and reclaims underlying map memory.
func (c *Cache[K, V]) Clear() {
	if c == nil || c.impl == nil {
		return
	}

	c.impl.mu.Lock()
	defer c.impl.mu.Unlock()

	c.impl.data = make(map[K]cacheItem[V])
}

// PurgeExpired removes all expired items from the cache and returns the number of evicted items.
func (c *Cache[K, V]) PurgeExpired() int {
	if c == nil || c.impl == nil {
		return 0
	}
	return c.impl.PurgeExpired()
}

func (impl *cacheImpl[K, V]) PurgeExpired() int {
	impl.mu.Lock()
	defer impl.mu.Unlock()
	return impl.purgeExpiredLocked(time.Now())
}

func (impl *cacheImpl[K, V]) purgeExpiredLocked(now time.Time) int {
	if impl.data == nil {
		return 0
	}

	evicted := 0
	for k, item := range impl.data {
		if !now.Before(item.expiresAt) {
			delete(impl.data, k)
			evicted++
		}
	}

	if len(impl.data) == 0 && evicted > 1024 {
		impl.data = make(map[K]cacheItem[V])
	}

	return evicted
}
