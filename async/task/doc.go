// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package task provides a concurrent-safe mechanism for tracking asynchronous
// request-response cycles by unique correlation IDs.
//
// It is primarily designed for high-performance network protocol implementations
// (such as TCP, UDP, WebSockets) where a client transmits a request with an ID
// and expects a matching response to arrive asynchronously at some later time.
// This package manages the entire job lifecycle, including configurable timeouts,
// request-scoped context cancellation, asynchronous callbacks, and synchronous
// waiting.
//
// # Architecture
//
// The central coordinator is the [Manager]. It maps unique correlation IDs
// of type K to job entries of type [Entry]. A job can be configured with an
// optional timeout, an associated [context.Context], and a persistence flag.
// When a response arrives, [Manager.Resolve] is called with the matching ID,
// which triggers the job's callback and unblocks any goroutine currently waiting
// on [Manager.WaitFor].
//
// # Memory Management & Pool Reuse
//
// To prevent excessive heap allocation churn and minimize Garbage Collector (GC)
// pressure in high-throughput network applications, [Manager] utilizes an internal,
// thread-safe object pool ([sync.Pool]) for recycling [Entry] structures.
// Once a job completes execution and is fully resolved, its state is wiped,
// its pointers are nil'ed out, and the structure is returned to the pool for reuse.
//
// # Concurrency & Safety Invariants
//
//   - Exclusive Manager Lock: The [Manager] protects all operations (including
//     all read and write operations on the underlying [Store] backend) using
//     a single, highly optimized mutex. Custom implementations of the [Store]
//     interface do not need to implement their own internal locking.
//   - Safe Callback Execution: Callbacks registered via [Manager.Add] are executed
//     asynchronously using a configurable [CallbackStrategy] (defaulting to [AsyncStrategy]).
//     This completely prevents deadlocks if a callback attempts to call back into
//     the same [Manager] instance.
//   - Safe Resource Cleanup: Upon resolution ([Manager.Resolve]), cancellation, or timeout,
//     all associated background resources (such as active [time.AfterFunc] timers or
//     [context.AfterFunc] subscription watchers) are guaranteed to be stopped and cleaned up
//     to prevent resource leaks.
//
// # Example - Callback Style
//
//	package main
//
//	import (
//		"context"
//		"fmt"
//		"log"
//		"sync"
//
//		"github.com/lemon4ksan/foundation/async/task"
//	)
//
//	func main() {
//		mgr := jobs.NewManager[string, string](100)
//		id := mgr.NextID()
//
//		var wg sync.WaitGroup
//		wg.Add(1)
//
//		err := mgr.Add(id, func(ctx context.Context, res string, err error) {
//			defer wg.Done()
//			if err != nil {
//				log.Printf("Job %s failed: %v", id, err)
//				return
//			}
//			fmt.Printf("Job %s received: %s\n", id, res)
//		})
//		if err != nil {
//			log.Fatalf("Failed to add job: %v", err)
//		}
//
//		// Simulate asynchronous response arrival
//		mgr.Resolve(id, "Hello, World!", nil)
//
//		wg.Wait()
//	}
//
// # Example - Blocking Style
//
//	package main
//
//	import (
//		"context"
//		"fmt"
//		"log"
//		"time"
//
//		"github.com/lemon4ksan/foundation/async/task"
//	)
//
//	func main() {
//		mgr := jobs.NewManager[string, string](0)
//		id := mgr.NextID()
//
//		// Configure with WithWait and a 1-second timeout limit
//		err := mgr.Add(id, nil, jobs.WithWait[string](), jobs.WithTimeout[string](time.Second))
//		if err != nil {
//			log.Fatalf("Failed to add job: %v", err)
//		}
//
//		// Block and wait for resolution synchronously
//		go func() {
//			time.Sleep(100 * time.Millisecond)
//			mgr.Resolve(id, "Hello from background!", nil)
//		}()
//
//		res, err := mgr.WaitFor(context.Background(), id)
//		if err != nil {
//			log.Fatalf("Job failed: %v", err)
//		}
//		fmt.Println("Result:", res)
//	}
package task
