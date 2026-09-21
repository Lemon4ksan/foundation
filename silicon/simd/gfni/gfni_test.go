// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gfni_test

import (
	"fmt"
	"testing"

	"github.com/lemon4ksan/foundation/silicon/simd/gfni"
	"github.com/lemon4ksan/foundation/testing/require"
)

func TestMultiplyGF2P8_Identities(t *testing.T) {
	// Zero annihilation: a * 0 = 0 * a = 0
	// Multiplicative identity: a * 1 = 1 * a = a for all a in GF(2^8)
	for i := range 256 {
		a := byte(i)
		require.Equal(t, byte(0), gfni.MultiplyGF2P8(a, 0))
		require.Equal(t, byte(0), gfni.MultiplyGF2P8(0, a))
		require.Equal(t, a, gfni.MultiplyGF2P8(a, 1))
		require.Equal(t, a, gfni.MultiplyGF2P8(1, a))
	}
}

func TestMultiplyGF2P8_KnownVectors(t *testing.T) {
	// Known AES Galois Field GF(2^8) reduction vectors with irreducible polynomial:
	// m(x) = x^8 + x^4 + x^3 + x + 1 (0x11B).
	testCases := []struct {
		a, b     byte
		expected byte
		comment  string
	}{
		// Canonical FIPS-197 Section 4.2 Rijndael example: {57} * {83} = {c1}
		{0x57, 0x83, 0xc1, "FIPS-197 4.2 canonical example"},
		// Power-of-x doublings (xtime steps) from FIPS-197 Section 4.2:
		{0x57, 0x01, 0x57, "{57} * {01}"},
		{0x57, 0x02, 0xae, "{57} * {02}"},
		{0x57, 0x04, 0x47, "{57} * {04}"},
		{0x57, 0x08, 0x8e, "{57} * {08}"},
		{0x57, 0x10, 0x07, "{57} * {10}"},
		{0x57, 0x20, 0x0e, "{57} * {20}"},
		{0x57, 0x40, 0x1c, "{57} * {40}"},
		{0x57, 0x80, 0x38, "{57} * {80}"},
		{0x57, 0x13, 0xfe, "{57} * {13} = {07} ^ {ae} ^ {57}"},
		// Modular reductions at boundary x^8 = 0x1B:
		{0x02, 0x80, 0x1b, "0x80 << 1 = 0x100 ^ 0x11b = 0x1b"},
		{0x02, 0x8f, 0x05, "0x8f << 1 = 0x11e ^ 0x11b = 0x05"},
		{0x02, 0xff, 0xe5, "0xff << 1 = 0x1fe ^ 0x11b = 0xe5"},
		// Polynomial square and products:
		{0x03, 0x03, 0x05, "(x + 1)^2 = x^2 + 1 = 0x05"},
		{0x07, 0x09, 0x3f, "(x^2 + x + 1)(x^3 + 1) = 0x3f"},
		{0x11, 0x22, 0x34, "0x11 * 0x22 = 0x34"},
		{0xff, 0xff, 0x13, "0xff * 0xff = 0x13"},
		{0xaa, 0x55, 0x59, "0xaa * 0x55 = 0x59"},
		// AES MixColumns transformations (FIPS-197 Section 5.1.3):
		{0x02, 0xd4, 0xb3, "MixColumns: {02} * {d4} = {b3}"},
		{0x03, 0xbf, 0xda, "MixColumns: {03} * {bf} = {da}"},
		{0x02, 0xdb, 0xad, "MixColumns: {02} * {db} = {ad}"},
		// Multiplicative inverses (a * a^-1 = 1) in GF(2^8):
		{0x02, 0x8d, 0x01, "Inverse: 0x02 * 0x8d = 0x01"},
		{0x03, 0xf6, 0x01, "Inverse: 0x03 * 0xf6 = 0x01"},
		{0x04, 0xcb, 0x01, "Inverse: 0x04 * 0xcb = 0x01"},
		{0xff, 0x1c, 0x01, "Inverse: 0xff * 0x1c = 0x01"},
	}

	for _, tc := range testCases {
		actual := gfni.MultiplyGF2P8(tc.a, tc.b)
		require.Equalf(t, tc.expected, actual, "MultiplyGF2P8(%#x, %#x) [%s]", tc.a, tc.b, tc.comment)
	}
}

func TestMultiplyGF2P8_Commutativity(t *testing.T) {
	// Commutativity: a * b == b * a for all 256 * 256 pairs
	for i := range 256 {
		for j := range 256 {
			a := byte(i)
			b := byte(j)
			require.Equal(t, gfni.MultiplyGF2P8(a, b), gfni.MultiplyGF2P8(b, a))
		}
	}
}

func TestMultiplyGF2P8_AssociativityAndDistributivity(t *testing.T) {
	// Associativity: (a * b) * c == a * (b * c)
	// Distributivity: a * (b ^ c) == (a * b) ^ (a * c)
	// Evaluated over a comprehensive 32-element structural test grid covering zero,
	// identity, generator x, reduction constant, bit powers, alternating patterns,
	// and inverse pairs (32^3 = 32,768 combinations).
	grid := []byte{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x07, 0x08, 0x09,
		0x0b, 0x0d, 0x0e, 0x10, 0x1b, 0x20, 0x34, 0x40,
		0x55, 0x57, 0x7f, 0x80, 0x83, 0x8d, 0x8f, 0xaa,
		0xbf, 0xc1, 0xcb, 0xd4, 0xdb, 0xe5, 0xf6, 0xff,
	}

	for _, a := range grid {
		for _, b := range grid {
			for _, c := range grid {
				// Associativity: (a * b) * c == a * (b * c)
				assocLeft := gfni.MultiplyGF2P8(gfni.MultiplyGF2P8(a, b), c)
				assocRight := gfni.MultiplyGF2P8(a, gfni.MultiplyGF2P8(b, c))
				require.Equal(t, assocLeft, assocRight)

				// Distributivity: a * (b ^ c) == (a * b) ^ (a * c)
				distLeft := gfni.MultiplyGF2P8(a, b^c)
				distRight := gfni.MultiplyGF2P8(a, b) ^ gfni.MultiplyGF2P8(a, c)
				require.Equal(t, distLeft, distRight)
			}
		}
	}
}

func TestMultiplyGF2P8_FieldInverses(t *testing.T) {
	// Every non-zero element in GF(2^8) must have a unique multiplicative inverse
	// such that a * a^-1 == 1.
	for i := 1; i < 256; i++ {
		a := byte(i)
		inverseCount := 0
		var inv byte
		for j := 1; j < 256; j++ {
			b := byte(j)
			if gfni.MultiplyGF2P8(a, b) == 1 {
				inverseCount++
				inv = b
			}
		}
		require.Equalf(t, 1, inverseCount, "element %#x must have exactly one multiplicative inverse", a)
		require.Equal(t, byte(1), gfni.MultiplyGF2P8(inv, a))
	}
}

func TestMultiplyGF2P8_FrobeniusEndomorphism(t *testing.T) {
	// In characteristic 2 fields: (a ^ b)^2 == a^2 ^ b^2 (squaring is linear).
	for i := range 256 {
		for j := range 256 {
			a := byte(i)
			b := byte(j)
			sum := a ^ b
			sumSquared := gfni.MultiplyGF2P8(sum, sum)
			squaresSum := gfni.MultiplyGF2P8(a, a) ^ gfni.MultiplyGF2P8(b, b)
			require.Equal(t, sumSquared, squaresSum)
		}
	}
}

func TestMultiplyGF2P8Vector_Lengths(t *testing.T) {
	// Slice lengths explicitly required: 0, 1, 7, 8, 15, 16, 31, 32, 63, 64, 1024 bytes.
	lengths := []int{0, 1, 7, 8, 15, 16, 31, 32, 63, 64, 1024}
	scalars := []byte{0x00, 0x01, 0x02, 0x03, 0x1b, 0x57, 0x83, 0xaa, 0xff}

	for _, length := range lengths {
		for _, scalar := range scalars {
			t.Run(fmt.Sprintf("len_%d_scalar_%#x", length, scalar), func(t *testing.T) {
				src := make([]byte, length)
				for i := range src {
					src[i] = byte((i*37 + 13) & 0xff)
				}

				dst := make([]byte, length)
				gfni.MultiplyGF2P8Vector(dst, src, scalar)

				for i := range src {
					expected := gfni.MultiplyGF2P8(src[i], scalar)
					require.Equalf(t, expected, dst[i], "index %d for length %d", i, length)
				}
			})
		}
	}
}

func TestMultiplyGF2P8Vector_InPlaceAndOversized(t *testing.T) {
	lengths := []int{1, 7, 8, 15, 16, 31, 32, 63, 64, 1024}
	scalar := byte(0x57)

	for _, length := range lengths {
		t.Run(fmt.Sprintf("in_place_len_%d", length), func(t *testing.T) {
			buf := make([]byte, length)
			expected := make([]byte, length)
			for i := range buf {
				val := byte((i*19 + 29) & 0xff)
				buf[i] = val
				expected[i] = gfni.MultiplyGF2P8(val, scalar)
			}

			// In-place: dst and src are the exact same slice
			gfni.MultiplyGF2P8Vector(buf, buf, scalar)
			require.Equal(t, expected, buf)
		})
	}

	t.Run("oversized_dst_preserves_trailing_bytes", func(t *testing.T) {
		src := []byte{0x10, 0x20, 0x30, 0x40}
		dst := make([]byte, 16)
		const canary = byte(0xee)
		for i := range dst {
			dst[i] = canary
		}

		gfni.MultiplyGF2P8Vector(dst, src, 0x02)

		for i := range src {
			expected := gfni.MultiplyGF2P8(src[i], 0x02)
			require.Equal(t, expected, dst[i])
		}
		for i := len(src); i < len(dst); i++ {
			require.Equalf(t, canary, dst[i], "trailing byte at index %d modified", i)
		}
	})
}

func TestMultiplyGF2P8Vector_BoundaryAndPanics(t *testing.T) {
	// Empty slices should safely return without panic
	t.Run("empty_slices", func(t *testing.T) {
		require.NotPanics(t, func() {
			gfni.MultiplyGF2P8Vector(nil, nil, 0x57)
		})
		require.NotPanics(t, func() {
			gfni.MultiplyGF2P8Vector([]byte{}, []byte{}, 0x57)
		})
		require.NotPanics(t, func() {
			dst := make([]byte, 8)
			gfni.MultiplyGF2P8Vector(dst, nil, 0x57)
			for _, b := range dst {
				require.Equal(t, byte(0), b)
			}
		})
		require.NotPanics(t, func() {
			dst := make([]byte, 8)
			gfni.MultiplyGF2P8Vector(dst, []byte{}, 0x57)
			for _, b := range dst {
				require.Equal(t, byte(0), b)
			}
		})
	})

	// Mismatched slice lengths (len(dst) < len(src)) must panic
	t.Run("panics_on_short_dst", func(t *testing.T) {
		testCases := []struct {
			srcLen, dstLen int
		}{
			{1, 0},
			{8, 7},       // off-by-one boundary check on 64-bit word
			{16, 15},     // off-by-one boundary check on 128-bit SIMD register
			{32, 31},     // off-by-one boundary check on 256-bit SIMD register
			{64, 63},     // off-by-one boundary check on 512-bit SIMD register
			{64, 10},     // significantly shorter
			{1024, 1023}, // off-by-one boundary check on multi-block buffer
		}

		for _, tc := range testCases {
			t.Run(fmt.Sprintf("src_%d_dst_%d", tc.srcLen, tc.dstLen), func(t *testing.T) {
				src := make([]byte, tc.srcLen)
				dst := make([]byte, tc.dstLen)
				require.Panics(t, func() {
					gfni.MultiplyGF2P8Vector(dst, src, 0x02)
				})
			})
		}

		t.Run("panic_on_nil_dst_nonempty_src", func(t *testing.T) {
			src := make([]byte, 16)
			require.Panics(t, func() {
				gfni.MultiplyGF2P8Vector(nil, src, 0x02)
			})
		})
	})
}

func TestMultiplyGF2P8_ZeroAllocations(t *testing.T) {
	allocs := testing.AllocsPerRun(100, func() {
		_ = gfni.MultiplyGF2P8(0x57, 0x83)
	})
	require.Equal(t, float64(0), allocs)

	src := make([]byte, 1024)
	dst := make([]byte, 1024)
	vectorAllocs := testing.AllocsPerRun(100, func() {
		gfni.MultiplyGF2P8Vector(dst, src, 0x57)
	})
	require.Equal(t, float64(0), vectorAllocs)
}

func BenchmarkMultiplyGF2P8(b *testing.B) {
	b.ReportAllocs()
	x := byte(0x57)
	y := byte(0x83)
	for b.Loop() {
		x = gfni.MultiplyGF2P8(x, y)
	}
}

func BenchmarkMultiplyGF2P8Vector(b *testing.B) {
	sizes := []int{16, 64, 256, 1024, 65536}
	scalar := byte(0x07)

	for _, size := range sizes {
		b.Run(fmt.Sprintf("%dB", size), func(b *testing.B) {
			src := make([]byte, size)
			dst := make([]byte, size)
			for i := range src {
				src[i] = byte(i & 0xff)
			}

			b.ReportAllocs()
			b.SetBytes(int64(size))
			b.ResetTimer()

			for b.Loop() {
				gfni.MultiplyGF2P8Vector(dst, src, scalar)
			}
		})
	}
}

func BenchmarkMultiplyGF2P8Vector_64B(b *testing.B) {
	src := make([]byte, 64)
	dst := make([]byte, 64)
	for i := range src {
		src[i] = byte(i)
	}
	scalar := byte(0x03)

	b.ReportAllocs()
	b.SetBytes(64)
	b.ResetTimer()

	for b.Loop() {
		gfni.MultiplyGF2P8Vector(dst, src, scalar)
	}
}

func BenchmarkMultiplyGF2P8Vector_1KB(b *testing.B) {
	src := make([]byte, 1024)
	dst := make([]byte, 1024)
	for i := range src {
		src[i] = byte(i)
	}
	scalar := byte(0x07)

	b.ReportAllocs()
	b.SetBytes(1024)
	b.ResetTimer()

	for b.Loop() {
		gfni.MultiplyGF2P8Vector(dst, src, scalar)
	}
}
