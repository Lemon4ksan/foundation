// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package bytesconv provides zero-allocation unsafe string-to-byte conversions,
// high-speed scanners, and slice splitters for performance-critical hot paths.
//
// To guarantee string immutability, the Go runtime allocates a new heap copy on
// every string(b) or []byte(s) conversion. While safe, this behavior produces
// continuous garbage collector pressure when parsing high-frequency data.
// This package implements conversion using modern Go compiler intrinsics
// and provides a slicer iterator that maintains a cursor pointer into the underlying
// byte array, returning subslices without allocating string or slice descriptors on the heap.
//
// # Compared to the standard library
//
// Stdlib counterpart: strings, bytes
//
// Rejected compromise: The Go runtime allocating a new heap copy on every string(b) or []byte(s) conversion, and standard splitters allocating a new slice of string descriptors on every invocation.
//
// Accepted cost: String immutability guarantees are lost when derived from mutable byte slices, and slicers hold references to the original backing array preventing it from being garbage collected.
//
// Allocations: zero
package bytesconv
