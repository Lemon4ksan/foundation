// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package urlkit provides high-speed URL parsing, serialization, and path variable expansion.
//
// This package is designed for extreme throughput. It offers [ParseView] for parsing URLs directly
// from string slices without allocations, and relies on SIMD instructions and CRC32 sharded caches
// to accelerate query parameter unescaping and URL string assembly.
//
// # Compared to the standard library
//
// Stdlib counterpart: net/url
//
// Rejected compromise: net/url.Parse heavily allocates by creating standard library pointers, URL structures, and unescaped strings up front.
//
// Accepted cost: the [URLView] relies on underlying string slices, meaning the caller must manage lifetimes, and some URL edge cases might behave slightly differently than the standard library on malformed inputs.
//
// Allocations: zero
package urlkit
