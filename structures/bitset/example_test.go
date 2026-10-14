// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bitset_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/structures/bitset"
)

func ExampleSet() {
	b := bitset.New(10)
	b.Set(2).Set(5).Set(8)

	fmt.Printf("Bit 2 is set: %v\n", b.Test(2))
	fmt.Printf("Total set bits: %d\n", b.Count())

	// Output:
	// Bit 2 is set: true
	// Total set bits: 3
}
