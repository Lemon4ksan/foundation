// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package dedup provides generic duplicate call suppression with context awareness and panic isolation.
//
// It solves the "thundering herd" problem by ensuring that duplicate concurrent
// requests with the same key execute only once. Subsequent callers wait for the
// active execution to complete and receive the same result. The [Group] maintains
// an internal map of in-flight calls. When a new request arrives for an already
// in-flight key, the caller blocks until the original worker completes.
// If a worker function panics, the panic is re-propagated only to the goroutine
// that initiated the call. All secondary waiters receive [ErrWorkerPanicked],
// preventing a single panicking worker from crashing the entire pool.
//
// # Compared to the standard library
//
// Stdlib counterpart: golang.org/x/sync/singleflight
//
// Rejected compromise: singleflight panics all waiting callers when the primary worker panics.
//
// Accepted cost: A more complex internal state machine and a goroutine per request to isolate panics.
//
// Allocations: bounded(2/op)
package dedup
