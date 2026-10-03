// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shuffle_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/codec/filter/shuffle"
)

func ExampleEncode() {
	// An array of three 4-byte (FP32 or INT32) elements.
	// Element 1: A1 A2 A3 A4
	// Element 2: B1 B2 B3 B4
	// Element 3: C1 C2 C3 C4
	src := []byte{
		0xA1, 0xA2, 0xA3, 0xA4,
		0xB1, 0xB2, 0xB3, 0xB4,
		0xC1, 0xC2, 0xC3, 0xC4,
	}

	dst := make([]byte, len(src))

	// Encode: transpose so first bytes of each element group together.
	shuffle.Encode(src, dst, shuffle.WidthFP32)
	fmt.Printf("Encoded: %X\n", dst)

	// Decode: restore original arrangement.
	shuffle.Decode(dst, src, shuffle.WidthFP32)
	fmt.Printf("Decoded: %X\n", src)

	// Output:
	// Encoded: A1B1C1A2B2C2A3B3C3A4B4C4
	// Decoded: A1A2A3A4B1B2B3B4C1C2C3C4
}
