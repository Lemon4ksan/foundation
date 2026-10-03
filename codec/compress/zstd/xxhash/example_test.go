// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xxhash_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/codec/compress/zstd/xxhash"
)

func ExampleSum64() {
	hash := xxhash.Sum64([]byte("hello world"))
	fmt.Printf("%016x\n", hash)
	// Output: 45ab6734b21e6968
}

func ExampleDigest() {
	d := xxhash.New()
	d.Write([]byte("hello "))
	d.WriteString("world")
	fmt.Printf("%016x\n", d.Sum64())
	// Output: 45ab6734b21e6968
}
