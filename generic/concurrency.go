// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generic

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"sync"
	"time"
)

// ParallelMap concurrently transforms each element of a slice using fn,
// limiting the number of active goroutines to the specified limit.
//
// If the provided context is cancelled during execution, ParallelMap returns nil immediately.
func ParallelMap[F, T any](ctx context.Context, slice []F, limit int, fn func(context.Context, F) T) []T {
	if fn == nil || len(slice) == 0 {
		return nil
	}
	if ctx != nil && ctx.Err() != nil {
		return nil
	}

	if limit <= 0 {
		limit = 1
	}

	res := make([]T, len(slice))
	sem := make(chan struct{}, limit)

	var wg sync.WaitGroup

	for i, v := range slice {
		if ctx != nil && ctx.Err() != nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return nil
		case sem <- struct{}{}:
		}

		wg.Add(1)

		go func(idx int, val F) {
			defer wg.Done()
			defer func() { <-sem }()

			res[idx] = fn(ctx, val)
		}(i, v)
	}

	wg.Wait()

	if ctx != nil && ctx.Err() != nil {
		return nil
	}

	return res
}

// ParallelForEach executes the side-effect function fn on each slice element concurrently,
// limiting the number of active goroutines to the specified limit.
//
// It returns the first non-nil error encountered during execution.
// If the provided context is cancelled, it returns the context error.
func ParallelForEach[T any](ctx context.Context, slice []T, limit int, fn func(context.Context, T) error) error {
	if fn == nil || len(slice) == 0 {
		return nil
	}
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}

	if limit <= 0 {
		limit = 1
	}

	sem := make(chan struct{}, limit)
	errs := make(chan error, len(slice))

	var wg sync.WaitGroup

	for _, v := range slice {
		if ctx != nil && ctx.Err() != nil {
			return ctx.Err()
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case sem <- struct{}{}:
		}

		wg.Add(1)

		go func(val T) {
			defer wg.Done()
			defer func() { <-sem }()

			if err := fn(ctx, val); err != nil {
				select {
				case errs <- err:
				default:
				}
			}
		}(v)
	}

	wg.Wait()
	close(errs)

	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}

	return <-errs
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

// NewFutureFunc runs the provided function fn asynchronously in a new goroutine
// and returns a [Future] representing its deferred result.
func NewFutureFunc[T any](fn func() (T, error)) *Future[T] {
	f := NewFuture[T]()
	if fn == nil {
		f.Set(*new(T), errors.New("future: fn is nil"))
		return f
	}

	go func() {
		val, err := fn()
		f.Set(val, err)
	}()

	return f
}

// NewFutureContext runs the provided function fn asynchronously in a new goroutine,
// passing it the provided context, and returns a [Future] representing its deferred result.
func NewFutureContext[T any](ctx context.Context, fn func(context.Context) (T, error)) *Future[T] {
	f := NewFuture[T]()
	if fn == nil {
		f.Set(*new(T), errors.New("future: fn is nil"))
		return f
	}

	go func() {
		val, err := fn(ctx)
		f.Set(val, err)
	}()

	return f
}

// Get blocks the calling goroutine until the associated task completes execution,
// or until the provided context is cancelled or expires.
//
// If the context expires, Get returns the zero value of type T and the context error.
func (f *Future[T]) Get(ctx context.Context) (T, error) {
	if f == nil {
		var zero T
		return zero, errors.New("future is nil")
	}

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
	if f == nil {
		return
	}

	f.once.Do(func() {
		f.val = val
		f.err = err
		close(f.ch)
	})
}

type sfCall[T any] struct {
	wg  sync.WaitGroup
	val T
	err error
}

// SingleFlight prevents duplicate concurrent executions of identical tasks (De-duplicator).
//
// It ensures that concurrent requests for the same key execute the payload function only once,
// sharing the resulting value and error. SingleFlight is safe for concurrent use.
type SingleFlight[T any] struct {
	mu sync.Mutex
	m  map[string]*sfCall[T]
}

// NewSingleFlight creates a new [SingleFlight] instance.
func NewSingleFlight[T any]() *SingleFlight[T] {
	return &SingleFlight[T]{m: make(map[string]*sfCall[T])}
}

// Do executes the function fn for the given key, suppressing duplicate concurrent calls.
func (g *SingleFlight[T]) Do(key string, fn func() (T, error)) (T, error) {
	if g == nil {
		var zero T
		return zero, errors.New("singleflight is nil")
	}

	if fn == nil {
		var zero T
		return zero, errors.New("singleflight fn is nil")
	}

	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*sfCall[T])
	}

	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}

	c := new(sfCall[T])
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	c.val, c.err = fn()
	c.wg.Done()

	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	return c.val, c.err
}

// RetryConfig defines the parameters for generic execution retries.
type RetryConfig struct {
	Attempts int
	Delay    time.Duration
}

// Retry executes the function fn up to config.Attempts times if it returns an error.
// It respects context cancellation between and during attempts.
func Retry(ctx context.Context, cfg RetryConfig, fn func(ctx context.Context) error) error {
	if fn == nil {
		return errors.New("retry fn is nil")
	}

	if cfg.Attempts <= 0 {
		cfg.Attempts = 1
	}

	var err error
	for i := 0; i < cfg.Attempts; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err = fn(ctx)
		if err == nil {
			return nil
		}

		if i+1 < cfg.Attempts && cfg.Delay > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(cfg.Delay):
			}
		}
	}

	return err
}

// Backoff implements a thread-safe exponential backoff with randomized jitter (AWS Full Jitter algorithm).
type Backoff struct {
	mu       sync.Mutex
	min      time.Duration
	max      time.Duration
	factor   float64
	jitter   float64
	attempts int
}

// NewBackoff creates and initializes a new [Backoff] instance.
// If factor is less than or equal to 0, a default factor of 2.0 is applied.
func NewBackoff(min, max time.Duration, factor, jitter float64) *Backoff {
	if factor <= 0 {
		factor = 2
	}

	return &Backoff{
		min:    min,
		max:    max,
		factor: factor,
		jitter: jitter,
	}
}

// Next returns the delay duration for the next attempt, incrementing the internal
// attempt counter. Next is safe for concurrent calls by multiple goroutines.
func (b *Backoff) Next() time.Duration {
	if b == nil {
		return 0
	}

	b.mu.Lock()
	attempts := b.attempts
	b.attempts++
	b.mu.Unlock()

	ms := float64(b.min.Milliseconds()) * math.Pow(b.factor, float64(attempts))
	if b.jitter > 0 {
		deviation := math.Floor(rand.Float64() * b.jitter * ms) //nolint:gosec
		if rand.IntN(2) == 0 {                                  //nolint:gosec
			ms -= deviation
		} else {
			ms += deviation
		}
	}

	if ms > float64(b.max.Milliseconds()) {
		ms = float64(b.max.Milliseconds())
	}

	if ms < 0 {
		ms = 0
	}

	return time.Duration(ms) * time.Millisecond
}

// Reset resets the attempt counter of the backoff back to zero.
func (b *Backoff) Reset() {
	if b == nil {
		return
	}

	b.mu.Lock()
	b.attempts = 0
	b.mu.Unlock()
}

type loaderResult[V any] struct {
	val V
	err error
}

// DataLoader aggregates multiple individual requests by key K into a single,
// batched call, returning results of type V.
//
// It is fully thread-safe and prevents "N+1" query problems in data fetching.
type DataLoader[K comparable, V any] struct {
	mu      sync.Mutex
	delay   time.Duration
	batchFn func(context.Context, []K) (map[K]V, error)
	pending map[K][]chan loaderResult[V]
	timer   *time.Timer
}

// NewDataLoader creates and returns a new [DataLoader] with the given batch delay window and batch function.
func NewDataLoader[K comparable, V any](
	delay time.Duration,
	batchFn func(context.Context, []K) (map[K]V, error),
) *DataLoader[K, V] {
	return &DataLoader[K, V]{
		delay:   delay,
		batchFn: batchFn,
		pending: make(map[K][]chan loaderResult[V]),
	}
}

// Load loads the value for the given key K.
//
// If other concurrent Load calls are received within the configured delay window,
// they are accumulated and dispatched as a single batch call to batchFn.
func (l *DataLoader[K, V]) Load(ctx context.Context, key K) (V, error) {
	if l == nil {
		var zero V
		return zero, errors.New("dataloader is nil")
	}

	if l.batchFn == nil {
		var zero V
		return zero, errors.New("dataloader batchFn is nil")
	}

	ch := make(chan loaderResult[V], 1)

	l.mu.Lock()
	if l.pending == nil {
		l.pending = make(map[K][]chan loaderResult[V])
	}

	l.pending[key] = append(l.pending[key], ch)

	if l.timer == nil {
		l.timer = time.AfterFunc(l.delay, func() {
			l.executeBatch()
		})
	}

	l.mu.Unlock()

	select {
	case <-ctx.Done():
		var zero V
		return zero, ctx.Err()
	case res := <-ch:
		return res.val, res.err
	}
}

func (l *DataLoader[K, V]) executeBatch() {
	if l == nil {
		return
	}

	l.mu.Lock()
	pending := l.pending
	l.pending = make(map[K][]chan loaderResult[V])
	l.timer = nil
	l.mu.Unlock()

	if len(pending) == 0 {
		return
	}

	keys := make([]K, 0, len(pending))
	for k := range pending {
		keys = append(keys, k)
	}

	results, err := l.batchFn(context.Background(), keys)

	for _, k := range keys {
		chans := pending[k]

		var (
			val     V
			itemErr error
		)

		if err != nil {
			itemErr = err
		} else if v, ok := results[k]; ok {
			val = v
		} else {
			itemErr = errors.New("foundation dataloader: key not found in batch results")
		}

		for _, ch := range chans {
			ch <- loaderResult[V]{val: val, err: itemErr}
		}
	}
}

// TaskGroup coordinates a group of concurrent tasks that yield a value of type T
// and can fail with an error. It provides structured concurrency similar to Swift's TaskGroup.
// If any task fails or panics, the Group's context is cancelled.
type TaskGroup[T any] struct {
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	mu      sync.Mutex
	results []Result[T]
}

// NewTaskGroup creates a new [TaskGroup] linked to the provided context.
func NewTaskGroup[T any](ctx context.Context) *TaskGroup[T] {
	cCtx, cancel := context.WithCancel(ctx) //nolint:gosec

	return &TaskGroup[T]{
		ctx:    cCtx,
		cancel: cancel,
	}
}

// Add runs the provided function fn concurrently in the group.
// If the task returns a Failure or panics, the Group's context is cancelled.
func (tg *TaskGroup[T]) Add(fn func(ctx context.Context) Result[T]) {
	if tg == nil || fn == nil {
		return
	}

	tg.wg.Go(func() {
		// Handle panic gracefully and convert it to error
		defer func() {
			if r := recover(); r != nil {
				tg.cancel()
				tg.mu.Lock()
				tg.results = append(tg.results, Failure[T](errors.New("task panicked")))
				tg.mu.Unlock()
			}
		}()

		res := fn(tg.ctx)
		if !res.IsSuccess() {
			tg.cancel() // Cancel context of all other tasks in the group on first failure
		}

		tg.mu.Lock()
		tg.results = append(tg.results, res)
		tg.mu.Unlock()
	})
}

// Wait blocks until all tasks in the group have finished and returns their results.
func (tg *TaskGroup[T]) Wait() []Result[T] {
	if tg == nil {
		return nil
	}

	tg.wg.Wait()
	tg.cancel() // Ensure resources are cleaned up
	tg.mu.Lock()
	defer tg.mu.Unlock()

	return tg.results
}

// RaceFirstSuccess executes multiple task functions concurrently and returns the first
// [Result] that represents a success (r.IsSuccess() is true).
//
// Once the first successful result is obtained, all other ongoing tasks are immediately
// cancelled via their execution context.
//
// If all tasks fail, RaceFirstSuccess returns the error of the last failed task.
// If tasks is empty, it returns a Failure with an error.
func RaceFirstSuccess[T any](ctx context.Context, tasks ...func(context.Context) Result[T]) Result[T] {
	if len(tasks) == 0 {
		return Failure[T](errors.New("generic: no tasks to race"))
	}

	raceCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	resCh := make(chan Result[T], len(tasks))
	var wg sync.WaitGroup

	for _, task := range tasks {
		if task == nil {
			continue
		}

		wg.Add(1)
		go func(fn func(context.Context) Result[T]) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					resCh <- Failure[T](fmt.Errorf("generic: task panic: %v", r))
				}
			}()

			res := fn(raceCtx)
			resCh <- res
			if res.IsSuccess() {
				cancel()
			}
		}(task)
	}

	go func() {
		wg.Wait()
		close(resCh)
	}()

	var lastErr error
	for res := range resCh {
		if res.IsSuccess() {
			return res
		}
		_, err := res.Unwrap()
		lastErr = err
	}

	if lastErr == nil {
		lastErr = ctx.Err()
	}

	return Failure[T](lastErr)
}
