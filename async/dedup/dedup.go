// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dedup

import (
	"context"
	"errors"
	"sync"
)

// ErrWorkerPanicked is returned to secondary waiting callers
// if the initiating worker goroutine terminates due to a panic.
var ErrWorkerPanicked = errors.New("worker panicked during execution")

// Result represents the deferred execution outcome of a [CallFn] worker.
type Result[V any] struct {
	// Val contains the value returned by a successful worker execution.
	Val V
	// Err contains the error returned by the worker execution.
	Err error
	// PanicVal contains the recovered panic value if the worker panicked.
	PanicVal any
}

// call represents the state of an active or completing execution.
//
// Lock ordering invariant: To prevent deadlocks, the group-level lock
// (Group.mu) must always be acquired before the call-level lock (call.mu).
type call[V any] struct {
	mu        sync.Mutex
	cancel    context.CancelFunc
	waiters   map[chan<- Result[V]]context.Context
	initiator chan<- Result[V]
	done      bool
	val       V
	err       error
	panicVal  any
}

// Group cooridinates concurrent executions and suppresses duplicate calls
// for identical parameterized keys.
//
// The zero value of Group is ready to use and safe for concurrent execution
// by multiple goroutines.
type Group[K comparable, V any] struct {
	mu    sync.Mutex
	calls map[K]*call[V]
}

// CallFn represents a context-aware generic function executed by a [Group].
type CallFn[V any] func(ctx context.Context) (V, error)

// Do executes and suppresses duplicate concurrent calls for a given key.
//
// If an execution for the key is already in progress, Do blocks subsequent callers
// until the active execution completes, returning the shared result.
//
// If the context passed to Do is cancelled while waiting, the caller returns immediately
// with the context error. If all concurrent waiting contexts are cancelled, the active
// worker's context is cancelled immediately.
//
// If the worker function panics, the panic is re-raised (propagated) on the goroutine
// of the initiating caller, while secondary waiting callers receive [ErrWorkerPanicked].
//
// It returns an error if the passed context is already expired.
func (group *Group[K, V]) Do(ctx context.Context, key K, callFn CallFn[V]) (V, error) {
	if group == nil {
		var zero V
		return zero, errors.New("batto: group is nil")
	}

	if callFn == nil {
		var zero V
		return zero, errors.New("batto: callFn is nil")
	}

	if err := ctx.Err(); err != nil {
		var zero V
		return zero, err
	}

	group.mu.Lock()
	if group.calls == nil {
		group.calls = make(map[K]*call[V])
	}

	call, ok := group.calls[key]
	if !ok {
		// Current call is the initiator for this key.
		call, ch := group.startWorker(key, callFn)
		return group.wait(ctx, key, call, ch)
	}

	call.mu.Lock()
	if call.done {
		// The call finished just as we were locking group.mu and c.mu.
		// Extract values and return immediately without enqueuing.
		val, err, panicVal := call.val, call.err, call.panicVal
		call.mu.Unlock()
		group.mu.Unlock()

		if panicVal != nil {
			var zero V
			return zero, ErrWorkerPanicked
		}

		return val, err
	}

	// Register a new waiting channel for this duplicate concurrent call.
	ch := make(chan Result[V], 1)
	call.waiters[ch] = ctx
	call.mu.Unlock()
	group.mu.Unlock()

	return group.wait(ctx, key, call, ch)
}

// startWorker initializes the call structure, registers it in the group,
// and spawns a background goroutine to execute the task.
//
// This method must be called under group.mu locked, and it unlocks it before returning.
func (group *Group[K, V]) startWorker(key K, fn CallFn[V]) (*call[V], chan Result[V]) {
	ctx, cancel := context.WithCancel(context.Background()) //nolint:gosec

	initCh := make(chan Result[V], 1)
	c := &call[V]{
		cancel:    cancel,
		waiters:   make(map[chan<- Result[V]]context.Context),
		initiator: initCh,
	}
	c.waiters[initCh] = ctx
	group.calls[key] = c
	group.mu.Unlock()

	go group.run(ctx, key, c, fn)

	return c, initCh
}

// wait blocks until the call completes or the caller's context is cancelled.
func (group *Group[K, V]) wait(ctx context.Context, key K, c *call[V], resCh chan Result[V]) (V, error) {
	select {
	case res := <-resCh:
		if res.PanicVal != nil {
			panic(res.PanicVal)
		}

		return res.Val, res.Err

	case <-ctx.Done():
		// The caller's context was cancelled or expired.
		// Acquire locks in strict hierarchical order: group.mu -> c.mu.
		group.mu.Lock()
		c.mu.Lock()

		if c.done {
			// The task completed just before the cancellation was processed.
			// Prioritize the actual computed result.
			c.mu.Unlock()
			group.mu.Unlock()

			res := <-resCh
			if res.PanicVal != nil {
				panic(res.PanicVal)
			}

			return res.Val, res.Err
		}

		// Remove the caller's channel from the active waiters.
		delete(c.waiters, resCh)

		// If no other callers are waiting, cancel the worker's context
		// and remove the call from the active group to reclaim resources.
		if len(c.waiters) == 0 {
			c.cancel()
			delete(group.calls, key)
		}

		c.mu.Unlock()
		group.mu.Unlock()

		var zero V

		return zero, ctx.Err()
	}
}

// run executes the worker function and distributes the result to all waiters.
func (group *Group[K, V]) run(ctx context.Context, key K, c *call[V], fn CallFn[V]) {
	var (
		val V
		err error
	)

	defer func() {
		if r := recover(); r != nil {
			group.mu.Lock()
			c.mu.Lock()

			c.done = true
			c.panicVal = r

			delete(group.calls, key)

			for ch := range c.waiters {
				if ch == c.initiator {
					ch <- Result[V]{PanicVal: r}
				} else {
					ch <- Result[V]{Err: ErrWorkerPanicked}
				}
			}

			c.mu.Unlock()
			group.mu.Unlock()
		}
	}()

	val, err = fn(ctx)

	group.mu.Lock()
	c.mu.Lock()

	c.done = true
	c.val = val
	c.err = err

	delete(group.calls, key)

	for ch := range c.waiters {
		ch <- Result[V]{Val: val, Err: err}
	}

	c.mu.Unlock()
	group.mu.Unlock()
}
