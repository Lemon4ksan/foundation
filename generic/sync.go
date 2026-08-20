// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generic

import (
	"sync"
	"sync/atomic"
)

// Safe encapsulates a value of type T and protects concurrent read and write access via [sync.RWMutex].
type Safe[T any] struct {
	mu sync.RWMutex
	v  T
}

// NewSafe constructs a new thread-safe [Safe] container initialized with initial.
func NewSafe[T any](initial T) *Safe[T] {
	return &Safe[T]{v: initial}
}

// Get returns a copy of the current protected value under a shared read lock.
func (s *Safe[T]) Get() T {
	s.mu.RLock()
	val := s.v
	s.mu.RUnlock()

	return val
}

// Set replaces the protected value under an exclusive write lock.
func (s *Safe[T]) Set(v T) {
	s.mu.Lock()
	s.v = v
	s.mu.Unlock()
}

// Swap atomically replaces the protected value and returns the previous value.
func (s *Safe[T]) Swap(v T) T {
	s.mu.Lock()
	old := s.v
	s.v = v
	s.mu.Unlock()

	return old
}

// Update atomically mutates the value via fn under an exclusive lock, resolving Read-Modify-Write races.
func (s *Safe[T]) Update(fn func(current T) T) T {
	s.mu.Lock()
	s.v = fn(s.v)
	res := s.v
	s.mu.Unlock()

	return res
}

// Mutate provides direct mutable pointer access to T inside an exclusive lock (ideal for maps, slices, or complex structs).
func (s *Safe[T]) Mutate(fn func(val *T)) {
	s.mu.Lock()
	fn(&s.v)
	s.mu.Unlock()
}

// Read provides scoped read-only access to T inside a shared lock without escaping value references.
func (s *Safe[T]) Read(fn func(val T)) {
	s.mu.RLock()
	fn(s.v)
	s.mu.RUnlock()
}

// Atomic is a generic, lock-free atomic container built on [atomic.Pointer].
type Atomic[T any] struct {
	ptr atomic.Pointer[T]
}

// NewAtomic constructs a new [Atomic] container initialized with initial.
func NewAtomic[T any](initial T) *Atomic[T] {
	a := &Atomic[T]{}
	a.Store(initial)

	return a
}

// NewAtomicPtr constructs an [Atomic] container storing an initial pointer.
func NewAtomicPtr[T any](initial *T) *Atomic[T] {
	a := &Atomic[T]{}
	if initial != nil {
		a.ptr.Store(initial)
	}

	return a
}

// Load atomically returns a copy of the stored value, or zero T if nil.
func (a *Atomic[T]) Load() T {
	p := a.ptr.Load()
	if p == nil {
		var zero T
		return zero
	}

	return *p
}

// LoadPtr atomically returns the stored pointer directly.
func (a *Atomic[T]) LoadPtr() *T {
	return a.ptr.Load()
}

// Store atomically replaces the stored value.
func (a *Atomic[T]) Store(val T) {
	a.ptr.Store(&val)
}

// StorePtr atomically replaces the stored pointer directly.
func (a *Atomic[T]) StorePtr(val *T) {
	a.ptr.Store(val)
}

// Swap atomically replaces the stored value and returns the previous value.
func (a *Atomic[T]) Swap(val T) T {
	old := a.ptr.Swap(&val)
	if old == nil {
		var zero T
		return zero
	}

	return *old
}

// SwapPtr atomically replaces the stored pointer and returns the previous pointer.
func (a *Atomic[T]) SwapPtr(val *T) *T {
	return a.ptr.Swap(val)
}

// Signal is a thread-safe, idempotent, one-shot broadcast coordination primitive.
type Signal struct {
	ch   chan struct{}
	once sync.Once
}

// NewSignal creates an initialized [Signal].
func NewSignal() *Signal {
	return &Signal{ch: make(chan struct{})}
}

// Emit broadcasts the signal to all listeners waiting on [Done]. Safe for multiple concurrent calls.
func (s *Signal) Emit() {
	s.once.Do(func() {
		close(s.ch)
	})
}

// Done returns the read-only channel closed when [Emit] is invoked.
func (s *Signal) Done() <-chan struct{} {
	return s.ch
}

// IsSignaled reports whether [Emit] has already been called.
func (s *Signal) IsSignaled() bool {
	select {
	case <-s.ch:
		return true
	default:
		return false
	}
}

// Pool is a generic, type-safe wrapper over [sync.Pool] that eliminates boxing and interface assertions.
type Pool[T any] struct {
	pool sync.Pool
}

// NewPool constructs a new type-safe [Pool] with a factory producing fresh *T instances.
func NewPool[T any](factory func() *T) *Pool[T] {
	return &Pool[T]{
		pool: sync.Pool{
			New: func() any {
				return factory()
			},
		},
	}
}

// Get selects an arbitrary item from the Pool, removes it from the Pool, and returns it to the caller.
func (p *Pool[T]) Get() *T {
	val, ok := p.pool.Get().(*T)
	if !ok {
		return nil
	}

	return val
}

// Put adds v to the pool. Nil values are safely ignored.
func (p *Pool[T]) Put(v *T) {
	if v != nil {
		p.pool.Put(v)
	}
}

// ConcurrentMap is a type-safe, concurrent hash map wrapping [sync.Map] with zero runtime type assertion overhead.
type ConcurrentMap[K comparable, V any] struct {
	m sync.Map
}

// Load returns the value stored in the map for a key, or zero V if no value is present.
func (m *ConcurrentMap[K, V]) Load(key K) (V, bool) {
	val, ok := m.m.Load(key)
	if !ok {
		var zero V
		return zero, false
	}

	typed, ok := val.(V)
	if !ok {
		var zero V
		return zero, false
	}

	return typed, true
}

// Store sets the value for a key.
func (m *ConcurrentMap[K, V]) Store(key K, val V) {
	m.m.Store(key, val)
}

// LoadOrStore returns the existing value for the key if present.
// Otherwise, it stores and returns the given value.
// The loaded result is true if the value was loaded, false if stored.
func (m *ConcurrentMap[K, V]) LoadOrStore(key K, val V) (V, bool) {
	actual, loaded := m.m.LoadOrStore(key, val)
	typed, ok := actual.(V)
	if !ok {
		var zero V
		return zero, false
	}

	return typed, loaded
}

// LoadAndDelete deletes the value for a key, returning the previous value if any.
func (m *ConcurrentMap[K, V]) LoadAndDelete(key K) (V, bool) {
	val, loaded := m.m.LoadAndDelete(key)
	if !loaded {
		var zero V
		return zero, false
	}

	typed, ok := val.(V)
	if !ok {
		var zero V
		return zero, false
	}

	return typed, true
}

// Delete deletes the value for a key.
func (m *ConcurrentMap[K, V]) Delete(key K) {
	m.m.Delete(key)
}

// Range calls f sequentially for each key and value present in the map.
// If f returns false, range stops the iteration.
func (m *ConcurrentMap[K, V]) Range(fn func(key K, val V) bool) {
	m.m.Range(func(k, v any) bool {
		typedK, okK := k.(K)
		typedV, okV := v.(V)
		if !okK || !okV {
			return true
		}

		return fn(typedK, typedV)
	})
}

// Clear removes all entries from the map.
func (m *ConcurrentMap[K, V]) Clear() {
	m.m.Clear()
}

type singleflightCall[V any] struct {
	val V
	err error
	sig Signal
}

// Singleflight provides duplicate function call suppression across concurrent callers.
type Singleflight[K comparable, V any] struct {
	mu sync.Mutex
	m  map[K]*singleflightCall[V]
}

// NewSingleflight creates a new generic [Singleflight] coalescer.
func NewSingleflight[K comparable, V any]() *Singleflight[K, V] {
	return &Singleflight[K, V]{
		m: make(map[K]*singleflightCall[V]),
	}
}

// Do executes and returns the results of the given function, making
// sure that only one execution is in-flight for a given key at a
// time.
func (g *Singleflight[K, V]) Do(key K, fn func() (V, error)) (V, error) {
	g.mu.Lock()
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		<-c.sig.Done()

		return c.val, c.err
	}

	c := &singleflightCall[V]{
		sig: *NewSignal(),
	}
	g.m[key] = c
	g.mu.Unlock()

	c.val, c.err = fn()
	c.sig.Emit()

	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	return c.val, c.err
}
