// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package iokit provides high-performance streaming I/O wrappers.
//
// The package offers low-level streaming primitives, replayable body decorators,
// multi-read buffers, and pooled zero-allocation copy helpers designed for line-rate data processing.
// A common use case is wrapping an incoming request body in a [MultiReadBody] to inspect
// payload contents (e.g. for cryptographic hashing or routing) before resetting the stream
// and passing it to the final handler.
//
// # Compared to the standard library
//
// Stdlib counterpart: io / bytes
//
// Rejected compromise: allocating fresh 32KB buffers for every stream copy and forcing full payload materialization in memory for replay
//
// Accepted cost: manual buffer pooling management and explicit ReallyClose() calls for disk-backed streams
//
// Allocations: bounded(1/op)
package iokit
