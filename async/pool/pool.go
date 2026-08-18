// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pool

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	// ErrPoolClosed is returned by [Pool.Submit] when attempting to enqueue
	// a task to a pool that has already been closed.
	ErrPoolClosed = errors.New("worker pool is closed")

	// ErrQueueFull is returned by [Pool.Submit] when the task queue reaches
	// its configured capacity limit and cannot accept more tasks.
	ErrQueueFull = errors.New("worker pool queue is full")
)

// task represents the internal payload containing the work function,
// its execution context, and the future associated with its result.
type task[T any] struct {
	ctx    context.Context
	work   func(context.Context) (T, error)
	future *Future[T]
}

// Config defines the scaling boundaries, queue limits, and idle timeouts
// for a [Pool].
//
// The zero value of Config is not directly usable. Call [Config.ResolveDefaults]
// to apply safe fallback values, or instantiate the pool via [New], which
// resolves defaults automatically.
type Config struct {
	// MinWorkers is the baseline number of workers that always remain active.
	MinWorkers int
	// MaxWorkers is the absolute upper bound on the number of concurrently running workers.
	MaxWorkers int
	// IdleTimeout is the duration an idle worker (above MinWorkers) waits before shutting down.
	IdleTimeout time.Duration
	// QueueLimit is the maximum number of pending tasks allowed in the queue.
	QueueLimit int
}

// ResolveDefaults validates the configuration boundaries and applies safe fallback
// defaults for any non-positive or conflicting fields.
//
// It ensures that:
//   - MinWorkers is at least 1.
//   - MaxWorkers is at least equal to MinWorkers.
//   - IdleTimeout is at least 5 seconds if unspecified or non-positive.
//   - QueueLimit is at least 100 if unspecified or non-positive.
func (c *Config) ResolveDefaults() {
	if c.MinWorkers <= 0 {
		c.MinWorkers = 1
	}

	if c.MaxWorkers < c.MinWorkers {
		c.MaxWorkers = c.MinWorkers
	}

	if c.IdleTimeout <= 0 {
		c.IdleTimeout = 5 * time.Second
	}

	if c.QueueLimit <= 0 {
		c.QueueLimit = 100
	}
}

// Pool manages a set of worker goroutines that scale dynamically under load.
//
// It maintains a minimum number of idle workers, scales up to a maximum limit
// when tasks accumulate in the queue, and automatically prunes idle workers
// after a configured timeout.
//
// A Pool is safe for concurrent use by multiple goroutines.
type Pool[T any] struct {
	mu             sync.Mutex
	cfg            Config
	tasks          chan task[T]
	currentWorkers int
	busyWorkers    int
	closed         bool
	wg             sync.WaitGroup
}

// New creates, initializes, and starts a worker Pool with the provided configuration.
//
// It immediately spawns the configured minimum number of worker goroutines (MinWorkers).
// New automatically resolves any zero or invalid configuration fields by calling
// [Config.ResolveDefaults].
func New[T any](cfg Config) *Pool[T] {
	cfg.ResolveDefaults()

	p := &Pool[T]{
		cfg:   cfg,
		tasks: make(chan task[T], cfg.QueueLimit),
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.currentWorkers = cfg.MinWorkers
	for range cfg.MinWorkers {
		p.spawnWorker()
	}

	return p
}

// spawnWorker starts an asynchronous worker goroutine.
//
// The caller must increment pool.currentWorkers under pool.mu protection
// before calling this method.
func (pool *Pool[T]) spawnWorker() {
	pool.wg.Go(func() {
		timer := time.NewTimer(pool.cfg.IdleTimeout)
		defer timer.Stop()

		for {
			// Safely drain and reset the timer before reuse.
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}

			timer.Reset(pool.cfg.IdleTimeout)

			select {
			case t, ok := <-pool.tasks:
				if !ok {
					pool.mu.Lock()
					pool.currentWorkers--
					pool.mu.Unlock()

					return
				}

				pool.runTask(t)

			case <-timer.C:
				// Worker has been idle for IdleTimeout. Decide if we should scale down.
				pool.mu.Lock()
				if pool.currentWorkers > pool.cfg.MinWorkers {
					pool.currentWorkers--
					pool.mu.Unlock()
					return
				}

				pool.mu.Unlock()
			}
		}
	})
}

// runTask executes the work payload of a task, capturing panics and updating metrics.
func (pool *Pool[T]) runTask(t task[T]) {
	pool.mu.Lock()
	pool.busyWorkers++
	pool.mu.Unlock()

	var (
		val T
		err error
	)

	defer func() {
		pool.mu.Lock()
		pool.busyWorkers--
		pool.mu.Unlock()

		if r := recover(); r != nil {
			err = fmt.Errorf("task panicked: %v", r)
		}

		t.future.Set(val, err)
	}()

	val, err = t.work(t.ctx)
}

// Submit enqueues a task for asynchronous execution by an idle worker.
//
// It returns a [Future] to await the task result, or an error if the pool is closed
// ([ErrPoolClosed]) or its task queue is full ([ErrQueueFull]).
//
// Submit dynamically evaluates the load on the pool: if all current workers are busy
// or tasks are queuing up, and the current worker count is below [Config.MaxWorkers], it
// spawns a new worker goroutine.
func (pool *Pool[T]) Submit(ctx context.Context, fn func(context.Context) (T, error)) (*Future[T], error) {
	pool.mu.Lock()

	if pool.closed {
		pool.mu.Unlock()
		return nil, ErrPoolClosed
	}

	if len(pool.tasks) >= pool.cfg.QueueLimit {
		pool.mu.Unlock()
		return nil, ErrQueueFull
	}

	future := NewFuture[T]()

	select {
	case pool.tasks <- task[T]{ctx, fn, future}:
	default:
		pool.mu.Unlock()
		return nil, ErrQueueFull
	}

	// Safely evaluate worker scaling under mutex protection.
	shouldSpawn := false
	if pool.currentWorkers < pool.cfg.MaxWorkers && (pool.busyWorkers == pool.currentWorkers || len(pool.tasks) > 0) {
		pool.currentWorkers++
		shouldSpawn = true
	}

	pool.mu.Unlock()

	if shouldSpawn {
		pool.spawnWorker()
	}

	return future, nil
}

// Close initiates a graceful shutdown of the pool.
//
// Close marks the pool as closed, preventing any subsequent tasks from being
// submitted, and closes the internal task queue. It blocks the calling goroutine
// until all currently queued and executing tasks are completed and all worker
// goroutines have exited.
//
// Calling Close on an already closed pool returns nil immediately.
func (pool *Pool[T]) Close() error {
	pool.mu.Lock()
	if pool.closed {
		pool.mu.Unlock()
		return nil
	}

	pool.closed = true
	close(pool.tasks)
	pool.mu.Unlock()

	pool.wg.Wait()

	return nil
}

// Future represents an asynchronous, deferred execution (a Promise pattern).
// It is safe for concurrent use by multiple goroutines.
type Future[T any] struct {
	ch   chan struct{}
	val  T
	err  error
	once sync.Once
}

// NewFuture returns a new, uncompleted [Future] instance.
func NewFuture[T any]() *Future[T] {
	return &Future[T]{
		ch: make(chan struct{}),
	}
}

// Get blocks the calling goroutine until the associated task completes execution,
// or until the provided context is cancelled or expires.
//
// If the context expires, Get returns the zero value of type T and the context error.
func (f *Future[T]) Get(ctx context.Context) (T, error) {
	select {
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	case <-f.ch:
		return f.val, f.err
	}
}

// Set completes the [Future] by storing the computed value and error,
// and unblocks any goroutines waiting on Get.
//
// Set is thread-safe and idempotent; only the first call is executed.
func (f *Future[T]) Set(val T, err error) {
	f.once.Do(func() {
		f.val = val
		f.err = err
		close(f.ch)
	})
}
