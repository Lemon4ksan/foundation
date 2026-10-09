// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cpukit_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/silicon/cpukit"
)

func ExampleHasGFNI() {
	// Query hardware Galois Field New Instructions (GFNI) capability at runtime.
	_ = cpukit.HasGFNI()
	fmt.Println("GFNI probe completed")

	// Output:
	// GFNI probe completed
}
