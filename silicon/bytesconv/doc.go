// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package bytesconv provides zero-copy string-to-byte conversions, expandable byte buffers,
// buffered stream readers, zero-allocation token/pair scanners, slice splitters,
// case-folded ASCII searches, number formatting, and hashing utilities for hot paths.
//
// To guarantee string immutability, the Go runtime allocates a new heap copy on
// every string(b) or []byte(s) conversion. While safe, this behavior produces
// continuous garbage collector pressure when parsing high-frequency streams.
// Package bytesconv provides zero-copy conversions ([B2S], [S2B]), pooled expandable
// buffers ([ByteBuffer]), SIMD-aligned buffered readers ([StreamReader]), zero-allocation
// iterators ([ScanTokens], [ScanPairs]), pattern splitters ([PatternSlicer]), and
// zero-copy stream copying ([CopyZeroAlloc]).
//
// # Compared to the standard library
//
// Stdlib counterpart: strings, bytes, bufio, strconv
//
// Rejected compromise: The Go runtime allocating a new heap copy on every string(b) or []byte(s) conversion, and standard splitters allocating a new slice of string descriptors on every invocation.
//
// Accepted cost: String immutability guarantees are lost when derived from mutable byte slices, and slicers hold references to the original backing array preventing it from being garbage collected.
//
// Allocations: amortized
package bytesconv
