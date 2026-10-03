// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bytesconv_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/silicon/bytesconv"
)

func ExampleScanTokensBytes() {
	header := []byte("application/json; charset=utf-8; boundary=something")

	// Zero-allocation parsing
	for token := range bytesconv.ScanTokensBytes(header, ';') {
		// Zero-copy conversion
		fmt.Println("Header token:", bytesconv.B2S(token))
	}

	// Output:
	// Header token: application/json
	// Header token: charset=utf-8
	// Header token: boundary=something
}

func ExampleB2S() {
	b := []byte("hello")
	s := bytesconv.B2S(b)
	fmt.Println(s)

	// Output:
	// hello
}
