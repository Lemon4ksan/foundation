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
	// ErrJobTimeout is returned when a job exceeds its allowed execution time
	// defined by [WithTimeout].
	ErrJobTimeout = errors.New("job: request timed out")

	// ErrJobClosed is returned when the manager is shutting down and all
	// pending jobs are being cancelled.
	ErrJobClosed = errors.New("job: manager closed")

	// ErrJobCancelled is returned when the context associated with the job
	// is cancelled or expires.
	ErrJobCancelled = errors.New("job: context cancelled")

	// ErrJobDuplicate is returned when attempting to add a job ID that is
	// already being tracked by the manager.
	ErrJobDuplicate = errors.New("job: duplicate job ID")

	// ErrJobNotFound is returned when attempting to resolve, wait for, or remove
	// a job ID that does not exist in the manager's registry.
	ErrJobNotFound = errors.New("job: not found")

	// ErrWaitFor is returned when [Manager.WaitFor] is invoked on a job
	// that was not initialized with the [WithWait] option.
	ErrWaitFor = errors.New("job was not created with WithWait option")
)

// Callback defines the function signature for handling completed jobs.
//
// The ctx parameter propagates the context of the resolution stage.
// The response contains the computed result, while err contains any error
// that occurred during the job's lifecycle (e.g., timeout, cancellation, or shutdown).
type Callback[T any] func(ctx context.Context, response T, err error)

// Option configures the operational parameters of a job upon registration.
type Option[T any] func(*config[T])

// CallbackStrategy defines how job callbacks are executed.
type CallbackStrategy func(fn func())

var (
	// AsyncStrategy executes the callback asynchronously in a new background goroutine (default).
	AsyncStrategy CallbackStrategy = func(fn func()) { go fn() }

	// SyncStrategy executes the callback synchronously in the current goroutine context.
	SyncStrategy CallbackStrategy = func(fn func()) { fn() }
)

// Entry represents the internal state, callbacks, and cleanup handlers of a tracked job.
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
type Result[T any] struct {
	val T
	err error
}

// Store defines the pluggable storage interface for managing pending jobs.
//
// This allows users to implement custom backends (e.g., in-memory map, Redis,
// or persistent SQL databases). All write and read operations on the Store
// are fully protected by the [Manager]'s internal lock, meaning custom implementations
// do not need to be safe for concurrent use.
type Store[K comparable, T any] interface {
	// Add registers a new job entry.
	Add(ctx context.Context, id K, e *Entry[T]) error

	// Get retrieves an active job entry by ID.
	Get(ctx context.Context, id K) (*Entry[T], bool, error)

	// Delete removes an active job entry by ID.
	Delete(ctx context.Context, id K) (bool, error)

	// Len returns the total number of currently active jobs.
	Len(ctx context.Context) (int, error)

	// GetAll retrieves a shallow copy of all currently active job entries.
	GetAll(ctx context.Context) (map[K]*Entry[T], error)
}

// Manager coordinates the lifecycle, asynchronous resolution, and synchronous
// wait states of multiple correlation-tracked jobs.
//
// Entries are highly lightweight and recycled through an internal [sync.Pool]
// to eliminate memory allocation overhead on hot paths.
//
// A Manager is safe for concurrent use by multiple goroutines.
type Manager[K comparable, T any] struct {
	mu       sync.Mutex
	store    Store[K, T]
	counter  atomic.Uint64
	closed   bool
	capacity int
	strategy CallbackStrategy

	entryPool sync.Pool
}

// NewManager creates a new [Manager] instance using the default in-memory storage.
//
// The capacity parameter limits the maximum number of concurrent pending jobs
// to protect against memory exhaustion. Set capacity to 0 for unlimited jobs.
func NewManager[K comparable, T any](capacity int) *Manager[K, T] {
	m := &Manager[K, T]{
		capacity: capacity,
		store:    newMemoryStore[K, T](),
	}
	m.entryPool.New = func() any {
		return new(Entry[T])
	}

	return m
}

// NewManagerWithStore creates a new [Manager] instance using a custom [Store] backend.
//
// The capacity parameter limits the maximum number of concurrent pending jobs.
// Set capacity to 0 for unlimited jobs.
func NewManagerWithStore[K comparable, T any](capacity int, store Store[K, T]) *Manager[K, T] {
	m := &Manager[K, T]{
		capacity: capacity,
		store:    store,
	}
	m.entryPool.New = func() any {
		return new(Entry[T])
	}

	return m
}

// SetStore updates the backend storage used by the manager.
// It is safe to call concurrently while the manager is running.
func (m *Manager[K, T]) SetStore(store Store[K, T]) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if store != nil {
		m.store = store
	}
}

// SetCallbackStrategy updates the default callback execution strategy.
// It is safe to call concurrently while the manager is running.
func (m *Manager[K, T]) SetCallbackStrategy(strategy CallbackStrategy) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if strategy != nil {
		m.strategy = strategy
	}
}

// WithTimeout sets a maximum duration the job is allowed to remain pending.
// If the timeout is reached before resolution, the job is automatically resolved
// with [ErrJobTimeout].
func WithTimeout[T any](timeout time.Duration) Option[T] {
	return func(c *config[T]) {
		c.timeout = timeout
	}
}

// WithContext associates a [context.Context] with the job.
// If the context is cancelled or expires before resolution, the job is automatically
// resolved with [ErrJobCancelled].
func WithContext[T any](ctx context.Context) Option[T] {
	return func(c *config[T]) {
		c.ctx = ctx
	}
}

// WithKeepAlive indicates that the job should persist in the storage after its
// first resolution. This is useful for streaming, multiplexed, or multipart responses.
//
// If keepAlive is true, the job must be manually removed using [Manager.Remove]
// once the stream is completed to prevent memory leaks.
func WithKeepAlive[T any](keepAlive bool) Option[T] {
	return func(c *config[T]) {
		c.keepAlive = keepAlive
	}
}

// WithWait enables synchronous waiting for this job using the [Manager.WaitFor] method.
// Without this option, calling WaitFor on the job ID will return an [ErrWaitFor] error.
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
// This ID should be sent to the remote system to be returned in the response.
//
// If the key type K is not supported (not an integer or string type), it returns
// the zero value of K.
func (m *Manager[K, T]) NextID() K {
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
		return zero
	}
}

// Add registers a new job for tracking.
//
// The cb argument can be nil if you plan to wait for the job synchronously
// using the [Manager.WaitFor] method. If you pass nil, make sure to configure
// the job with the [WithWait] option.
//
// # Errors
//
// It returns [ErrJobClosed] if the manager is already closed, or [ErrJobDuplicate]
// if the provided id is already registered. If a capacity limit is configured
// and reached, it returns an error wrapping the capacity limit details.
//
// # Complexity
//
// Time Complexity: O(1) when using the default in-memory store.
func (m *Manager[K, T]) Add(id K, cb Callback[T], opts ...Option[T]) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrJobClosed
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
		return fmt.Errorf("job manager capacity reached (%d)", m.capacity)
	}

	_, exists, err := m.store.Get(cfg.ctx, id)
	if err != nil {
		return fmt.Errorf("store get: %w", err)
	}

	if exists {
		return ErrJobDuplicate
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
			m.Resolve(id, *new(T), ErrJobTimeout)
		})
		e.timerStop = timer.Stop
	}

	if cfg.ctx != nil && cfg.ctx != context.Background() {
		stop := context.AfterFunc(cfg.ctx, func() {
			m.Resolve(id, *new(T), ErrJobCancelled)
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

// Resolve marks a job as complete by providing a response or an error.
//
// The internal state is cleaned up immediately unless the job was registered
// with the [WithKeepAlive] option. Any goroutine currently blocked in the
// [Manager.WaitFor] call for this job ID will be unblocked and receive
// the response and error.
//
// If a callback was registered, it is executed according to the configured
// strategy (defaulting to asynchronously in a new goroutine) to prevent
// blocking the caller thread.
//
// It returns true if the job was found and successfully resolved, or false if
// the job did not exist (e.g., it had already timed out or been resolved).
//
// # Complexity
//
// Time Complexity: O(1) when using the default in-memory store.
func (m *Manager[K, T]) Resolve(id K, response T, err error) bool {
	return m.ResolveContext(context.Background(), id, response, err)
}

// ResolveContext marks a job as complete by providing a response or an error using a specific context.
func (m *Manager[K, T]) ResolveContext(ctx context.Context, id K, response T, err error) bool {
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
		cbCtx := coalesceNil(ctx, e.ctx, context.Background())
		strategy := coalesceNil(e.strategy, m.strategy, AsyncStrategy)

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

// Remove removes the specific job without resolving it or invoking its callback.
//
// This method can be used to manually clean up jobs that were registered
// with the [WithKeepAlive] option. Any blocked [Manager.WaitFor] calls for
// this job ID will be closed immediately without a result.
//
// It returns true if the job was found and removed, or false otherwise.
//
// # Complexity
//
// Time Complexity: O(1) when using the default in-memory store.
func (m *Manager[K, T]) Remove(id K) bool {
	return m.RemoveContext(context.Background(), id)
}

// RemoveContext removes the specific job without resolving it or invoking its callback using a specific context.
func (m *Manager[K, T]) RemoveContext(ctx context.Context, id K) bool {
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

// WaitFor blocks the current goroutine until the specific job is resolved,
// the provided ctx is canceled, or the manager is closed.
//
// It returns [ErrWaitFor] if the job was registered without the [WithWait] option,
// or [ErrJobNotFound] if the job does not exist. If the context is canceled
// before the job is resolved, it returns the context error. If the manager
// is closed while waiting, it returns [ErrJobClosed].
func (m *Manager[K, T]) WaitFor(ctx context.Context, id K) (T, error) {
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
		return *new(T), ErrJobNotFound
	}

	if wCh == nil {
		return *new(T), ErrWaitFor
	}

	select {
	case res, ok := <-wCh:
		if !ok {
			return *new(T), ErrJobClosed
		}

		return res.val, res.err

	case <-ctx.Done():
		return *new(T), ctx.Err()
	}
}

// CancelAll cancels all the currently pending jobs.
//
// All active job callbacks are executed, and any blocked [Manager.WaitFor]
// calls are unblocked, receiving the provided err value.
func (m *Manager[K, T]) CancelAll(err error) {
	m.CancelAllContext(context.Background(), err)
}

// CancelAllContext cancels all the currently pending jobs using a specific context.
func (m *Manager[K, T]) CancelAllContext(ctx context.Context, err error) {
	m.mu.Lock()

	pending, getErr := m.store.GetAll(ctx)
	if getErr == nil {
		for id := range pending {
			_, _ = m.store.Delete(ctx, id)
		}
	}

	m.mu.Unlock()

	if getErr == nil {
		m.closePending(pending, err)
	}
}

// Close shuts down the manager, marks it as closed, and cancels all currently
// pending jobs with [ErrJobClosed].
//
// Once the manager is closed, no new jobs can be added via the [Manager.Add] method.
// Close is safe for concurrent use and idempotent; subsequent calls return nil.
func (m *Manager[K, T]) Close() error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil
	}

	m.closed = true

	pending, getErr := m.store.GetAll(context.Background())
	if getErr == nil {
		for id := range pending {
			_, _ = m.store.Delete(context.Background(), id)
		}
	}

	m.mu.Unlock()

	if getErr == nil {
		m.closePending(pending, ErrJobClosed)
	}

	return nil
}

// Count returns the number of currently active jobs being tracked.
func (m *Manager[K, T]) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	c, _ := m.store.Len(context.Background())

	return c
}

func (m *Manager[K, T]) closePending(pending map[K]*Entry[T], err error) {
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
			cbCtx := coalesceNil(e.ctx, context.Background())
			strategy := coalesceNil(e.strategy, m.strategy, AsyncStrategy)

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
// Since all store access within Manager is fully guarded by the manager's
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

func (s *memoryStore[K, T]) GetAll(ctx context.Context) (map[K]*Entry[T], error) {
	res := make(map[K]*Entry[T], len(s.jobs))
	maps.Copy(res, s.jobs)
	return res, nil
}

// coalesceNil returns the first non-nil value from the provided list.
func coalesceNil[T any](vals ...T) T {
	for _, v := range vals {
		if !isNil(v) {
			return v
		}
	}

	var zero T

	return zero
}

// isNil safely determines if the provided interface value is nil.
func isNil(val any) bool {
	if val == nil {
		return true
	}

	v := reflect.ValueOf(val)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Pointer, reflect.UnsafePointer, reflect.Slice:
		return v.IsNil()
	}

	return false
}
