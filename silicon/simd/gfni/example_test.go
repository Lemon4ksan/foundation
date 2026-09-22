// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gfni_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/silicon/simd/gfni"
)

func ExampleMultiplyGF2P8() {
	// Galois Field GF(2^8) multiplication modulo irreducible polynomial x^8+x^4+x^3+x+1 (0x11B)
	res := gfni.MultiplyGF2P8(0x57, 0x83)
	fmt.Printf("0x57 * 0x83 in GF(2^8) = 0x%02X\n", res)

	// Vector multiplication in-place
	data := []byte{0x01, 0x02, 0x04}
	gfni.MultiplyGF2P8Vector(data, data, 0x02)
	fmt.Printf("Vector * 2: [% 02X]\n", data)

	// Output:
	// 0x57 * 0x83 in GF(2^8) = 0xC1
	// Vector * 2: [02 04 08]
}
