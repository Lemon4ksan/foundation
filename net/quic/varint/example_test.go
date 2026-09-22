// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package varint_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/net/quic/varint"
)

func ExampleEncodeVarintSlice() {
	buf := make([]byte, 16)

	// Encode values into byte buffer with 0 heap allocations
	n1 := varint.EncodeVarintSlice(25, buf)
	n2 := varint.EncodeVarintSlice(15000, buf[n1:])

	fmt.Printf("Encoded 2 varints in %d bytes\n", n1+n2)

	// Decode using push iterator
	for val := range varint.Values(buf[:n1+n2]) {
		fmt.Println("Decoded:", val)
	}

	// Output:
	// Encoded 2 varints in 3 bytes
	// Decoded: 25
	// Decoded: 15000
}
