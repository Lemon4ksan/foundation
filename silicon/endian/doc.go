// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package endian provides high-throughput zero-copy Little and Big Endian memory primitives
// using unsafe pointer indexing and modern Go intrinsics.
//
// # Compared to the standard library
//
// Stdlib counterpart: encoding/binary
//
// Rejected compromise: encoding/binary causes bound-checking overhead when dealing with slices, slowing down high-throughput hot paths.
//
// Accepted cost: Out-of-bounds pointer accesses will cause memory corruption and panic the Go runtime since bounds checking is bypassed.
//
// Allocations: zero
package endian
