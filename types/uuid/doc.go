// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package uuid implements Universally Unique Identifiers strictly conforming to RFC 9562 (obsoleting RFC 4122).
//
// # Architecture
//
// The package models UUIDs as a compact 16-byte array ([UUID]) in big-endian network byte order,
// providing generation, validation, and serialization routines for both random and time-ordered identifiers:
//   - UUIDv4 (Random): Cryptographically secure pseudo-random UUIDs conforming to RFC 9562 §5.4.
//   - UUIDv7 (Time-Ordered): Monotonically increasing, time-ordered UUIDs conforming to RFC 9562 §5.7,
//     featuring a 48-bit Unix epoch millisecond timestamp prefix, ideal for database primary keys and B-tree locality.
//   - SIMD Hardware Acceleration: Vectorized parsers leveraging AMD64 AVX/SSE instruction sets
//     to validate and decode 36-character hexadecimal strings into binary byte arrays with zero heap allocations.
//   - Database Interoperability: Native implementation of [driver.Valuer] and [sql.Scanner] for seamless SQL persistence.
//
// # Design Rationale
//
// Traditional UUID libraries incur multiple allocations during formatting, parsing, and RNG sampling.
// This package is engineered for ultra-high throughput environments:
//   - Strictly zero heap allocations for formatting ([UUID.String], [UUID.AppendTo]) and parsing ([Parse], [ParseBytes]).
//   - Branchless, vectorized ASCII-to-nibble translation paths on supported architectures.
//   - Sentinel constants [Nil] and [Max] for boundary checking without runtime overhead.
//
// # Concurrency Guarantees
//
// All functions and methods in this package—including [NewV4], [NewV7], [Parse], and string formatting methods—are
// fully thread-safe and safe for concurrent execution across arbitrary numbers of goroutines. The [UUID] type
// is a value type and can be passed, copied, or stored concurrently without race conditions.
//
// # Example
//
//	package main
//
//	import (
//		"fmt"
//
//		"github.com/lemon4ksan/foundation/types/uuid"
//	)
//
//	func main() {
//		// Generate time-ordered UUIDv7
//		id, err := uuid.NewV7()
//		if err != nil {
//			panic(err)
//		}
//		fmt.Println("Generated UUIDv7:", id.String())
//
//		// Parse existing UUID string with zero allocations
//		parsed, err := uuid.Parse("018e3a2b-7c4d-7000-8000-000000000000")
//		if err != nil {
//			panic(err)
//		}
//		fmt.Println("Parsed Version:", parsed.Version())
//	}
package uuid
