// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ip_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/net/ip"
)

func ExampleSourceRotator() {
	rotator, err := ip.NewSourceRotator([]string{"10.0.0.1", "10.0.0.2"})
	if err != nil {
		return
	}

	// NextOK returns the next IP and whether the pool was non-empty.
	if addr, ok := rotator.NextOK(); ok {
		fmt.Println(addr.String())
	}
	if addr, ok := rotator.NextOK(); ok {
		fmt.Println(addr.String())
	}

	// Output:
	// 10.0.0.1
	// 10.0.0.2
}
