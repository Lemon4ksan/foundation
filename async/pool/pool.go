// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pool

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"sync"
	"time"
)

var (
	// ErrClosed is returned by [Pool.Submit] when attempting to enqueue a task to a pool that has already been closed.
	ErrClosed = errors.New("worker pool is closed")

	// ErrQueueFull is returned by [Pool.Submit] when the task queue reaches its configured capacity limit and cannot accept more tasks.
	ErrQueueFull = errors.New("worker pool queue is full")
)

// task represents the internal payload containing the work function,
// its execution context, and the future associated with its result.
type task[T any] struct {
	ctx    context.Context
	work   func(context.Context) (T, error)
	future *Future[T]
}

// Config defines the scaling boundaries, queue limits, and idle timeouts for a [Pool].
//
// The zero value is empty and not directly usable.
// Call [Config.ResolveDefaults] to apply safe fallback values, or instantiate the pool via [New].
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
// Methods on [Pool] are safe for concurrent use.
// It allocates memory for tracking workers and tasks.
type Pool[T any] struct {
	mu             sync.Mutex
	cfg            Config
	tasks          chan task[T]
	currentWorkers int
	busyWorkers    int
	closed         bool
	wg             sync.WaitGroup
}

// New creates, initializes, and starts a worker [Pool] with the provided configuration.
//
// It immediately spawns the configured minimum number of worker goroutines.
// It automatically resolves any zero or invalid configuration fields by calling [Config.ResolveDefaults].
// It allocates memory for the pool and its internal channels.
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
func (p *Pool[T]) spawnWorker() {
	p.wg.Go(func() {
		timer := time.NewTimer(p.cfg.IdleTimeout)
		defer timer.Stop()

		for {
			// Safely drain and reset the timer before reuse.
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}

			timer.Reset(p.cfg.IdleTimeout)

			select {
			case t, ok := <-p.tasks:
				if !ok {
					p.mu.Lock()
					p.currentWorkers--
					p.mu.Unlock()

					return
				}

				p.runTask(t)

			case <-timer.C:
				// Worker has been idle for IdleTimeout. Decide if we should scale down.
				p.mu.Lock()
				if p.currentWorkers > p.cfg.MinWorkers {
					p.currentWorkers--
					p.mu.Unlock()
					return
				}

				p.mu.Unlock()
			}
		}
	})
}

// runTask executes the work payload of a task, capturing panics and updating metrics.
func (p *Pool[T]) runTask(t task[T]) {
	p.mu.Lock()
	p.busyWorkers++
	p.mu.Unlock()

	var (
		val T
		err error
	)

	defer func() {
		p.mu.Lock()
		p.busyWorkers--
		p.mu.Unlock()

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
// ([ErrClosed]) or its task queue is full ([ErrQueueFull]).
//
// Submit dynamically evaluates the load on the pool: if all current workers are busy
// or tasks are queuing up, and the current worker count is below [Config.MaxWorkers], it
// spawns a new worker goroutine.
func (p *Pool[T]) Submit(ctx context.Context, fn func(context.Context) (T, error)) (*Future[T], error) {
	p.mu.Lock()

	if p.closed {
		p.mu.Unlock()
		return nil, ErrClosed
	}

	if len(p.tasks) >= p.cfg.QueueLimit {
		p.mu.Unlock()
		return nil, ErrQueueFull
	}

	future := NewFuture[T]()

	select {
	case p.tasks <- task[T]{ctx, fn, future}:
	default:
		p.mu.Unlock()
		return nil, ErrQueueFull
	}

	// Safely evaluate worker scaling under mutex protection.
	shouldSpawn := false
	if p.currentWorkers < p.cfg.MaxWorkers && (p.busyWorkers == p.currentWorkers || len(p.tasks) > 0) {
		p.currentWorkers++
		shouldSpawn = true
	}

	p.mu.Unlock()

	if shouldSpawn {
		p.spawnWorker()
	}

	return future, nil
}

// ItemsSeq executes a sequence of tasks using the pool and yields results as they complete.
//
// Iteration stops immediately if the yield function returns false or if the context is cancelled.
// It is safe for concurrent use.
// It allocates memory for internal tracking and channels.
func (p *Pool[T]) ItemsSeq(
	ctx context.Context,
	tasks iter.Seq[func(context.Context) (T, error)],
) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		type resItem struct {
			val T
			err error
		}

		runCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		resultsCh := make(chan resItem, p.cfg.QueueLimit)
		var futWg sync.WaitGroup

		go func() {
			for taskFn := range tasks {
				select {
				case <-runCtx.Done():
					futWg.Wait()
					close(resultsCh)
					return
				default:
				}

				fut, err := p.Submit(runCtx, taskFn)
				if err != nil {
					var zero T
					select {
					case <-runCtx.Done():
						futWg.Wait()
						close(resultsCh)
						return
					case resultsCh <- resItem{val: zero, err: err}:
						continue
					}
				}

				futWg.Add(1)
				go func(f *Future[T]) {
					defer futWg.Done()
					val, fErr := f.Get(runCtx)
					select {
					case <-runCtx.Done():
					case resultsCh <- resItem{val: val, err: fErr}:
					}
				}(fut)
			}

			futWg.Wait()
			close(resultsCh)
		}()

		for res := range resultsCh {
			if !yield(res.val, res.err) {
				cancel()
				break
			}
		}
	}
}

// ItemsSeq submits tasks from the input iterator to the provided [Pool] and yields results as they complete.
//
// It is safe for concurrent use.
func ItemsSeq[T any](
	ctx context.Context,
	pool *Pool[T],
	tasks iter.Seq[func(context.Context) (T, error)],
) iter.Seq2[T, error] {
	return pool.ItemsSeq(ctx, tasks)
}

// Close initiates a graceful shutdown of the pool.
//
// Close marks the pool as closed, preventing any subsequent tasks from being
// submitted, and closes the internal task queue. It blocks the calling goroutine
// until all currently queued and executing tasks are completed and all worker
// goroutines have exited.
//
// Calling Close on an already closed pool returns nil immediately.
func (p *Pool[T]) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}

	p.closed = true
	close(p.tasks)
	p.mu.Unlock()

	p.wg.Wait()

	return nil
}

// Future represents an asynchronous, deferred execution.
//
// The zero value is empty and not ready to use.
// It is safe for concurrent use.
type Future[T any] struct {
	ch   chan struct{}
	val  T
	err  error
	once sync.Once
}

// NewFuture returns a new, uncompleted [Future] instance.
// It allocates memory for the channel.
func NewFuture[T any]() *Future[T] {
	return &Future[T]{
		ch: make(chan struct{}),
	}
}

// Get blocks the calling goroutine until the associated task completes execution, or until the provided context is cancelled.
//
// It returns the zero value and the context error if the context expires.
// It is safe for concurrent use.
func (f *Future[T]) Get(ctx context.Context) (T, error) {
	select {
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	case <-f.ch:
		return f.val, f.err
	}
}

// Set completes the [Future] by storing the computed value and error, and unblocks any goroutines waiting on Get.
//
// It is safe for concurrent use.
func (f *Future[T]) Set(val T, err error) {
	f.once.Do(func() {
		f.val = val
		f.err = err
		close(f.ch)
	})
}
