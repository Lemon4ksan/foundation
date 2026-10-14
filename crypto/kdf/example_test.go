// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kdf_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/crypto/kdf"
)

func ExampleDeriveSubkey() {
	masterKey := make([]byte, 32)
	// Example static key for output predictability
	for i := range masterKey {
		masterKey[i] = byte(i)
	}

	subkey := kdf.DeriveSubkey(masterKey, "my-app-domain", 16)
	fmt.Printf("Subkey length: %d\n", len(subkey))
	// Output: Subkey length: 16
}
