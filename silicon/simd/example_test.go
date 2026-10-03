// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package simd_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/silicon/simd"
)

func ExampleXORMask32() {
	payload := []byte("hello world data!")
	maskKey := uint32(0xAABBCCDD)

	// In-place masking
	simd.XORMask32(payload, maskKey)

	// Unmasking by re-applying the same key
	simd.XORMask32(payload, maskKey)

	fmt.Println("Decoded:", string(payload))
	// Output:
	// Decoded: hello world data!
}
