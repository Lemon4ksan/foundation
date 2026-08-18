// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generic

import (
	"sync"
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

// Cache implements a thread-safe, in-memory key-value store with Time-To-Live (TTL) expiration.
// It is safe for concurrent use by multiple goroutines.
type Cache[K comparable, V any] struct {
	mu   sync.RWMutex
	data map[K]cacheItem[V]
}

// NewCache creates and initializes a new thread-safe [Cache] instance.
func NewCache[K comparable, V any]() *Cache[K, V] {
	return &Cache[K, V]{
		data: make(map[K]cacheItem[V]),
	}
}

// Set stores a value in the cache under the specified key, with an associated TTL duration.
func (c *Cache[K, V]) Set(key K, val V, ttl time.Duration) {
	if c == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.data == nil {
		c.data = make(map[K]cacheItem[V])
	}

	c.data[key] = cacheItem[V]{
		value:     val,
		expiresAt: time.Now().Add(ttl),
	}
}

// Get retrieves a value from the cache by key.
//
// If the key is not found, or if the associated TTL has expired, Get returns
// the zero value of type V and false. Otherwise, it returns the value and true.
func (c *Cache[K, V]) Get(key K) (V, bool) {
	if c == nil {
		var zero V
		return zero, false
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.data == nil {
		var zero V
		return zero, false
	}

	item, ok := c.data[key]
	if !ok {
		var zero V
		return zero, false
	}

	if time.Now().After(item.expiresAt) {
		var zero V
		return zero, false
	}

	return item.value, true
}
