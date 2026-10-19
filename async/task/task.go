// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package task

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

var (
	// ErrTimeout is returned when a task exceeds its allowed execution time defined by [WithTimeout].
	ErrTimeout = errors.New("task: request timed out")

	// ErrClosed is returned when the tracker is shutting down and all pending tasks are being cancelled.
	ErrClosed = errors.New("task: tracker closed")

	// ErrCancelled is returned when the context associated with the task is cancelled or expires.
	ErrCancelled = errors.New("task: context cancelled")

	// ErrDuplicate is returned when attempting to add a task ID that is already being tracked by the tracker.
	ErrDuplicate = errors.New("task: duplicate task ID")

	// ErrNotFound is returned when attempting to resolve, wait for, or remove a task ID that does not exist in the tracker's registry.
	ErrNotFound = errors.New("task: not found")

	// ErrWaitFor is returned when [Tracker.WaitFor] is invoked on a task that was not initialized with the [WithWait] option.
	ErrWaitFor = errors.New("task: was not created with WithWait option")
)

// Callback defines the function signature for handling completed jobs.
//
// The zero value is nil and not ready to use.
type Callback[T any] func(ctx context.Context, response T, err error)

// Option configures the operational parameters of a job upon registration.
//
// The zero value is nil and not ready to use.
type Option[T any] func(*config[T])

// CallbackStrategy defines how job callbacks are executed.
//
// The zero value is nil and not ready to use.
type CallbackStrategy func(fn func())

var (
	// AsyncStrategy executes the callback asynchronously in a new background goroutine (default).
	AsyncStrategy CallbackStrategy = func(fn func()) { go fn() }

	// SyncStrategy executes the callback synchronously in the current goroutine context.
	SyncStrategy CallbackStrategy = func(fn func()) { fn() }
)

// Entry represents the internal state, callbacks, and cleanup handlers of a tracked job.
//
// The zero value is empty and not ready to use.
// Read-only operations are safe for concurrent use; mutations require external synchronization.
type Entry[T any] struct {
	callback  Callback[T]
	waitCh    chan Result[T]
	keepAlive bool
	strategy  CallbackStrategy
	ctx       context.Context

	timerStop func() bool
	ctxStop   func() bool
}

// Result holds the final outcome of an executed, timed out, or cancelled job.
//
// The zero value is empty and ready to use.
// Read-only operations are safe for concurrent use; mutations require external synchronization.
type Result[T any] struct {
	val T
	err error
}

// Store defines the pluggable storage interface for managing pending jobs.
type Store[K comparable, T any] interface {
	// Add registers a new job entry.
	Add(ctx context.Context, id K, e *Entry[T]) error

	// Get retrieves an active job entry by ID.
	Get(ctx context.Context, id K) (*Entry[T], bool, error)

	// Delete removes an active job entry by ID.
	Delete(ctx context.Context, id K) (bool, error)

	// Len returns the total number of currently active jobs.
	Len(ctx context.Context) (int, error)

	// All retrieves a shallow copy of all currently active job entries.
	All(ctx context.Context) (map[K]*Entry[T], error)
}

// Tracker coordinates the lifecycle, asynchronous resolution, and synchronous wait states of multiple correlation-tracked jobs.
//
// The zero value is empty and not ready to use; use [NewTracker] or [NewTrackerWithStore] to initialize.
// All exported methods manage their own internal synchronization and are safe for concurrent use.
type Tracker[K comparable, T any] struct {
	mu       sync.Mutex
	store    Store[K, T]
	counter  atomic.Uint64
	closed   bool
	capacity int
	strategy CallbackStrategy

	entryPool sync.Pool
}

// NewTracker creates a new [Tracker] instance using the default in-memory storage.
func NewTracker[K comparable, T any](capacity int) *Tracker[K, T] {
	m := &Tracker[K, T]{
		capacity: capacity,
		store:    newMemoryStore[K, T](),
	}
	m.entryPool.New = func() any {
		return new(Entry[T])
	}

	return m
}

// NewTrackerWithStore creates a new [Tracker] instance using a custom [Store] backend.
func NewTrackerWithStore[K comparable, T any](capacity int, store Store[K, T]) *Tracker[K, T] {
	m := &Tracker[K, T]{
		capacity: capacity,
		store:    store,
	}
	m.entryPool.New = func() any {
		return new(Entry[T])
	}

	return m
}

// SetStore updates the backend storage used by the tracker.
func (m *Tracker[K, T]) SetStore(store Store[K, T]) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if store != nil {
		m.store = store
	}
}

// SetCallbackStrategy updates the default callback execution strategy.
func (m *Tracker[K, T]) SetCallbackStrategy(strategy CallbackStrategy) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if strategy != nil {
		m.strategy = strategy
	}
}

// WithTimeout sets a maximum duration the job is allowed to remain pending.
func WithTimeout[T any](timeout time.Duration) Option[T] {
	return func(c *config[T]) {
		c.timeout = timeout
	}
}

// WithContext associates a [context.Context] with the job.
func WithContext[T any](ctx context.Context) Option[T] {
	return func(c *config[T]) {
		c.ctx = ctx
	}
}

// WithKeepAlive indicates that the job should persist in the storage after its first resolution.
func WithKeepAlive[T any](keepAlive bool) Option[T] {
	return func(c *config[T]) {
		c.keepAlive = keepAlive
	}
}

// WithWait enables synchronous waiting for this job using the [Tracker.WaitFor] method.
func WithWait[T any]() Option[T] {
	return func(c *config[T]) {
		c.wait = true
	}
}

// WithCallbackStrategy sets a specific callback execution strategy for this job.
func WithCallbackStrategy[T any](strategy CallbackStrategy) Option[T] {
	return func(c *config[T]) {
		c.strategy = strategy
	}
}

// NextID generates a unique, monotonically increasing ID for a new job.
func (m *Tracker[K, T]) NextID() K {
	val := m.counter.Add(1)

	var zero K
	switch any(zero).(type) {
	case uint64:
		return any(val).(K)
	case int64:
		return any(int64(val)).(K) //nolint:gosec
	case uint32:
		return any(uint32(val)).(K) //nolint:gosec
	case int32:
		return any(int32(val)).(K) //nolint:gosec
	case uint:
		return any(uint(val)).(K)
	case int:
		return any(int(val)).(K) //nolint:gosec
	case string:
		return any(strconv.FormatUint(val, 10)).(K)
	default:
		rv := reflect.ValueOf(&zero).Elem()
		switch rv.Kind() {
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			rv.SetUint(val)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			rv.SetInt(int64(val)) //nolint:gosec
		case reflect.String:
			rv.SetString(strconv.FormatUint(val, 10))
		}
		return zero
	}
}

// Add registers a new job for tracking.
func (m *Tracker[K, T]) Add(id K, cb Callback[T], opts ...Option[T]) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrClosed
	}

	cfg := defaultConfig[T]()
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	storeLen, err := m.store.Len(cfg.ctx)
	if err != nil {
		return fmt.Errorf("store len: %w", err)
	}

	if m.capacity > 0 && storeLen >= m.capacity {
		return fmt.Errorf("task tracker capacity reached (%d)", m.capacity)
	}

	_, exists, err := m.store.Get(cfg.ctx, id)
	if err != nil {
		return fmt.Errorf("store get: %w", err)
	}

	if exists {
		return ErrDuplicate
	}

	e := m.entryPool.Get().(*Entry[T])
	e.callback = cb
	e.keepAlive = cfg.keepAlive
	e.waitCh = nil
	e.timerStop = nil
	e.ctxStop = nil
	e.strategy = cfg.strategy
	e.ctx = cfg.ctx

	if cfg.wait {
		e.waitCh = make(chan Result[T], 1)
	}

	if cfg.timeout > 0 {
		timer := time.AfterFunc(cfg.timeout, func() {
			m.Resolve(context.Background(), id, *new(T), ErrTimeout)
		})
		e.timerStop = timer.Stop
	}

	if cfg.ctx != nil && cfg.ctx != context.Background() {
		stop := context.AfterFunc(cfg.ctx, func() {
			m.Resolve(cfg.ctx, id, *new(T), ErrCancelled)
		})
		e.ctxStop = stop
	}

	if err := m.store.Add(cfg.ctx, id, e); err != nil {
		if e.timerStop != nil {
			e.timerStop()
		}

		if e.ctxStop != nil {
			e.ctxStop()
		}

		m.entryPool.Put(e)

		return fmt.Errorf("store add: %w", err)
	}

	return nil
}

// Resolve marks a job as complete by providing a response or an error using the given context.
func (m *Tracker[K, T]) Resolve(ctx context.Context, id K, response T, err error) bool {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return false
	}

	e, ok, getErr := m.store.Get(ctx, id)
	if getErr != nil || !ok {
		m.mu.Unlock()
		return false
	}

	if !e.keepAlive {
		_, _ = m.store.Delete(ctx, id)
	}

	wCh := e.waitCh
	e.waitCh = nil
	cb := e.callback
	eCtx := e.ctx
	eStrat := e.strategy
	mStrat := m.strategy

	m.mu.Unlock()

	if e.timerStop != nil {
		e.timerStop()
	}

	if e.ctxStop != nil {
		e.ctxStop()
	}

	if wCh != nil {
		wCh <- Result[T]{val: response, err: err}

		close(wCh)
	}

	if cb != nil {
		cbCtx := ctx
		if cbCtx == nil {
			cbCtx = eCtx
		}
		if cbCtx == nil {
			cbCtx = context.Background()
		}

		strategy := eStrat
		if strategy == nil {
			strategy = mStrat
		}
		if strategy == nil {
			strategy = AsyncStrategy
		}

		strategy(func() {
			defer func() { _ = recover() }()

			cb(cbCtx, response, err)
		})
	}

	if !e.keepAlive {
		e.callback = nil
		e.timerStop = nil
		e.ctxStop = nil
		e.ctx = nil
		m.entryPool.Put(e)
	}

	return true
}

// Remove removes the specific job without resolving it or invoking its callback using the given context.
func (m *Tracker[K, T]) Remove(ctx context.Context, id K) bool {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return false
	}

	e, ok, getErr := m.store.Get(ctx, id)
	if getErr != nil || !ok {
		m.mu.Unlock()
		return false
	}

	_, _ = m.store.Delete(ctx, id)
	m.mu.Unlock()

	if e.timerStop != nil {
		e.timerStop()
	}

	if e.ctxStop != nil {
		e.ctxStop()
	}

	if e.waitCh != nil {
		close(e.waitCh)
	}

	e.callback = nil
	e.waitCh = nil
	e.timerStop = nil
	e.ctxStop = nil
	e.ctx = nil
	m.entryPool.Put(e)

	return true
}

// WaitFor blocks the current goroutine until the specific job is resolved, the provided ctx is canceled, or the tracker is closed.
func (m *Tracker[K, T]) WaitFor(ctx context.Context, id K) (T, error) {
	m.mu.Lock()

	e, ok, err := m.store.Get(ctx, id)
	if err != nil {
		m.mu.Unlock()
		return *new(T), err
	}

	var wCh chan Result[T]
	if ok {
		wCh = e.waitCh
	}

	m.mu.Unlock()

	if !ok {
		return *new(T), ErrNotFound
	}

	if wCh == nil {
		return *new(T), ErrWaitFor
	}

	select {
	case res, ok := <-wCh:
		if !ok {
			return *new(T), ErrClosed
		}

		return res.val, res.err

	case <-ctx.Done():
		select {
		case res, ok := <-wCh:
			if ok {
				return res.val, res.err
			}
		default:
		}
		return *new(T), ctx.Err()
	}
}

// CancelAll cancels all the currently pending jobs using the given context.
func (m *Tracker[K, T]) CancelAll(ctx context.Context, err error) {
	m.mu.Lock()

	pending, getErr := m.store.All(ctx)
	if getErr == nil {
		for id := range pending {
			_, _ = m.store.Delete(ctx, id)
		}
	}
	mStrat := m.strategy

	m.mu.Unlock()

	if getErr == nil {
		m.closePending(pending, err, mStrat)
	}
}

// Close shuts down the tracker, marks it as closed, and cancels all currently pending jobs with [ErrClosed].
func (m *Tracker[K, T]) Close() error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil
	}

	m.closed = true

	pending, getErr := m.store.All(context.Background())
	if getErr == nil {
		for id := range pending {
			_, _ = m.store.Delete(context.Background(), id)
		}
	}
	mStrat := m.strategy

	m.mu.Unlock()

	if getErr == nil {
		m.closePending(pending, ErrClosed, mStrat)
	}

	return nil
}

// Count returns the number of currently active jobs being tracked.
func (m *Tracker[K, T]) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	c, _ := m.store.Len(context.Background())

	return c
}

func (m *Tracker[K, T]) closePending(pending map[K]*Entry[T], err error, mStrat CallbackStrategy) {
	for _, e := range pending {
		if e.timerStop != nil {
			e.timerStop()
		}

		if e.ctxStop != nil {
			e.ctxStop()
		}

		if e.waitCh != nil {
			close(e.waitCh)
		}

		cb := e.callback
		if cb != nil {
			cbCtx := e.ctx
			if cbCtx == nil {
				cbCtx = context.Background()
			}
			strategy := e.strategy
			if strategy == nil {
				strategy = mStrat
			}
			if strategy == nil {
				strategy = AsyncStrategy
			}

			strategy(func() {
				defer func() { _ = recover() }()

				cb(cbCtx, *new(T), err)
			})
		}

		e.callback = nil
		e.waitCh = nil
		e.timerStop = nil
		e.ctxStop = nil
		e.ctx = nil
		m.entryPool.Put(e)
	}
}

type config[T any] struct {
	timeout   time.Duration
	ctx       context.Context
	keepAlive bool
	wait      bool
	strategy  CallbackStrategy
}

func defaultConfig[T any]() config[T] {
	return config[T]{
		timeout: 30 * time.Second,
		ctx:     context.Background(),
	}
}

// memoryStore is the default in-memory implementation of [Store].
// Since all store access within Tracker is fully guarded by the tracker's
// internal lock, memoryStore does not implement internal locking.
type memoryStore[K comparable, T any] struct {
	jobs map[K]*Entry[T]
}

func newMemoryStore[K comparable, T any]() *memoryStore[K, T] {
	return &memoryStore[K, T]{
		jobs: make(map[K]*Entry[T]),
	}
}

func (s *memoryStore[K, T]) Add(ctx context.Context, id K, e *Entry[T]) error {
	s.jobs[id] = e
	return nil
}

func (s *memoryStore[K, T]) Get(ctx context.Context, id K) (*Entry[T], bool, error) {
	e, ok := s.jobs[id]
	return e, ok, nil
}

func (s *memoryStore[K, T]) Delete(ctx context.Context, id K) (bool, error) {
	_, ok := s.jobs[id]
	if ok {
		delete(s.jobs, id)
	}

	return ok, nil
}

func (s *memoryStore[K, T]) Len(ctx context.Context) (int, error) {
	return len(s.jobs), nil
}

func (s *memoryStore[K, T]) All(ctx context.Context) (map[K]*Entry[T], error) {
	res := make(map[K]*Entry[T], len(s.jobs))
	maps.Copy(res, s.jobs)
	return res, nil
}
