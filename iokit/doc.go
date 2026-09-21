// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package iokit provides high-performance streaming I/O wrappers, replayable body decorators,
// and zero-allocation copy helpers designed for high-throughput networking and proxy systems.
//
// # Architecture
//
// The package revolves around stream efficiency, memory bounds enforcement, and zero-allocation pipelines:
//   - Replayable Streams: [MultiReadBody] wraps any [io.ReadCloser], caching read payloads in memory
//     up to a configurable threshold, spilling over to temporary disk storage if permitted, or rejecting
//     oversized payloads via [ErrBufferLimitExceeded]. Calling Close() resets the read cursor to permit replay,
//     while ReallyClose() irrevocably purges off-heap buffers and unlinks spill files.
//   - Volatile In-Memory Readers: [BytesReader] implements an optimized [io.Reader] over raw byte slices.
//     When marked volatile, callers are explicitly signaled that the underlying memory is reused across iterations.
//   - Zero-Allocation Stream Copying: [CopyZeroAlloc] copies data between [io.Reader] and [io.Writer]
//     leveraging internal 32KB buffer pooling ([sync.Pool]) to eliminate heap churn.
//   - Boundary & Keep-Alive Protection: [LimitToContentLength] caps reader streams at exact Content-Length
//     boundaries, preventing socket poisoning from trailing garbage in HTTP Keep-Alive connections.
//   - Content Transformation: Transparent BOM stripping ([NewBOMSnifferReader]), counting wrappers
//     ([NewCountingReader], [NewCountingWriter]), and rate-monitored progress hooks ([ProgressFunc]).
//
// # Design Rationale
//
// Standard library I/O primitives often allocate intermediate buffers on each invocation (e.g. [io.Copy]
// allocating 32KB slices if buffer interfaces are absent). In high-concurrency microservices, repeated buffer
// allocations create severe GC pressure. [iokit] ensures that common I/O transformations—capping, counting,
// re-reading, and copying—execute with zero or strictly bounded heap allocations.
//
// # Concurrency Guarantees
//
//   - Buffer pools and stateless transformation functions are thread-safe and safe for concurrent use.
//   - Individual stream instances ([MultiReadBody], [BytesReader], [CountingReader]) are not safe for
//     concurrent reading from multiple goroutines simultaneously without external synchronization.
//   - For [BytesReader] instances marked volatile, callers must not retain or transmit the backing slice
//     to other goroutines without explicitly cloning it.
//
// # Example
//
//	package main
//
//	import (
//		"bytes"
//		"fmt"
//		"io"
//
//		"github.com/lemon4ksan/foundation/iokit"
//	)
//
//	func main() {
//		data := []byte("payload to inspect and replay")
//		body := iokit.NewMultiReadBody(io.NopCloser(bytes.NewReader(data)), 1024, false)
//		defer body.ReallyClose()
//
//		// First pass: inspect prefix
//		prefix := make([]byte, 7)
//		_, _ = io.ReadFull(body, prefix)
//		fmt.Printf("Prefix: %s\n", prefix)
//
//		// Reset cursor for full downstream replay
//		_ = body.Close()
//		all, _ := io.ReadAll(body)
//		fmt.Printf("Replayed: %s\n", all)
//	}
package iokit
