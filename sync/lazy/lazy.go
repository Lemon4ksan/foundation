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

// Value implements a thread-safe, generic lazy initializer with support for cache resetting and lockless reads.
//
// The zero value for Value is not valid; use [New] to initialize.
type Value[T any] struct {
	mu   sync.Mutex
	init func() (T, error)
	res  atomic.Pointer[result[T]]
}

// New returns a [Value] configured with the given initialization function.
func New[T any](init func() (T, error)) *Value[T] {
	return &Value[T]{init: init}
}

// Get returns the computed value.
//
// If the value has not been initialized yet, Get executes the initialization
// function and caches both the returned value and error. Subsequent calls to Get
// return this cached result immediately via a lockless atomic read without
// re-executing the function.
//
// Once initialized, multiple readers retrieve the value concurrently via
// lockless atomic loads with zero lock contention and zero memory allocations.
func (l *Value[T]) Get() (T, error) {
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

// Reset clears the cached state and marks the initializer as uncompleted,
// causing the next invocation of [Value.Get] to re-run the initialization function.
// Reset is safe for concurrent use: concurrent readers will either observe the
// previous result or wait and re-initialize safely without data races.
func (l *Value[T]) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.res.Store(nil)
}
