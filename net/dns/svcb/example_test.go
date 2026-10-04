// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svcb_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/net/dns/svcb"
)

func ExampleBuildHTTPSQueryName() {
	qname := svcb.BuildHTTPSQueryName("example.com", 443)
	fmt.Println(qname)

	// Output:
	// example.com.
}
