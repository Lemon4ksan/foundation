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
// # Compared to the standard library
//
// Stdlib counterpart: manual maps and unpooled goroutines
//
// Rejected compromise: manual maps and per-request timer goroutines introduce significant scheduler churn and garbage collector overhead under high concurrency.
//
// Accepted cost: tasks are managed centrally with a mutex, introducing lock contention under extreme concurrency.
//
// Allocations: bounded(4/op)
package task
