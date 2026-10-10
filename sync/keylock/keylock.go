// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package keylock

import (
	"iter"
	"sync"
)

// refCounter tracks the active waiters and holding status of a specific key.
type refCounter struct {
	mu    sync.Mutex
	count int  // Number of goroutines waiting for or holding this key
	held  bool // True if the key is currently held by a goroutine
}

var refCounterPool = sync.Pool{
	New: func() any {
		return &refCounter{}
	},
}

// Mutex implements a thread-safe, striped key-based locking mechanism.
//
// The zero value for Mutex is valid and ready for use.
// It is safe for concurrent use.
type Mutex[K comparable] struct {
	mu    sync.Mutex
	locks map[K]*refCounter
}

// New creates a [Mutex] instance for keys of type K.
//
// Executes in O(1) time and allocates memory for the tracking structures.
func New[K comparable]() *Mutex[K] {
	return &Mutex[K]{
		locks: make(map[K]*refCounter),
	}
}

// Keys returns a newly allocated slice containing all currently tracked keys, including those waiting to be locked.
//
// Complexity: O(N) time and space, where N is the number of tracked keys.
func (km *Mutex[K]) Keys() []K {
	km.mu.Lock()
	defer km.mu.Unlock()

	keys := make([]K, 0, len(km.locks))
	for k := range km.locks {
		keys = append(keys, k)
	}

	return keys
}

// IsLocked reports whether the specified key is currently held by a goroutine.
//
// Unlike simple map lookups, IsLocked only returns true if the lock has been successfully acquired.
// Complexity: Amortized O(1) time; zero-allocation.
func (km *Mutex[K]) IsLocked(key K) bool {
	km.mu.Lock()
	defer km.mu.Unlock()

	ref, exists := km.locks[key]

	return exists && ref.held
}

// ForceUnlock forcibly unlocks the specified key.
//
// Use only in test fixtures or when no other waiters are present to prevent orphans and concurrency races.
// Complexity: Amortized O(1) time.
func (km *Mutex[K]) ForceUnlock(key K) {
	km.mu.Lock()

	ref, exists := km.locks[key]
	if !exists || !ref.held {
		km.mu.Unlock()
		return
	}

	ref.held = false
	ref.count--
	toRecycle := false
	if ref.count == 0 {
		delete(km.locks, key)
		toRecycle = true
	}
	km.mu.Unlock()

	defer func() {
		_ = recover() // Swallow any double-unlock panics gracefully
	}()

	ref.mu.Unlock()
	if toRecycle {
		refCounterPool.Put(ref)
	}
}

// Lock acquires exclusive ownership of the specified key.
//
// If the key is already locked by another goroutine, the calling goroutine blocks until the key is unlocked.
// Complexity: Amortized O(1) time.
func (km *Mutex[K]) Lock(key K) {
	km.mu.Lock()
	if km.locks == nil {
		km.locks = make(map[K]*refCounter)
	}

	ref, exists := km.locks[key]
	if !exists {
		ref = refCounterPool.Get().(*refCounter)
		ref.count = 0
		ref.held = false
		km.locks[key] = ref
	}

	ref.count++
	km.mu.Unlock()

	ref.mu.Lock()

	// Update the held status under global lock protection to keep IsLocked accurate.
	km.mu.Lock()
	ref.held = true
	km.mu.Unlock()
}

// Unlock releases exclusive ownership of the specified key.
//
// It panics if the key is not currently registered or held by any goroutine.
// Complexity: Amortized O(1) time; zero-allocation.
func (km *Mutex[K]) Unlock(key K) {
	km.mu.Lock()

	ref, exists := km.locks[key]
	if !exists || !ref.held {
		km.mu.Unlock()
		panic("foundation/keylock: unlock of unlocked key")
	}

	ref.held = false

	ref.count--
	toRecycle := false
	if ref.count == 0 {
		delete(km.locks, key)
		toRecycle = true
	}

	km.mu.Unlock()

	ref.mu.Unlock()
	if toRecycle {
		refCounterPool.Put(ref)
	}
}

// TryLock attempts to acquire exclusive ownership of the specified key without blocking.
//
// It returns true if the lock was successfully acquired, and false if the key is already held by another goroutine.
// Complexity: Amortized O(1) time.
func (km *Mutex[K]) TryLock(key K) bool {
	km.mu.Lock()
	if km.locks == nil {
		km.locks = make(map[K]*refCounter)
	}

	ref, exists := km.locks[key]
	if !exists {
		ref = refCounterPool.Get().(*refCounter)
		ref.count = 0
		ref.held = false
		km.locks[key] = ref
	}

	ref.count++
	km.mu.Unlock()

	if ref.mu.TryLock() {
		km.mu.Lock()
		ref.held = true
		km.mu.Unlock()

		return true
	}

	// Failed to acquire. Decrement reference count and cleanup if necessary.
	km.mu.Lock()

	ref.count--
	toRecycle := false
	if ref.count == 0 {
		delete(km.locks, key)
		toRecycle = true
	}

	km.mu.Unlock()

	if toRecycle {
		refCounterPool.Put(ref)
	}

	return false
}

// KeysSeq provides a sequence iterator yielding all currently tracked keys.
//
// Complexity: O(N) time and space for initialization, where N is the number of active tracking structures.
func (km *Mutex[K]) KeysSeq() iter.Seq[K] {
	return func(yield func(K) bool) {
		km.mu.Lock()
		if len(km.locks) == 0 {
			km.mu.Unlock()
			return
		}
		keys := make([]K, 0, len(km.locks))
		for k := range km.locks {
			keys = append(keys, k)
		}
		km.mu.Unlock()

		for _, k := range keys {
			if !yield(k) {
				return
			}
		}
	}
}

// WithLock securely executes fn while possessing exclusive ownership of the lock for the given key on km.
//
// It guarantees that the key is reliably unlocked upon return.
// Complexity: Amortized O(1) time excluding the execution duration of fn.
func WithLock[K comparable](km *Mutex[K], key K, fn func()) {
	km.Lock(key)
	defer km.Unlock(key)
	fn()
}

// WithLockResult securely executes fn while possessing exclusive ownership of the lock for the given key on km, returning its derived result and error.
//
// It guarantees that the key is reliably unlocked upon return.
// Complexity: Amortized O(1) time excluding the execution duration of fn.
func WithLockResult[K comparable, V any](km *Mutex[K], key K, fn func() (V, error)) (V, error) {
	km.Lock(key)
	defer km.Unlock(key)
	return fn()
}
