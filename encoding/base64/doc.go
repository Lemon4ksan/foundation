// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package base64 provides high-performance and SIMD-accelerated Base64 encoding.
//
// Base64 prioritizes speed and zero-allocation processing for URL-safe unpadded encodings. It acts as a fast-path override for high-throughput network transmission where standard library encoding becomes a bottleneck.
//
// See [Base64EncodeURL] for the primary encoding function.
//
// # Compared to the standard library
//
// Stdlib counterpart: encoding/base64
//
// Rejected compromise: encoding/base64 processes data byte-by-byte in pure Go without SIMD acceleration, reducing throughput.
//
// Accepted cost: Platform-specific assembly means fallback paths are necessary for unsupported architectures.
//
// Allocations: zero
package base64
