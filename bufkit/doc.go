// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package bufkit provides zero-allocation, cacheline-aligned memory buffer
// primitives, chunked scatter-gather chains, and lock-free ring buffers designed
// for high-throughput networking, streaming, and I/O pipelines.
//
// # Architectural Philosophy
//
// Traditional memory buffers in Go rely heavily on continuous reallocation when
// growing dynamic payloads (such as HTTP/2 frames, WebSocket frames, or serialized RPC messages).
// Continuous resizing triggers memory copies and garbage collector churn.
//
// [bufkit] eliminates reallocation overhead by introducing:
//   - [Chain]: A scatter-gather chunked buffer that chains fixed-size pooled memory blocks.
//   - [Ring]: A lock-free circular buffer optimized for high-throughput single-producer single-consumer (SPSC) pipelines.
//   - [AlignedBytes]: Memory buffers aligned to processor cachelines (64 bytes) or pages (4096 bytes) for vector/SIMD operations.
//
// # Concurrency & Thread-Safety
//
//   - [Chain] is designed for single-goroutine assembly and multi-chunk streaming reads.
//   - [Ring] provides lock-free thread-safe enqueue/dequeue between concurrent goroutines.
//   - Pooled memory structures are recycled via thread-safe synchronization pools.
package bufkit
