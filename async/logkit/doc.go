// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package logkit provides an asynchronous, structured logger designed to keep
// application threads running at line-rate during I/O stalls.
//
// The [Logger] interface defines standard structured logging methods. When using
// the default [AsyncLogger], log messages are formatted into a pooled buffer
// on the calling goroutine, then pushed to a fixed-size channel queue. A dedicated
// background worker reads from the queue and writes to the destination. If the
// queue fills up (for example, during disk writes stalling), new messages are
// dropped to protect application liveness rather than blocking the caller.
//
// Structured context is added using the [Logger.With] method, which accepts
// strongly-typed attributes like [String] and [Int].
//
// # Compared to the standard library
//
// Stdlib counterpart: log/slog
//
// Rejected compromise: log/slog blocks the calling goroutine to format and write the log record synchronously, causing I/O latency spikes to stall application throughput.
//
// Accepted cost: Logs are queued in a channel, which drops messages if the queue fills up during disk stalls, meaning logs can be lost under extreme pressure.
//
// Allocations: bounded(9/op)
package logkit
