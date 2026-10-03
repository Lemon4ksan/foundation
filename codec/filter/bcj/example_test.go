// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bcj_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/codec/filter/bcj"
)

func ExampleFilter() {
	// A simple x86 relative CALL instruction sequence.
	// E8 is the CALL opcode, followed by a 32-bit relative offset.
	data := []byte{0xE8, 0x05, 0x00, 0x00, 0x00, 0xE8, 0x10, 0x00, 0x00, 0x00}

	// Convert relative offsets to absolute addresses (encode)
	// using an instruction pointer base of 0x1000.
	bcj.Filter(bcj.X86, data, 0x1000, true)
	fmt.Printf("Encoded: %X\n", data)

	// Convert absolute addresses back to relative offsets (decode)
	bcj.Filter(bcj.X86, data, 0x1000, false)
	fmt.Printf("Decoded: %X\n", data)

	// Output:
	// Encoded: E80A100000E81A100000
	// Decoded: E805000000E810000000
}
