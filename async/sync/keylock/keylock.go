// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package keylock

import (
	"sync"
)

// refCounter tracks the active waiters and holding status of a specific key.
type refCounter struct {
	mu    sync.Mutex
	count int  // Number of goroutines waiting for or holding this key
	held  bool // True if the key is currently held by a goroutine
}

// KeyMutex implements a thread-safe, striped/key-based locking mechanism.
//
// It allows goroutine A to lock key "X" while goroutine B works concurrently
// with key "Y" without blockages. KeyMutex automatically reclaims unused internal
// mutexes from memory using reference counting.
//
// A KeyMutex is safe for concurrent use by multiple goroutines.
type KeyMutex[K comparable] struct {
	mu    sync.Mutex
	locks map[K]*refCounter
}

// New creates and returns a new [KeyMutex] instance for keys of type K.
func New[K comparable]() *KeyMutex[K] {
	return &KeyMutex[K]{
		locks: make(map[K]*refCounter),
	}
}

// Keys returns a newly allocated slice containing all currently tracked
// and locked keys.
//
// # Complexity
//
// Time Complexity: O(N), where N is the number of currently active locks.
// Space Complexity: O(N) allocation for the returned slice.
func (km *KeyMutex[K]) Keys() []K {
	km.mu.Lock()
	defer km.mu.Unlock()

	keys := make([]K, 0, len(km.locks))
	for k := range km.locks {
		keys = append(keys, k)
	}

	return keys
}

// IsLocked returns true if the specified key is currently locked.
//
// Unlike simple map lookups, IsLocked is highly precise: it only returns true
// if the lock has been successfully acquired by a goroutine, ignoring temporary
// states during active acquisition or failed TryLock attempts.
func (km *KeyMutex[K]) IsLocked(key K) bool {
	km.mu.Lock()
	defer km.mu.Unlock()

	ref, exists := km.locks[key]

	return exists && ref.held
}

// ForceUnlock forcibly unlocks and deletes the specified key from the registry,
// even if the caller is not the lock holder.
//
// # Safety Warning
//
// ForceUnlock violates standard mutex ownership invariants. If other goroutines
// are currently waiting for this lock (blocked inside [KeyMutex.Lock]), calling
// ForceUnlock will cause those waiting goroutines to acquire a disconnected,
// orphaned lock. When they eventually attempt to call [KeyMutex.Unlock],
// they will panic with "unlock of unlocked key".
// Use ForceUnlock only in test fixtures or when no other waiters are present.
func (km *KeyMutex[K]) ForceUnlock(key K) {
	km.mu.Lock()

	ref, exists := km.locks[key]
	if !exists {
		km.mu.Unlock()
		return
	}

	delete(km.locks, key)
	km.mu.Unlock()

	defer func() {
		_ = recover() // Swallow any double-unlock panics gracefully
	}()

	ref.mu.Unlock()
}

// Lock locks the specified key.
//
// If the key is already locked by another goroutine, the calling goroutine
// blocks until the key is unlocked.
func (km *KeyMutex[K]) Lock(key K) {
	km.mu.Lock()
	if km.locks == nil {
		km.locks = make(map[K]*refCounter)
	}

	ref, exists := km.locks[key]
	if !exists {
		ref = &refCounter{}
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

// Unlock unlocks the specified key, allowing waiting goroutines to proceed.
//
// Unlock panics if the key is not currently registered or held.
func (km *KeyMutex[K]) Unlock(key K) {
	km.mu.Lock()

	ref, exists := km.locks[key]
	if !exists {
		km.mu.Unlock()
		panic("foundation/keylock: unlock of unlocked key")
	}

	ref.held = false

	ref.count--
	if ref.count == 0 {
		delete(km.locks, key)
	}

	km.mu.Unlock()

	ref.mu.Unlock()
}

// TryLock attempts to lock the specified key without blocking the caller.
//
// It returns true if the lock was successfully acquired, and false if the
// key is already locked by another goroutine.
func (km *KeyMutex[K]) TryLock(key K) bool {
	km.mu.Lock()
	if km.locks == nil {
		km.locks = make(map[K]*refCounter)
	}

	ref, exists := km.locks[key]
	if !exists {
		ref = &refCounter{}
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
	if ref.count == 0 {
		delete(km.locks, key)
	}

	km.mu.Unlock()

	return false
}
