// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generic

import (
	"maps"
	"sync"
)

// ShardCount is the number of shards in the sharded map.
const ShardCount = 32

// ShardedMap is a high-performance, thread-safe sharded map with zero lock contention between different keys.
type ShardedMap[K ~uint64 | ~int64, V any] struct {
	shards [ShardCount]struct {
		mu    sync.RWMutex
		items map[K]V
	}
}

// NewShardedMap creates a new sharded map.
func NewShardedMap[K ~uint64 | ~int64, V any]() *ShardedMap[K, V] {
	m := &ShardedMap[K, V]{}
	for i := range ShardCount {
		m.shards[i].items = make(map[K]V)
	}

	return m
}

// Get retrieves the value for the given key from the sharded map.
func (m *ShardedMap[K, V]) Get(key K) (V, bool) {
	if m == nil {
		var zero V
		return zero, false
	}

	shard := &m.shards[uint64(key)%ShardCount]
	shard.mu.RLock()
	val, ok := shard.items[key]
	shard.mu.RUnlock()

	return val, ok
}

// Set sets the value for the given key in the sharded map.
func (m *ShardedMap[K, V]) Set(key K, val V) {
	if m == nil {
		return
	}

	shard := &m.shards[uint64(key)%ShardCount]
	shard.mu.Lock()
	shard.items[key] = val
	shard.mu.Unlock()
}

// Delete deletes the value for the given key from the sharded map.
func (m *ShardedMap[K, V]) Delete(key K) {
	if m == nil {
		return
	}

	shard := &m.shards[uint64(key)%ShardCount]
	shard.mu.Lock()
	delete(shard.items, key)
	shard.mu.Unlock()
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
