// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package pool provides dynamic, auto-scaling worker pools with panic recovery
// and type-safe future results.
//
// A worker pool automatically scales the number of active goroutines up to a maximum
// bound when the internal queue fills, and scales them down to a minimum baseline
// during periods of idleness. This balances throughput capacity during traffic spikes
// against memory usage during quiet periods. It gracefully catches panics within
// worker tasks to prevent the pool from degrading over time.
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: dynamic scaling worker pool
//
// Rejected compromise: static worker pools maintain a fixed number of goroutines regardless of workload, consuming system memory during idle periods and bottlenecking throughput during sudden traffic spikes.
//
// Accepted cost: Every submitted task allocates a new Future to return the result asynchronously, and synchronization primitives add overhead to dispatching.
//
// Allocations: bounded(58/op)
package pool
