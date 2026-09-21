// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lazy

import (
	"sync"
	"sync/atomic"
)

// result stores the outcome of the initialization function.
type result[T any] struct {
	value T
	err   error
}

// Lazy implements a thread-safe, generic lazy initializer with support
// for transactional cache resetting and lockless reads.
type Lazy[T any] struct {
	mu   sync.Mutex
	init func() (T, error)
	res  atomic.Pointer[result[T]]
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
// return this cached result immediately via a lockless atomic read without
// re-executing the function.
//
// Get is fully optimized for concurrent use: once initialized, multiple readers
// retrieve the value concurrently via lockless atomic loads with zero lock
// contention and zero memory allocations.
//
// # Complexity
//
// Time Complexity: O(1) lockless atomic load on already-initialized fast-path reads.
func (l *Lazy[T]) Get() (T, error) {
	// Fast Path: Check if the value is already cached via lockless atomic load.
	if r := l.res.Load(); r != nil {
		return r.value, r.err
	}

	// Slow Path: Acquire an exclusive write lock to perform initialization.
	l.mu.Lock()
	defer l.mu.Unlock()

	// Double-Check: Another goroutine might have initialized it while we were
	// waiting to acquire the exclusive lock.
	if r := l.res.Load(); r != nil {
		return r.value, r.err
	}

	val, err := l.init()
	r := &result[T]{
		value: val,
		err:   err,
	}
	l.res.Store(r)

	return val, err
}

// Reset clears the cached state and marks the initializer as uncompleted.
//
// The next invocation of [Lazy.Get] will re-run the initialization function.
// Reset is safe for concurrent use: concurrent readers will either observe the
// previous result or wait/re-initialize safely without data races.
func (l *Lazy[T]) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.res.Store(nil)
}
