// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package clock provides nanosecond-precision monotonic time tracking without OS syscall overhead.
//
// Querying system time via time.Now on millions of operations incurs operating system
// vDSO and syscall overhead. Cached timestamp tickers update timestamps in background
// goroutines, reducing timestamp reads to a single atomic load for high-concurrency
// loops where microsecond precision is sufficient.
//
// # Compared to the standard library
//
// Stdlib counterpart: time
//
// Rejected compromise: Querying system time via time.Now incurs operating system vDSO and syscall overhead on millions of operations.
//
// Accepted cost: Cached timestamps are slightly stale compared to the actual system time, trading exact precision for microsecond-level precision and reduced overhead.
//
// Allocations: zero
package clock
