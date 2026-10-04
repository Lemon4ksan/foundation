// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wire_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/net/dns/wire"
)

func ExamplePackDNSQuery() {
	query, err := wire.PackDNSQuery(0x1234, "example.com", wire.TypeA)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Query length: %d\n", len(query))

	// Output:
	// Query length: 128
}
