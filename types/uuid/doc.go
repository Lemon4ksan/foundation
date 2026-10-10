// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package uuid implements Universally Unique Identifiers strictly conforming to RFC 9562.
//
// UUIDs are represented as a compact 16-byte array ([UUID]) in big-endian network byte order.
// The package provides generation, validation, and serialization routines for both random (UUIDv4) and
// time-ordered (UUIDv7) identifiers. [Parse], [UUID.Format], and [UUID.Append] are designed
// for high-throughput pipelines and use SIMD hardware acceleration when available to avoid heap allocations.
// Note that while the core parsing and formatting methods allocate zero bytes, the [UUID.String]
// method inevitably allocates once to return a native Go string.
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: standard UUID generation and validation
//
// Rejected compromise: incurring heap allocations during high-frequency parsing and formatting of UUID strings
//
// Accepted cost: manual buffer management via Format/Append to achieve zero allocations
//
// Allocations: zero
package uuid
