// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dedup

import (
	"context"
	"errors"
	"iter"
	"sync"
)

// ErrWorkerPanicked is returned to secondary waiting callers
// if the initiating worker goroutine terminates due to a panic.
var ErrWorkerPanicked = errors.New("worker panicked during execution")

// Result represents the deferred execution outcome of a [CallFn] worker.
//
// The zero value is empty and ready to use.
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

// Group coordinates concurrent executions and suppresses duplicate calls for identical keys.
//
// The zero value is valid and ready for use.
// All exported methods are safe for concurrent use and manage their own internal synchronization.
type Group[K comparable, V any] struct {
	mu    sync.Mutex
	calls map[K]*call[V]
}

// KeysSeq returns an iterator yielding the keys of currently active deduplication calls.
//
// Complexity: O(N) time where N is the number of active calls to initialize the sequence; allocates memory for the snapshot.
func (g *Group[K, V]) KeysSeq() iter.Seq[K] {
	return func(yield func(K) bool) {
		if g == nil {
			return
		}
		g.mu.Lock()
		if len(g.calls) == 0 {
			g.mu.Unlock()
			return
		}
		keys := make([]K, 0, len(g.calls))
		for k := range g.calls {
			keys = append(keys, k)
		}
		g.mu.Unlock()

		for _, k := range keys {
			if !yield(k) {
				return
			}
		}
	}
}

// InFlight returns the number of currently active deduplication calls.
//
// Complexity: O(1) time; zero-allocation.
func (g *Group[K, V]) InFlight() int {
	if g == nil {
		return 0
	}
	g.mu.Lock()
	n := len(g.calls)
	g.mu.Unlock()
	return n
}

// Len returns the number of currently active deduplication calls.
//
// Complexity: O(1) time; zero-allocation.
func (g *Group[K, V]) Len() int {
	return g.InFlight()
}

// CallFn represents a context-aware generic function executed by a [Group].
type CallFn[V any] func(ctx context.Context) (V, error)

// Do executes callFn and suppresses duplicate concurrent calls for the specified key.
//
// If an execution for the key is already in progress, it blocks until completion and returns the shared result.
// If the provided context is canceled while waiting, it returns immediately with the context error.
// If all concurrent contexts waiting on the same call are canceled, the underlying worker's context is canceled.
// If callFn panics, the panic is propagated to the initiating caller, while secondary waiting callers receive [ErrWorkerPanicked].
//
// Complexity: O(1) time excluding callFn execution; allocates memory when initiating or joining a call.
func (g *Group[K, V]) Do(ctx context.Context, key K, callFn CallFn[V]) (V, error) {
	if g == nil {
		var zero V
		return zero, errors.New("dedup: group is nil")
	}

	if callFn == nil {
		var zero V
		return zero, errors.New("dedup: callFn is nil")
	}

	if err := ctx.Err(); err != nil {
		var zero V
		return zero, err
	}

	g.mu.Lock()
	if g.calls == nil {
		g.calls = make(map[K]*call[V])
	}

	c, ok := g.calls[key]
	if !ok {
		// Current call is the initiator for this key.
		c, ch := g.startWorker(key, callFn)
		return g.wait(ctx, key, c, ch)
	}

	c.mu.Lock()
	if c.done {
		// The call finished just as we were locking group.mu and c.mu.
		// Extract values and return immediately without enqueuing.
		val, err, panicVal := c.val, c.err, c.panicVal
		c.mu.Unlock()
		g.mu.Unlock()

		if panicVal != nil {
			var zero V
			return zero, ErrWorkerPanicked
		}

		return val, err
	}

	// Register a new waiting channel for this duplicate concurrent call.
	ch := make(chan Result[V], 1)
	c.waiters[ch] = ctx
	c.mu.Unlock()
	g.mu.Unlock()

	return g.wait(ctx, key, c, ch)
}

// DoChan executes callFn and suppresses duplicate concurrent calls for the specified key, returning a channel that yields the [Result].
//
// Complexity: O(1) time excluding callFn execution; allocates memory for channels and call tracking.
func (g *Group[K, V]) DoChan(ctx context.Context, key K, callFn CallFn[V]) <-chan Result[V] {
	ch := make(chan Result[V], 1)

	if g == nil {
		ch <- Result[V]{Err: errors.New("dedup: group is nil")}
		return ch
	}

	if callFn == nil {
		ch <- Result[V]{Err: errors.New("dedup: callFn is nil")}
		return ch
	}

	if err := ctx.Err(); err != nil {
		ch <- Result[V]{Err: err}
		return ch
	}

	g.mu.Lock()
	if g.calls == nil {
		g.calls = make(map[K]*call[V])
	}

	c, ok := g.calls[key]
	if !ok {
		c, initCh := g.startWorker(key, callFn)
		go func() {
			val, err := g.wait(ctx, key, c, initCh)
			ch <- Result[V]{Val: val, Err: err}
		}()
		return ch
	}

	c.mu.Lock()
	if c.done {
		val, err, panicVal := c.val, c.err, c.panicVal
		c.mu.Unlock()
		g.mu.Unlock()

		if panicVal != nil {
			ch <- Result[V]{Err: ErrWorkerPanicked, PanicVal: panicVal}
		} else {
			ch <- Result[V]{Val: val, Err: err}
		}
		return ch
	}

	waiterCh := make(chan Result[V], 1)
	c.waiters[waiterCh] = ctx
	c.mu.Unlock()
	g.mu.Unlock()

	go func() {
		val, err := g.wait(ctx, key, c, waiterCh)
		ch <- Result[V]{Val: val, Err: err}
	}()

	return ch
}

// startWorker initializes the call structure, registers it in the group,
// and spawns a background goroutine to execute the task.
//
// This method must be called under group.mu locked, and it unlocks it before returning.
func (g *Group[K, V]) startWorker(key K, fn CallFn[V]) (*call[V], chan Result[V]) {
	ctx, cancel := context.WithCancel(context.Background()) //nolint:gosec

	initCh := make(chan Result[V], 1)
	c := &call[V]{
		cancel:    cancel,
		waiters:   make(map[chan<- Result[V]]context.Context),
		initiator: initCh,
	}
	c.waiters[initCh] = ctx
	g.calls[key] = c
	g.mu.Unlock()

	go g.run(ctx, key, c, fn)

	return c, initCh
}

// wait blocks until the call completes or the caller's context is cancelled.
func (g *Group[K, V]) wait(ctx context.Context, key K, c *call[V], resCh chan Result[V]) (V, error) {
	select {
	case res := <-resCh:
		if res.PanicVal != nil {
			panic(res.PanicVal)
		}

		return res.Val, res.Err

	case <-ctx.Done():
		// The caller's context was cancelled or expired.
		// Acquire locks in strict hierarchical order: group.mu -> c.mu.
		g.mu.Lock()
		c.mu.Lock()

		if c.done {
			// The task completed just before the cancellation was processed.
			// Prioritize the actual computed result.
			c.mu.Unlock()
			g.mu.Unlock()

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
			delete(g.calls, key)
		}

		c.mu.Unlock()
		g.mu.Unlock()

		var zero V

		return zero, ctx.Err()
	}
}

// run executes the worker function and distributes the result to all waiters.
func (g *Group[K, V]) run(ctx context.Context, key K, c *call[V], fn CallFn[V]) {
	var (
		val V
		err error
	)

	defer func() {
		if r := recover(); r != nil {
			g.mu.Lock()
			c.mu.Lock()

			c.done = true
			c.panicVal = r

			if g.calls[key] == c {
				delete(g.calls, key)
			}

			initiatorPresent := false
			for ch := range c.waiters {
				if ch == c.initiator {
					initiatorPresent = true
					ch <- Result[V]{PanicVal: r}
				} else {
					ch <- Result[V]{Err: ErrWorkerPanicked}
				}
			}

			c.mu.Unlock()
			g.mu.Unlock()

			if !initiatorPresent {
				panic(r)
			}
		}
	}()

	val, err = fn(ctx)

	g.mu.Lock()
	c.mu.Lock()

	c.done = true
	c.val = val
	c.err = err

	if g.calls[key] == c {
		delete(g.calls, key)
	}

	for ch := range c.waiters {
		ch <- Result[V]{Val: val, Err: err}
	}

	c.mu.Unlock()
	g.mu.Unlock()
}
