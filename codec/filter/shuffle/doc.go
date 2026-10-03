// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package shuffle implements a high-performance byte transposition (shuffle) filter.
//
// Shuffling groups bytes of identical significance (such as IEEE-754 sign/exponent bytes)
// from arrays of multi-byte types (e.g. FP16, FP32, ML tensors, safetensors, time-series)
// so they are contiguous. This dramatically improves entropy and LZ-based compression
// ratios because bytes in the same position of structured numeric data are highly correlated.
// Use [Encode] to transpose data before compression, and [Decode] to restore it.
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: multi-byte transposition filters for structured data.
//
// Rejected compromise: standard library tools only see raw byte streams, producing poor compression density on floating-point or multi-byte integer arrays.
//
// Accepted cost: the caller must specify the byte width of elements and manage memory slices manually.
//
// Allocations: zero
package shuffle
