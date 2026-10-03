// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package flate implements the DEFLATE compressed data format specification (RFC 1951).
//
// The package is optimized with SIMD-accelerated match copying (128-bit wildcopy)
// and a 32-byte SWAR match comparator that utilizes parallel CPU execution ports.
// Static precomputed Huffman tables are generated during init to remove atomic
// memory barriers. Dedicated fast paths exist for repeating byte sequences.
//
// This package is a fork of github.com/klauspost/compress/flate.
//
// # Compared to the standard library
//
// Stdlib counterpart: compress/flate
//
// Rejected compromise: stdlib compress/flate allocates heavily and guards initialization with sync.Once; this package precomputes tables and uses Per-P storage to avoid GC overhead.
//
// Accepted cost: increased binary size from SIMD assembly and statically initialized Huffman tables.
//
// Allocations: zero
package flate
