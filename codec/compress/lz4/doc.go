// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package lz4 implements reading and writing lz4 compressed data.
//
// The package supports both the LZ4 stream format and the LZ4 block format.
// It provides high-speed compression and decompression, offering a different
// tradeoff between ratio and speed compared to deflate. Functions like
// [CompressBlock] and [UncompressBlock] operate on pre-allocated buffers
// to avoid allocations.
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: LZ4 compression codec.
//
// Rejected compromise: the standard library lacks high-speed compression algorithms, relying solely on deflate and gzip which prioritize ratio over throughput.
//
// Accepted cost: maintaining an additional codec implementation.
//
// Allocations: amortized
package lz4
