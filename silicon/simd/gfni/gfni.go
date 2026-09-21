// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package gfni provides Galois Field GF(2^8) arithmetic and vector multiplication primitives
// modeled after Intel/AMD GFNI instruction set extensions (specifically VGF2P8MULB).
//
// All field operations use the standard AES/Rijndael irreducible polynomial:
//
//	m(x) = x^8 + x^4 + x^3 + x + 1 (0x11B)
//
// All functions in this package are stateless, pure, concurrency-safe, and execute with
// strictly zero heap memory allocations.
package gfni

// MultiplyGF2P8 multiplies two 8-bit field elements a and b in the Galois Field GF(2^8)
// modulo the irreducible polynomial x^8 + x^4 + x^3 + x + 1 (0x11B).
//
// This function implements the Russian Peasant (xtime) shift-and-add algorithm and serves
// as the portable, architecture-independent reference implementation and software fallback
// for the hardware VGF2P8MULB SIMD instruction.
//
// Concurrency: Pure function, safe for concurrent use by multiple goroutines.
// Allocation: Strictly 0 allocs/op, zero heap overhead.
func MultiplyGF2P8(a, b byte) byte {
	var p byte = 0
	for range 8 {
		if (b & 1) == 1 {
			p ^= a
		}
		hiBitSet := (a & 0x80) != 0
		a <<= 1
		if hiBitSet {
			a ^= 0x1B // GF(2^8) reduction polynomial x^8 + x^4 + x^3 + x + 1
		}
		b >>= 1
	}
	return p
}

// MultiplyGF2P8Vector multiplies each element in the input vector src by scalar in GF(2^8)
// and writes the resulting field elements into dst.
//
// If len(src) is 0, the function returns immediately without modifying dst (safe when dst is nil).
// If len(dst) < len(src), MultiplyGF2P8Vector panics with a slice bounds check out of range.
// dst and src may alias the same underlying memory buffer (in-place scaling is fully supported).
//
// Concurrency: Safe for concurrent calls on distinct memory slices.
// Allocation: Strictly 0 allocs/op when using preallocated slices.
func MultiplyGF2P8Vector(dst, src []byte, scalar byte) {
	n := len(src)
	if n == 0 {
		return
	}
	_ = dst[n-1]
	for i := range n {
		dst[i] = MultiplyGF2P8(src[i], scalar)
	}
}
