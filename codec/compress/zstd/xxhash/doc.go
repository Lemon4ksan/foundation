// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package xxhash implements the 64-bit variant of xxHash (XXH64) as described
// at http://cyan4973.github.io/xxHash/.
//
// This package is a high-performance, pure Go implementation with architecture-specific
// assembly for amd64 and arm64. It is designed to be much faster than the standard library
// hash algorithms while producing a high-quality 64-bit hash.
//
// The core [Sum64] function operates directly on byte slices, while the [Digest] type
// implements the hash.Hash64 interface for streaming data.
//
// VENDORED: Originates from github.com/cespare/xxhash.
//
// # Compared to the standard library
//
// Stdlib counterpart: hash/crc64
//
// Rejected compromise: hash/crc64 uses lookup tables which are slower than pure arithmetic and SIMD hashing.
//
// Accepted cost: Architecture-specific assembly files for maximum performance.
//
// Allocations: zero
package xxhash
