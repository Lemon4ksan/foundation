// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package timekit provides high-throughput, zero-allocation coarse monotonic clocks,
// branchless date formatting (HTTP-date, RFC 3339, ISO 8601), and high-precision
// stopwatch utilities for performance-critical server backends.
//
// # Architectural Philosophy
//
// In high-concurrency servers processing hundreds of thousands of requests per second,
// invoking [time.Now] for every log entry, header, or rate-limit check incurs significant
// syscall overhead (such as clock_gettime on Linux or QueryPerformanceCounter on Windows).
//
// [timekit] addresses this with:
//   - [CoarseNow]: An atomic monotonic clock refreshed continuously in the background.
//   - [AppendHTTPDate]: Zero-allocation RFC 7231 / RFC 9110 HTTP date serialization.
//   - [AppendRFC3339]: Zero-allocation RFC 3339 / ISO 8601 timestamp generator.
//   - [Stopwatch]: Monotonic stopwatch for micro-profiling and latency measurements.
//
// # Concurrency & Thread-Safety
//
// All clock reading and formatting functions in [timekit] are fully thread-safe,
// lock-free, and safe for concurrent use across arbitrary numbers of goroutines.
package timekit
