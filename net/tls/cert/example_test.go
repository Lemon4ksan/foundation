// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cert_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/net/tls/cert"
)

func ExampleParseCompressionAlgorithm() {
	algo, err := cert.ParseCompressionAlgorithm("zstd")
	if err == nil {
		fmt.Println(algo)
	}

	// Output:
	// zstd
}
