// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package pipeline provides concurrent parallel mapping, worker fan-out/fan-in,
// token-bucket rate limiting, strict order preservation, and time-window bulk batching.
//
// Processing large slices of data concurrently requires careful concurrency budgeting
// and rate limiting to avoid overwhelming downstream services. This package simplifies
// these workflows by providing the [Map] and [ForEach] functions which distribute work
// across a fixed pool of goroutines, preserving the original order of the input slice
// without the data races associated with manual wait groups and uncoordinated channels.
//
// # Compared to the standard library
//
// Stdlib counterpart: sync.WaitGroup and unbuffered channels
//
// Rejected compromise: sync.WaitGroup and channel fan-out scramble the original slice ordering, necessitating manual index bookkeeping and locking to reconstruct results.
//
// Accepted cost: Background worker goroutines are spawned, and slice arrays and channels are allocated per pipeline run.
//
// Allocations: bounded(26/op)
package pipeline
