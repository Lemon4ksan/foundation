// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package scheduler provides a unified, precision-timed task execution loop.
//
// The [Scheduler] maintains a priority-sorted execution list of independent tasks.
// Instead of dedicating a ticker and a goroutine to every background job, tasks are
// scheduled into a central heap. A single coordinator loop sleeps exactly until the
// next task is due, avoiding CPU busy-wait and preventing uncoordinated timer drift.
// When a task deadline arrives, it is dispatched to a background goroutine for
// execution, ensuring the scheduler loop itself is never blocked by task payloads.
//
// Tasks are managed via a memory pool using [Scheduler.AcquireTask] and
// [Scheduler.ReleaseTask] to eliminate garbage collection overhead during
// high-frequency rescheduling.
//
// # Compared to the standard library
//
// Stdlib counterpart: time.Ticker and time.AfterFunc
//
// Rejected compromise: individual time.Ticker instances manage their own background timer goroutines, consuming system resources and drifting under load without centralized coordination.
//
// Accepted cost: all tasks are dispatched onto new background goroutines when executed, meaning a flood of tasks arriving simultaneously will spawn an equal number of goroutines.
//
// Allocations: amortized
package scheduler
