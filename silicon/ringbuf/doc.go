// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package ringbuf provides lock-free Single-Producer Single-Consumer (SPSC)
// and Multi-Producer Multi-Consumer (MPMC) ring buffers.
//
// Standard buffered Go channels rely on internal mutex locking for every enqueue
// and dequeue operation. Under multi-million message-per-second throughput across
// multiple CPU cores, channel lock contention causes severe cache-line bouncing
// and scheduler latency. Lock-free ring buffers with cache-line padding eliminate
// lock contention and false sharing, maximizing L1/L2 cache locality.
// SPSC ring buffers implement local cursor caching (cached tail on producer,
// cached head on consumer) to eliminate cache-line bouncing between producer
// and consumer cores until capacity boundaries are reached.
//
// The primary entry points are [New] for MPMC [Buffer] and [NewSPSC] for [SPSC].
//
// # Compared to the standard library
//
// Stdlib counterpart: channels
//
// Rejected compromise: Standard buffered Go channels rely on internal mutex locking for every enqueue and dequeue operation, causing contention under high throughput.
//
// Accepted cost: Ring buffers must have power-of-two capacities, and they lack the select statement integration that standard channels provide.
//
// Allocations: amortized
package ringbuf
