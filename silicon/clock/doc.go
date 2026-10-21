// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package clock provides hardware CPU cycle counters and a cached coarse wall clock
// without OS syscall overhead on hot paths.
//
// Querying system time via [time.Now] on millions of operations incurs operating system
// vDSO and syscall overhead. Package clock provides two complementary mechanisms:
//   - Hardware cycle counting ([RDTSC], [Cycles], [CyclesToDuration], [ElapsedCycles])
//     using serialized CPU timestamp instructions (RDTSC on amd64, CNTVCT_EL0 on arm64)
//     calibrated against the monotonic clock at startup for sub-nanosecond elapsed
//     timing.
//   - A background ticker updating cached Unix timestamps every ~1ms ([CoarseNowNano],
//     [CoarseNowUnix], [CoarseTime], [CoarseSince]), reducing wall-clock reads to a
//     single atomic load from L1 cache where millisecond resolution is sufficient.
//
// # Compared to the standard library
//
// Stdlib counterpart: time
//
// Rejected compromise: Querying system time via time.Now incurs operating system vDSO
// and syscall overhead on millions of operations.
//
// Accepted cost: Coarse timestamps have ~1ms resolution and are slightly stale compared
// to actual system time, while RDTSC cycle conversion relies on startup frequency
// calibration.
//
// Allocations: zero
package clock
