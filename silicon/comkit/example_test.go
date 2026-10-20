// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package comkit_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/silicon/comkit"
)

func ExampleParseGUID() {
	iid, err := comkit.ParseGUID("{AF86E2E0-B12D-4C6A-9C5A-D7AA65101E90}")
	if err != nil {
		panic(err)
	}
	fmt.Println(iid.Equal(comkit.IIDIInspectable))
	fmt.Println(iid.String())
	// Output:
	// true
	// af86e2e0-b12d-4c6a-9c5a-d7aa65101e90
}

func ExampleVARIANT() {
	v := comkit.NewVariantI4(1024)
	fmt.Println(v.VT == comkit.VTI4, v.Int32())
	// Output:
	// true 1024
}
