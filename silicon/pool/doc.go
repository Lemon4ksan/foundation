// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package pool provides high-performance memory arenas, perpetual byte storages,
// and tiered object pools for zero-allocation byte handling.
//
// Standard sync.Pool objects are subject to reclamation by the Go runtime during
// garbage collection cycles, causing periodic allocation spikes immediately following
// a collection. Furthermore, network protocol parsers that allocate numerous small
// metadata chunks per frame incur high per-object allocation overhead. Contiguous
// memory arenas allow allocating many small objects in a single contiguous block
// with instant O(1) reclamation via pointer reset.
//
// # Compared to the standard library
//
// Stdlib counterpart: sync.Pool
//
// Rejected compromise: sync.Pool reclaims memory during garbage collection cycles causing allocation spikes, and allocates per-object rather than per-arena.
//
// Accepted cost: Memory is manually managed and held until the arena is explicitly reset, requiring careful lifecycle management by the caller.
//
// Allocations: amortized
package pool
