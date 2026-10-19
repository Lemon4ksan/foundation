// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package endian_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/silicon/endian"
)

func ExampleLoad32() {
	buf := make([]byte, 8)
	endian.Store32(buf, 4, 0x12345678)
	fmt.Printf("0x%08x\n", endian.Load32(buf, 4))

	// Output:
	// 0x12345678
}

func ExampleLoadBE24() {
	buf := make([]byte, 4)
	endian.StoreBE24(buf, 1, 0x00AABBCC)
	fmt.Printf("0x%06x\n", endian.LoadBE24(buf, 1))

	// Output:
	// 0xaabbcc
}
