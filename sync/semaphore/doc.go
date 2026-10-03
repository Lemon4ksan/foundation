// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package semaphore provides a dynamically resizable counting semaphore.
//
// The package coordinates access to a shared pool of limited concurrent slots
// allowing the limit to be safely changed at runtime without restarting or
// rebuilding the coordinator. Context-aware acquisition ensures that waiting
// goroutines unblock immediately when their context is cancelled or expires,
// preventing resource leaks.
//
// # Compared to the standard library
//
// Stdlib counterpart: golang.org/x/sync/semaphore
//
// Rejected compromise: using a fixed-size buffered channel which cannot be resized
// dynamically, or standard library weighted semaphore which allocates under high contention.
//
// Accepted cost: each blocked goroutine requires allocating a channel in the waiter queue
// when the capacity limit is exceeded.
//
// Allocations: amortized
package semaphore
