// Copyright (c) 2018-2023 Klaus Post. All rights reserved.
// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// Based on work Copyright (c) 2013, Yann Collet, released under BSD License.

// Package fse provides Finite State Entropy encoding and decoding.
//
// Finite State Entropy encoding provides a fast near-optimal symbol encoding and decoding
// for byte blocks. It is primarily used as the entropy coder in zstd. The package
// uses [Scratch] as an explicit working space to avoid memory allocations on the hot path.
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: Finite State Entropy (FSE) codec for advanced compression.
//
// Rejected compromise: the standard library restricts itself to older entropy codecs (Huffman) lacking the speed and ratio tradeoff of FSE.
//
// Accepted cost: manual memory management of [Scratch] buffers and complexity of state table initialization.
//
// Allocations: amortized
package fse
