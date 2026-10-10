// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event

import (
	"context"
	"iter"
	"reflect"
	"slices"
	"sync"
	"sync/atomic"
)

// Event is the marker interface satisfied by all types that can be dispatched through a [Bus].
//
// Custom event structures must embed [Envelope] to satisfy this interface.
type Event interface {
	isEvent()
}

// Envelope provides a reusable implementation of the [Event] interface.
//
// Embedding it allows custom event structs to satisfy the [Event] interface
// and enables request-scoped context propagation.
// The zero value is valid and ready for use.
type Envelope struct {
	// Ctx represents the request context associated with the event execution.
	Ctx context.Context
}

func (Envelope) isEvent() {}

// Context returns the event's execution context, defaulting to [context.Background] if nil.
//
// Complexity: O(1) time; zero-allocation.
func (b *Envelope) Context() context.Context {
	if b.Ctx == nil {
		b.Ctx = context.Background()
	}

	return b.Ctx
}

// SetContext binds the provided request-scoped context to the event.
//
// Complexity: O(1) time; zero-allocation.
func (b *Envelope) SetContext(ctx context.Context) {
	b.Ctx = ctx
}

// Subscription represents an active event listener registered on a [Bus].
//
// It manages the lifecycle of an internal buffered event channel. It
// is created via [Bus.Subscribe] or [Bus.SubscribeAll] and must be closed
// using [Subscription.Unsubscribe] to prevent memory leaks.
// The zero value is not ready for use.
// All exported methods manage their own internal synchronization and are safe for concurrent use.
type Subscription struct {
	id     uint64
	types  []reflect.Type
	ch     chan Event
	closed atomic.Bool
	bus    *Bus
}

// C returns the read-only channel for receiving subscribed events.
//
// The channel is closed and drained when [Subscription.Unsubscribe] is called
// or when the associated [Bus] is closed.
// Complexity: O(1) time; zero-allocation.
func (s *Subscription) C() <-chan Event { return s.ch }

// Unsubscribe deregisters the subscription from its associated [Bus] and closes its channel.
//
// Complexity: amortized O(1) time; zero-allocation.
func (s *Subscription) Unsubscribe() {
	if s.closed.CompareAndSwap(false, true) {
		s.bus.unsubscribe(s)
	}
}

// EventsSeq returns an iterator yielding events received on this subscription until ctx is canceled or the subscription is closed.
//
// Iteration ceases immediately if yield returns false.
// Complexity: O(1) time per yielded element; zero-allocation.
func (s *Subscription) EventsSeq(ctx context.Context) iter.Seq[Event] {
	return func(yield func(Event) bool) {
		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-s.ch:
				if !ok {
					return
				}
				if !yield(ev) {
					return
				}
			}
		}
	}
}

// TypedSubscription provides a type-safe subscription wrapper for event type E.
//
// The zero value is not ready for use.
// All exported methods manage their own internal synchronization and are safe for concurrent use.
type TypedSubscription[E any] struct {
	sub    *Subscription
	ch     chan E
	closed atomic.Bool
}

// C returns the read-only channel for typed events.
//
// Complexity: O(1) time; zero-allocation.
func (ts *TypedSubscription[E]) C() <-chan E {
	return ts.ch
}

// Unsubscribe deregisters the subscription from its bus and closes its channel.
//
// Complexity: amortized O(1) time; zero-allocation.
func (ts *TypedSubscription[E]) Unsubscribe() {
	if ts.closed.CompareAndSwap(false, true) {
		ts.sub.Unsubscribe()
	}
}

// EventsSeq returns an iterator yielding typed events received on this subscription.
//
// Complexity: O(1) time per yielded element; zero-allocation.
func (ts *TypedSubscription[E]) EventsSeq(ctx context.Context) iter.Seq[E] {
	return func(yield func(E) bool) {
		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-ts.ch:
				if !ok {
					return
				}
				if !yield(ev) {
					return
				}
			}
		}
	}
}

// Bus implements a thread-safe, non-blocking, type-based event dispatcher.
//
// It routes published events to subscriptions based on their [reflect.Type].
// The zero value is not ready for use.
// All exported methods manage their own internal synchronization and are safe for concurrent use.
type Bus struct {
	mu        sync.RWMutex
	subs      map[reflect.Type]map[uint64]*Subscription
	all       map[uint64]*Subscription
	active    map[uint64]*Subscription
	nextID    atomic.Uint64
	closed    bool
	onDropped func(event Event, subID uint64)
}

// New returns a new, initialized [Bus] instance ready to route events.
//
// Complexity: O(1) time; allocates memory for internal structures.
func New() *Bus {
	return &Bus{
		subs:   make(map[reflect.Type]map[uint64]*Subscription),
		all:    make(map[uint64]*Subscription),
		active: make(map[uint64]*Subscription),
	}
}

// SetOnDropped sets a callback function to be called when events are dropped due to full subscription channels.
//
// Complexity: O(1) time; zero-allocation.
func (b *Bus) SetOnDropped(fn func(event Event, subID uint64)) {
	b.mu.Lock()
	b.onDropped = fn
	b.mu.Unlock()
}

// Subscribe registers a subscription to receive events of the specified types with a default buffer size of 128.
//
// If no event types are provided or if an event is nil, those entries are ignored.
// Duplicate event types within a single call are automatically deduplicated.
// If the [Bus] is closed, it returns a subscription with an already closed channel.
// Complexity: O(N) time where N is the number of event types; allocates memory for the new subscription.
func (b *Bus) Subscribe(evs ...Event) *Subscription {
	return b.SubscribeBuffer(128, evs...)
}

// SubscribeBuffer registers a subscription to receive events with a custom channel buffer capacity.
//
// Complexity: O(N) time where N is the number of event types; allocates memory for the new subscription.
func (b *Bus) SubscribeBuffer(bufferSize int, evs ...Event) *Subscription {
	b.mu.Lock()
	defer b.mu.Unlock()

	if bufferSize <= 0 {
		bufferSize = 128
	}

	id := b.nextID.Add(1)
	sub := &Subscription{
		id:  id,
		ch:  make(chan Event, bufferSize),
		bus: b,
	}

	if b.closed {
		sub.closed.Store(true)
		close(sub.ch)
		return sub
	}

	b.active[id] = sub

	for _, ev := range evs {
		if ev == nil {
			continue
		}

		t := resolveType(ev)

		if !slices.Contains(sub.types, t) {
			sub.types = append(sub.types, t)
			if b.subs[t] == nil {
				b.subs[t] = make(map[uint64]*Subscription)
			}

			b.subs[t][id] = sub
		}
	}

	return sub
}

// SubscribeTyped registers a type-safe subscription on the [Bus] for events of type E.
//
// E can be a value or pointer type that satisfies [Event].
// An optional custom bufferSize can be provided (default 128).
// Complexity: amortized O(1) time; allocates memory.
func (b *Bus) SubscribeTyped[E any](bufSize ...int) *TypedSubscription[E] {
	size := 128
	if len(bufSize) > 0 && bufSize[0] > 0 {
		size = bufSize[0]
	}

	t := reflect.TypeFor[E]()
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	b.mu.Lock()
	id := b.nextID.Add(1)
	sub := &Subscription{
		id:    id,
		types: []reflect.Type{t},
		ch:    make(chan Event, size),
		bus:   b,
	}

	if b.closed {
		sub.closed.Store(true)
		close(sub.ch)
	} else {
		b.active[id] = sub
		if b.subs[t] == nil {
			b.subs[t] = make(map[uint64]*Subscription)
		}
		b.subs[t][id] = sub
	}
	b.mu.Unlock()

	ts := &TypedSubscription[E]{
		sub: sub,
		ch:  make(chan E, size),
	}

	go func() {
		defer close(ts.ch)
		for raw := range sub.C() {
			var typed E
			var ok bool
			if v, isExact := raw.(E); isExact {
				typed = v
				ok = true
			} else {
				rv := reflect.ValueOf(raw)
				rt := reflect.TypeFor[E]()
				if rv.Kind() != reflect.Pointer && rt.Kind() == reflect.Pointer {
					if rv.Type() == rt.Elem() {
						ptr := reflect.New(rv.Type())
						ptr.Elem().Set(rv)
						typed = ptr.Interface().(E)
						ok = true
					}
				} else if rv.Kind() == reflect.Pointer && rt.Kind() != reflect.Pointer {
					val := rv.Elem()
					if val.Type() == rt {
						typed = val.Interface().(E)
						ok = true
					}
				}
			}

			if ok {
				select {
				case ts.ch <- typed:
				default:
				}
			} else if typedPtr, ok := any(raw).(*E); ok && typedPtr != nil {
				select {
				case ts.ch <- *typedPtr:
				default:
				}
			}
		}
	}()

	return ts
}

// SubscribeAll registers a subscription to receive every event published on the [Bus].
//
// If the [Bus] is closed, it returns a subscription with an already closed channel.
// Complexity: amortized O(1) time; allocates memory.
func (b *Bus) SubscribeAll() *Subscription {
	b.mu.Lock()
	defer b.mu.Unlock()

	id := b.nextID.Add(1)
	sub := &Subscription{
		id:  id,
		ch:  make(chan Event, 256),
		bus: b,
	}

	if b.closed {
		sub.closed.Store(true)
		close(sub.ch)
		return sub
	}

	b.active[id] = sub
	b.all[id] = sub

	return sub
}

// Publish broadcasts an event to all matched subscriptions.
//
// It is non-blocking. If a matched subscriber's channel buffer
// is full, the event is dropped for that subscriber (or routed to onDropped).
// If the event is nil, or the [Bus] is closed, it returns immediately.
// Complexity: O(N + M) time where N is the number of active subscriptions for the event type, and M is the number of catch-all subscriptions; zero-allocation.
func (b *Bus) Publish(event Event) {
	if event == nil {
		return
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return
	}

	if typeSubs, ok := b.subs[resolveType(event)]; ok {
		for _, sub := range typeSubs {
			b.directSend(sub, event)
		}
	}

	for _, sub := range b.all {
		b.directSend(sub, event)
	}
}

// Close shuts down the [Bus], marks it as closed, and closes all active subscription channels.
//
// Subsequent calls are safe, idempotent no-ops and return nil immediately.
// Complexity: O(S) time where S is the total number of subscriptions; allocates memory for unique channel deduplication.
func (b *Bus) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}

	b.closed = true

	for _, s := range b.active {
		s.closed.Store(true)
		close(s.ch)
	}

	b.subs = nil
	b.all = nil
	b.active = nil

	return nil
}

// resolveType extracts the underlying [reflect.Type] of an event, stripping
// any pointer wrappers to treat MyEvent{} and &MyEvent{} identically.
func resolveType(ev any) reflect.Type {
	t := reflect.TypeOf(ev)
	if t != nil && t.Kind() == reflect.Pointer {
		return t.Elem()
	}

	return t
}

// directSend attempts a non-blocking write to the subscription's channel.
//
// This is safe from "send on closed channel" panics because the caller
// must hold b.mu.RLock(), preventing concurrent channel closure.
func (b *Bus) directSend(sub *Subscription, ev Event) {
	if sub.closed.Load() {
		return
	}

	select {
	case sub.ch <- ev:
	default:
		if b.onDropped != nil {
			b.onDropped(ev, sub.id)
		}
	}
}

// unsubscribe removes a subscription from the bus registry and closes its channel.
func (b *Bus) unsubscribe(sub *Subscription) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Double-close prevention: if the bus has already been closed, Close()
	// has already handled closing all active subscription channels.
	if b.closed {
		return
	}

	for _, t := range sub.types {
		if typeSubs, ok := b.subs[t]; ok {
			delete(typeSubs, sub.id)

			if len(typeSubs) == 0 {
				delete(b.subs, t)
			}
		}
	}

	delete(b.all, sub.id)
	delete(b.active, sub.id)
	close(sub.ch)
}
