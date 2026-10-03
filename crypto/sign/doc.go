// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package sign provides Ed25519 digital signature generation, verification, and PEM key marshaling.
//
// sign wraps Go's native Ed25519 implementation, exposing a high-throughput API for signature schemas. It strictly enforces standard key lengths and handles standard serialization formats for secure key management.
//
// See [Sign], [Verify], and [GenerateKey] for core operations.
//
// # Compared to the standard library
//
// Stdlib counterpart: crypto/ed25519
//
// Rejected compromise: crypto/ed25519 lacks integrated PEM serialization and ergonomic fingerprinting functions out-of-the-box.
//
// Accepted cost: Hardcodes Ed25519 as the primary scheme, making it less flexible if other signature algorithms are required.
//
// Allocations: zero
package sign
