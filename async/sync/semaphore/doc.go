// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package semaphore provides a dynamically resizable counting semaphore.
//
// Unlike a fixed-size buffered channel, the limit can be changed at runtime via
// [Semaphore.Resize] without restarting or rebuilding the coordinator. Context-aware
// acquisition ensures that waiting goroutines unblock immediately when their
// context is cancelled or expires, preventing resource leaks.
//
// # Architecture
//
// The [Semaphore] tracks active slot count and a queue of waiter channels.
// [Acquire] either increments the counter immediately or blocks on a channel.
// [Release] decrements the counter and wakes the next waiter. [Resize] updates
// the limit and wakes as many waiters as the new capacity allows.
//
// # Concurrency & Safety Invariants
//
//  1. Slot Leak Prevention: To prevent concurrency slot leaks during race conditions,
//     [Semaphore.Acquire]'s cancellation block checks if the waiter channel was
//     already popped and closed by [Semaphore.notifyWaiters]. If it was, the method
//     discards the context cancellation and returns success (nil) to ensure the acquired
//     slot is eventually released.
//  2. Safe Boundary Protection: The initial limit and any runtime resizing are guarded
//     to prevent negative capacities. Any negative input is automatically coerced to 0,
//     ensuring stable mathematical invariants for the active slot counters.
//
// # Error Handling
//
// [Acquire] returns [context.Err] immediately if the context is already
// cancelled. If the context is cancelled while waiting, the waiter is removed
// from the queue and the context error is returned - no zombie goroutines.
//
// # Example
//
//	package main
//
//	import (
//	    "context"
//	    "fmt"
//
//	    "github.com/lemon4ksan/foundation/async/sync/semaphore"
//	)
//
//	func main() {
//	    sem := semaphore.New(3)
//	    ctx := context.Background()
//
//	    if err := sem.Acquire(ctx); err != nil {
//	        panic(err)
//	    }
//	    fmt.Println("acquired slot")
//
//	    sem.Release()
//	    sem.Resize(10)
//	    fmt.Println("resized to 10")
//	}
package semaphore
