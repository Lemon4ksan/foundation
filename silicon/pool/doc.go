// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package pool provides sharded object pools, inline bump-allocation arenas, and timer
// recycling utilities for low-allocation hot paths.
//
// Standard sync.Pool objects are subject to reclamation by the Go runtime during
// garbage collection cycles, causing periodic allocation spikes immediately following a
// collection. Package pool provides:
//   - [PerCorePool]: a cache-line-padded, power-of-two sharded object pool that retains
//     pooled items across GC cycles and minimizes cross-core lock contention.
//   - [RequestArena] ([AcquireRequestArena] / [ReleaseRequestArena]): an inline 4KB
//     8-byte-aligned pointer-bump arena with O(1) reset for short-lived byte slices.
//   - [AcquireTimer] / [ReleaseTimer]: safe [*time.Timer] pooling with non-blocking
//     stop-and-drain channel cleanup.
//
// # Compared to the standard library
//
// Stdlib counterpart: sync.Pool
//
// Rejected compromise: sync.Pool reclaims memory during garbage collection cycles
// causing allocation spikes, and allocates per-object rather than per-arena.
//
// Accepted cost: Memory in PerCorePool is retained across GC cycles up to the bounded
// shard capacity, and RequestArena requires explicit lifecycle management by the
// caller.
//
// Allocations: amortized
package pool
