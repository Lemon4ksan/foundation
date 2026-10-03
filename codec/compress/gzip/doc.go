// Copyright 2009 The Go Authors. All rights reserved.
// Copyright (c) 2015-2023 Klaus Post. All rights reserved.
// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package gzip implements reading and writing of gzip format compressed files,
// as specified in RFC 1952.
//
// This package is based on the standard library compress/gzip and
// github.com/klauspost/compress/gzip, utilizing a faster flate compressor
// underneath. It provides [Reader] and [Writer] for streaming compression
// and decompression.
//
// # Compared to the standard library
//
// Stdlib counterpart: compress/gzip
//
// Rejected compromise: the standard library's gzip does not optimize aggressively for zero-allocation steady state, leading to heap allocations during streaming.
//
// Accepted cost: increased complexity in internal state management and pooling.
//
// Allocations: amortized
package gzip
