// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package brotli provides an RFC 7932 compliant Brotli compression and decompression engine.
//
// The package implements a single-pass streaming decoder architecture with a
// 64-bit sliding bit-reader accumulator and a compact memory layout aligned for
// CPU cache locality. It uses a static RFC 7932 dictionary and a ring-buffer
// sliding window. The [Reader] and [Writer] allocate on construction, but can
// be fully reused across streams using their Reset methods to eliminate heap churn.
//
// This package is a port of github.com/andybalholm/brotli.
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: RFC 7932 Brotli compression
//
// Rejected compromise: none (Brotli is missing from stdlib)
//
// Accepted cost: the caller must manually manage [Reader] and [Writer] lifetime and call Reset to avoid allocations.
//
// Allocations: amortized
package brotli
