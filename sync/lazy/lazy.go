// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lazy

import (
	"sync"
)

// Lazy implements a thread-safe, generic lazy initializer with support
// for transactional cache resetting.
type Lazy[T any] struct {
	mu    sync.RWMutex
	init  func() (T, error)
	value T
	err   error
	done  bool
}

// New creates and returns a new [Lazy] instance configured with the given
// initialization function.
func New[T any](init func() (T, error)) *Lazy[T] {
	return &Lazy[T]{init: init}
}

// Get returns the computed value.
//
// If the value has not been initialized yet, Get executes the initialization
// function and caches both the returned value and error. Subsequent calls to Get
// return this cached result immediately without re-executing the function.
//
// Get is fully optimized for concurrent use: once initialized, multiple readers
// retrieve the value concurrently via non-blocking read-locks.
//
// # Complexity
//
// Time Complexity: O(1) on already-initialized fast-path reads.
func (l *Lazy[T]) Get() (T, error) {
	// Fast Path: Check if the value is already cached under a shared read lock.
	l.mu.RLock()

	if l.done {
		val, err := l.value, l.err
		l.mu.RUnlock()
		return val, err
	}

	l.mu.RUnlock()

	// Slow Path: Acquire an exclusive write lock to perform initialization.
	l.mu.Lock()
	defer l.mu.Unlock()

	// Double-Check: Another goroutine might have initialized it while we were
	// waiting to acquire the exclusive write lock.
	if l.done {
		return l.value, l.err
	}

	l.value, l.err = l.init()
	l.done = true

	return l.value, l.err
}

// Reset clears the cached state and marks the initializer as uncompleted.
//
// The next invocation of [Lazy.Get] will re-run the initialization function.
// Reset is safe for concurrent use and blocks any active readers during the
// cache invalidation phase to prevent data races.
func (l *Lazy[T]) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.done = false

	var zero T

	l.value = zero
	l.err = nil
}
