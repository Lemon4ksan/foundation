// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event

import (
	"context"
	"maps"
	"reflect"
	"slices"
	"sync"
	"sync/atomic"
)

// Event is the marker interface satisfied by all types that can be dispatched through a [Bus].
//
// Custom event structures must embed [BaseEvent] to satisfy this interface.
type Event interface {
	isEvent()
}

// BaseEvent provides a default, reusable implementation of the [Event] interface.
//
// By embedding BaseEvent, custom event structs automatically satisfy the [Event]
// interface and gain request-scoped context propagation capabilities.
type BaseEvent struct {
	// Ctx represents the request context associated with the event execution.
	Ctx context.Context
}

func (BaseEvent) isEvent() {}

// Context returns the event's execution context, defaulting to [context.Background] if nil.
func (b *BaseEvent) Context() context.Context {
	if b.Ctx == nil {
		b.Ctx = context.Background()
	}

	return b.Ctx
}

// SetContext binds the provided request-scoped context to the event.
func (b *BaseEvent) SetContext(ctx context.Context) {
	b.Ctx = ctx
}

// Subscription represents an active event listener registered on a [Bus].
//
// It manages the lifecycle of an internal buffered event channel. Subscriptions
// are created via [Bus.Subscribe] or [Bus.SubscribeAll] and must be cleaned up
// using [Subscription.Unsubscribe] to prevent resource and memory leaks.
//
// All methods on Subscription are safe for concurrent use by multiple goroutines.
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
func (s *Subscription) C() <-chan Event { return s.ch }

// Unsubscribe deregisters the subscription from its associated [Bus] and closes its channel.
//
// It is safe to call Unsubscribe concurrently from multiple goroutines, and
// subsequent invocations are safe no-ops.
func (s *Subscription) Unsubscribe() {
	if s.closed.CompareAndSwap(false, true) {
		s.bus.unsubscribe(s)
	}
}

// Bus implements a thread-safe, non-blocking, type-based event dispatcher.
//
// It routes published events to subscriptions based on their [reflect.Type].
// Thread safety is internally managed using a [sync.RWMutex] protecting maps
// of active subscriptions.
//
// New instances of Bus must be created using the [New] constructor function.
type Bus struct {
	mu        sync.RWMutex
	subs      map[reflect.Type]map[uint64]*Subscription
	all       map[uint64]*Subscription
	nextID    atomic.Uint64
	closed    bool
	onDropped func(event Event, subID uint64)
}

// New returns a new, initialized [Bus] instance ready to route events.
func New() *Bus {
	return &Bus{
		subs: make(map[reflect.Type]map[uint64]*Subscription),
		all:  make(map[uint64]*Subscription),
	}
}

// SetOnDropped sets a callback function to be called when packets are dropped.
func (b *Bus) SetOnDropped(fn func(event Event, subID uint64)) {
	b.mu.Lock()
	b.onDropped = fn
	b.mu.Unlock()
}

// Subscribe registers a subscription to receive events of the specified types.
//
// If no event types are provided or if an event is nil, those entries are ignored.
// Duplicate event types within a single Subscribe call are automatically deduplicated.
// If the Bus is closed, Subscribe returns a subscription with an already closed channel.
func (b *Bus) Subscribe(evs ...Event) *Subscription {
	b.mu.Lock()
	defer b.mu.Unlock()

	id := b.nextID.Add(1)
	sub := &Subscription{
		id:  id,
		ch:  make(chan Event, 128),
		bus: b,
	}

	if b.closed {
		sub.closed.Store(true)
		close(sub.ch)
		return sub
	}

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

// SubscribeAll registers a subscription to receive every event published on the [Bus].
//
// If the Bus is closed, SubscribeAll returns a subscription with an already closed channel.
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

	b.all[id] = sub

	return sub
}

// Publish broadcasts a single event to all active matched subscriptions.
//
// Publish is strictly non-blocking. If a matched subscriber's channel buffer
// is full, the event is silently dropped for that subscriber.
// If the event is nil, or the Bus is closed, Publish returns immediately.
//
// # Complexity
//
// Time Complexity: O(N + M), where N is the number of active subscriptions registered
// for the given event type plus the M number of "subscribe-all" subscriptions.
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
// It returns nil to satisfy standard Closer interfaces. Subsequent calls to Close
// are safe, idempotent no-ops and return nil immediately.
func (b *Bus) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}

	b.closed = true

	unique := make(map[uint64]*Subscription)
	for _, m := range b.subs {
		maps.Copy(unique, m)
	}

	maps.Copy(unique, b.all)

	for _, s := range unique {
		s.closed.Store(true)
		close(s.ch)
	}

	b.subs = nil
	b.all = nil

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
	close(sub.ch)
}
