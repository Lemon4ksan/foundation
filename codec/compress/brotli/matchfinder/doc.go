// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package matchfinder defines reusable components and interfaces for data compression match-finding.
//
// The LZ77 stage of compression (finding repeated byte sequences) and the entropy
// coding stage (encoding the compressed format) are traditionally tightly coupled.
// This package provides a generic [MatchFinder] interface, an [Encoder] interface,
// and a [Match] struct as an intermediate representation. This allows mixing and
// matching different match-finders (e.g., hash-based, suffix trees) with different
// encoders (e.g., deflate, snappy).
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: modular match-finding for LZ77 compression
//
// Rejected compromise: stdlib flate tightly couples its match-finding logic with its DEFLATE encoder, preventing reuse of either half.
//
// Accepted cost: an intermediate slice of [Match] structs and interface dispatch overhead between the match-finder and the encoder.
//
// Allocations: zero
package matchfinder
