// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package varint implements QUIC variable-length integer encoding and decoding (RFC 9000).
//
// varint utilizes assembly-optimized SIMD intrinsics when available to process variable-length integers with zero allocations. It bounds-checks rigorously to fail fast on malformed streams, critical for secure QUIC transport.
//
// See [Read], [Parse], and [Append] for usage.
//
// # Compared to the standard library
//
// Stdlib counterpart: encoding/binary (specifically binary.Uvarint)
//
// Rejected compromise: binary.Uvarint implements standard protobuf varints rather than the QUIC-specific encoding format (RFC 9000).
//
// Accepted cost: Tightly coupled to QUIC's specific maximum bounds and format, making it unsuitable for general protobuf varints.
//
// Allocations: zero
package varint
