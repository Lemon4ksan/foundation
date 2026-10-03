// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package bitset implements a dense, memory-efficient bit array backed by []uint64 words.
//
// BitSet represents a growable bit array offering fast set, clear, and hardware-accelerated population counts. It is designed to minimize pointer indirection and GC tracking overhead by using a flat, contiguous word slice. Read-only queries are safe for concurrent use across multiple goroutines, but mutations must be synchronized.
//
// The primary entry points are [New], [BitSet.Set], and [BitSet.Test].
//
// # Compared to the standard library
//
// Stdlib counterpart: math/big - specifically big.Int used as a bitset
//
// Rejected compromise: math/big.Int allocates a new heap-allocated slice when operations return a new big.Int instead of mutating in-place, and lacks an explicit, ergonomic bitset API.
//
// Accepted cost: Manual synchronization is required for concurrent access, and slice capacities must be managed manually to avoid over-allocation.
//
// Allocations: amortized
package bitset
