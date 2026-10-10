// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package gfni provides Galois Field GF(2^8) arithmetic operations
// modulo the irreducible polynomial x^8 + x^4 + x^3 + x + 1 (0x11B).
//
// It provides software implementations of field multiplication and vector
// scaling, acting as the portable reference and fallback substrate for
// hardware VGF2P8MULB instructions.
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: Galois Field arithmetic primitives for erasure coding and cryptography
//
// Rejected compromise: Relying exclusively on hardware instructions breaks compatibility on platforms lacking GFNI/AVX512 support.
//
// Accepted cost: Software shift-and-add execution (Russian Peasant) is slower than single-cycle vector hardware instructions.
//
// Allocations: zero
package gfni
