// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package refkit provides zero-allocation reflection inspection helpers and high-speed struct tag parsing.
//
// Standard Go reflection frequently introduces performance bottlenecks in schema builders and serialization libraries due to string parsing overhead and interface allocations. This package optimizes introspection by parsing complex struct tags into structured types with support for key-value extraction and numeric conversion, and by providing panic-safe, zero-allocation type checking across all Go kinds.
//
// # Compared to the standard library
//
// Stdlib counterpart: reflect
//
// Rejected compromise: Panics on kind mismatches and string-parsing overhead on every struct tag inspection.
//
// Accepted cost: Parsing overhead for complex nested options.
//
// Allocations: bounded(N/op)
package refkit
