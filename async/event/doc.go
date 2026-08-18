// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// package event implements a thread-safe, non-blocking, type-based event bus
// for asynchronous in-process communication.
//
// It provides a decoupled architecture where independent components can publish
// and subscribe to typed events without direct compile-time dependencies.
// Events are routed dynamically based on their [reflect.Type]. To ensure
// consistent routing, both pointer and value representations of the same struct
// (e.g., MyEvent{} and &MyEvent{}) resolve to the same underlying type.
//
// # Non-Blocking Delivery & Buffering
//
// The bus prioritizes system availability and low latency. Event delivery is
// strictly non-blocking: if a subscriber's channel buffer becomes full,
// incoming events are silently dropped for that subscriber to prevent slow
// consumers from creating backpressure on publishers.
//
// Type-specific subscriptions created via [Bus.Subscribe] are allocated with
// an internal channel buffer capacity of 128 events. Global subscriptions
// created via [Bus.SubscribeAll] are allocated with a capacity of 256 events.
//
// # Concurrency Safety Invariants
//
// To prevent common concurrency pitfalls, the bus enforces two strict invariants:
//
//  1. Safe Publishing: The [Bus.Publish] method acquires a shared read lock ([sync.RWMutex.RLock]).
//     This guarantees that no active subscription channel can be closed concurrently by
//     [Subscription.Unsubscribe] or [Bus.Close] (both of which require an exclusive write lock)
//     while a send operation is in progress. This entirely prevents "send on closed channel" panics.
//
//  2. Safe Unsubscription: A double-close prevention check in [Bus.unsubscribe] ensures
//     that if [Bus.Close] and [Subscription.Unsubscribe] are invoked concurrently,
//     the underlying channel is closed exactly once, preventing "close of closed channel" panics.
//
// # Example
//
//	package main
//
//	import (
//		"fmt"
//		"sync"
//
//		"github.com/lemon4ksan/foundation/async/event"
//	)
//
//	type MyEvent struct {
//		bus.BaseEvent
//		Message string
//	}
//
//	func main() {
//		b := bus.New()
//		sub := b.Subscribe(MyEvent{})
//		defer sub.Unsubscribe()
//
//		var wg sync.WaitGroup
//		wg.Add(1)
//
//		go func() {
//			defer wg.Done()
//			for ev := range sub.C() {
//				msg := ev.(MyEvent).Message
//				fmt.Println("Received:", msg)
//			}
//		}()
//
//		b.Publish(MyEvent{Message: "Hello, World!"})
//		b.Close()
//		wg.Wait()
//	}
package event
