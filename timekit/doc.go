// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package timekit provides high-throughput, zero-allocation time reading and branchless date formatting.
//
// In high-concurrency servers, invoking time.Now triggers operating system vDSO and syscall overhead.
// This package resolves this with a lock-free atomic monotonic clock continuously refreshed by a
// background ticker ([CoarseNow]), branchless zero-allocation timestamp generators
// ([AppendRFC3339], [AppendISO8601], [AppendHTTPDate]), and a high-precision [Stopwatch].
//
// # Compared to the standard library
//
// Stdlib counterpart: time
//
// Rejected compromise: time.Now dynamically invokes vDSO or syscalls, and time.Time.Format allocates memory.
//
// Accepted cost: CoarseNow loses nanosecond precision, giving only millisecond-level accuracy.
//
// Allocations: zero
package timekit
