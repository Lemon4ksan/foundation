// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package base64_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/encoding/base64"
)

func ExampleBase64EncodeURL() {
	src := []byte("hello world")
	dst := make([]byte, base64.Base64URLEncodedLen(len(src)))
	base64.Base64EncodeURL(src, dst)

	fmt.Printf("%s\n", dst)
	// Output: aGVsbG8gd29ybGQ
}
