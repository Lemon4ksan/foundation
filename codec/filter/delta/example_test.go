// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package delta_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/codec/filter/delta"
)

func ExampleFilter() {
	// Sample data: a sequence of 16-bit PCM values (distance = 2).
	// Values: 1000, 1005, 1010
	data := []byte{
		0xE8, 0x03, // 1000 in little-endian
		0xED, 0x03, // 1005
		0xF2, 0x03, // 1010
	}

	filter, _ := delta.NewFilter(2)

	// Encode in-place
	filter.Encode(data)
	fmt.Printf("Encoded: %X\n", data)

	// Reset state before decoding
	filter.Reset()

	// Decode in-place
	filter.Decode(data)
	fmt.Printf("Decoded: %X\n", data)

	// Output:
	// Encoded: E80305000500
	// Decoded: E803ED03F203
}
