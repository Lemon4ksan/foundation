// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package entropy_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/codec/entropy"
)

func ExampleShannonEntropy() {
	payload := []byte("plain text sample repeating patterns")

	ent := entropy.ShannonEntropy(payload)
	incompressible := entropy.IsIncompressibleSample(payload)

	fmt.Printf("Entropy < 8.0: %v\n", ent < 8.0)
	fmt.Printf("Incompressible: %v\n", incompressible)
	// Output:
	// Entropy < 8.0: true
	// Incompressible: false
}
