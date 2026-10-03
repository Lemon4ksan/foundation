// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package bufkit provides zero-allocation, cacheline-aligned memory buffer primitives, chunked scatter-gather chains, and lock-free ring buffers.
//
// Traditional memory buffers in Go rely heavily on continuous reallocation when growing dynamic payloads, which triggers memory copies and garbage collector churn. This package eliminates reallocation overhead by introducing scatter-gather chunked buffers that chain fixed-size pooled memory blocks, lock-free circular buffers optimized for high-throughput single-producer single-consumer pipelines, and memory buffers aligned to processor cachelines for vector operations.
//
// # Compared to the standard library
//
// Stdlib counterpart: bytes.Buffer
//
// Rejected compromise: Continuous heap allocations and copying for dynamically sized buffers.
//
// Accepted cost: Manual memory pooling and fragmented chunk management overhead.
//
// Allocations: zero
package bufkit
