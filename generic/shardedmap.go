// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generic

import (
	"hash/maphash"
	"maps"
	"sync"
)

// ShardCount is the number of shards in the sharded map (power of two).
const (
	ShardCount = 32
	shardMask  = ShardCount - 1
)

type paddedShard[K comparable, V any] struct {
	mu    sync.RWMutex
	items map[K]V
	_     [32]byte // Cache-line padding isolating 64-byte L1 cache lines across CPU cores
}

// ShardedMap is a high-performance, thread-safe sharded map with zero lock contention between different keys,
// hardware-accelerated maphash hashing, and cache-line alignment to eliminate False Sharing.
type ShardedMap[K comparable, V any] struct {
	shards [ShardCount]paddedShard[K, V]
	seed   maphash.Seed
}

// NewShardedMap creates a new sharded map.
func NewShardedMap[K comparable, V any]() *ShardedMap[K, V] {
	m := &ShardedMap[K, V]{
		seed: maphash.MakeSeed(),
	}
	for i := range ShardCount {
		m.shards[i].items = make(map[K]V)
	}

	return m
}

func (m *ShardedMap[K, V]) getShard(key K) *paddedShard[K, V] {
	idx := int(maphash.Comparable(m.seed, key) & shardMask)
	_ = &m.shards[ShardCount-1]

	return &m.shards[idx]
}

// Get retrieves the value for the given key from the sharded map.
func (m *ShardedMap[K, V]) Get(key K) (V, bool) {
	if m == nil {
		var zero V
		return zero, false
	}

	shard := m.getShard(key)
	shard.mu.RLock()
	val, ok := shard.items[key]
	shard.mu.RUnlock()

	return val, ok
}

// TryGet attempts to retrieve the value for the given key without blocking if the shard is write-locked.
func (m *ShardedMap[K, V]) TryGet(key K) (val V, ok, acquired bool) {
	if m == nil {
		var zero V
		return zero, false, false
	}

	shard := m.getShard(key)
	if !shard.mu.TryRLock() {
		var zero V
		return zero, false, false
	}

	v, exists := shard.items[key]
	shard.mu.RUnlock()
	return v, exists, true
}

// Set sets the value for the given key in the sharded map.
func (m *ShardedMap[K, V]) Set(key K, val V) {
	if m == nil {
		return
	}

	shard := m.getShard(key)
	shard.mu.Lock()
	shard.items[key] = val
	shard.mu.Unlock()
}

// TrySet attempts to set the value for the given key without blocking if the shard is locked.
func (m *ShardedMap[K, V]) TrySet(key K, val V) bool {
	if m == nil {
		return false
	}

	shard := m.getShard(key)
	if !shard.mu.TryLock() {
		return false
	}

	shard.items[key] = val
	shard.mu.Unlock()
	return true
}

// GetOrSet returns the existing value for the key if present.
// Otherwise, it stores and returns the given value. The loaded result is true if the value was loaded, false if stored.
func (m *ShardedMap[K, V]) GetOrSet(key K, val V) (actual V, loaded bool) {
	if m == nil {
		return val, false
	}

	shard := m.getShard(key)
	shard.mu.Lock()
	if existing, ok := shard.items[key]; ok {
		shard.mu.Unlock()
		return existing, true
	}

	shard.items[key] = val
	shard.mu.Unlock()
	return val, false
}

// Delete deletes the value for the given key from the sharded map.
func (m *ShardedMap[K, V]) Delete(key K) {
	if m == nil {
		return
	}

	shard := m.getShard(key)
	shard.mu.Lock()
	delete(shard.items, key)
	shard.mu.Unlock()
}

// Len returns the total number of items stored across all shards.
func (m *ShardedMap[K, V]) Len() int {
	if m == nil {
		return 0
	}

	count := 0
	for i := range ShardCount {
		shard := &m.shards[i]
		shard.mu.RLock()
		count += len(shard.items)
		shard.mu.RUnlock()
	}

	return count
}

// All returns a copy of all key-value pairs in the sharded map.
func (m *ShardedMap[K, V]) All() map[K]V {
	if m == nil {
		return nil
	}

	result := make(map[K]V)
	for i := range ShardCount {
		shard := &m.shards[i]
		shard.mu.RLock()
		maps.Copy(result, shard.items)
		shard.mu.RUnlock()
	}

	return result
}
