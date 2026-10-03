// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package bcj implements Branch-Call-Jump (BCJ) bytecode pre-filters for executable binaries.
//
// BCJ filters improve the entropy and compression ratios of compiled binaries
// (like x86, ARM, or PowerPC executables) by converting relative branch targets
// (e.g. x86 E8/E9 CALL/JMP instructions) into absolute addresses before compression,
// and reverting them after decompression. This creates repeated identical instruction
// sequences that LZMA, Deflate, or Zstandard can compress more efficiently.
// The filter operates in-place on the provided byte slice. Use [Filter] to apply
// or revert the transformation.
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: architecture-specific bytecode pre-filters.
//
// Rejected compromise: standard library compression packages treat all data as opaque bytes, missing out on significant compression gains for executable code.
//
// Accepted cost: the caller must identify the target CPU architecture and invoke the filter manually before compression and after decompression.
//
// Allocations: zero
package bcj
