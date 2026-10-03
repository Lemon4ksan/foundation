// Copyright (c) 2018-2023 Klaus Post. All rights reserved.
// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// Based on work Copyright (c) 2013, Yann Collet, released under BSD License.

// Package huff0 provides fast Huffman encoding and decoding.
//
// Huff0 prefix entropy encoding provides fast symbol encoding and decoding
// for byte blocks. It is primarily used as the entropy coder in zstd. The package
// uses [Scratch] as an explicit working space to avoid memory allocations on the hot path.
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: Huff0 prefix entropy codec.
//
// Rejected compromise: the standard library lacks specialized, zero-allocation prefix codecs optimized for high-throughput block compression.
//
// Accepted cost: manual memory management of [Scratch] buffers and explicit block limits.
//
// Allocations: amortized
package huff0
