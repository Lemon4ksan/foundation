// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bufkit_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/bufkit"
)

func ExampleAlignedBytes() {
	// Allocate 4KB buffer aligned to 64-byte CPU cache line boundary
	slice := bufkit.AlignedBytes(4096, 64)

	// Verify alignment
	fmt.Printf("Aligned to 64 bytes: %v\n", bufkit.IsAligned(slice, 64))

	// Output:
	// Aligned to 64 bytes: true
}

func ExampleChain() {
	chain := bufkit.NewChain()
	defer chain.Release()

	chain.WriteString("HTTP/1.1 200 OK\r\n")
	chain.WriteString("Content-Length: 1024\r\n\r\n")

	var dest [512]byte
	n, _ := chain.Read(dest[:])
	fmt.Printf("Read %d bytes\n", n)

	// Output:
	// Read 41 bytes
}

func ExampleRing() {
	ring := bufkit.NewRing[byte](64 * 1024)

	ring.Push('a')
	ring.Push('b')
	ring.Push('c')

	val, ok := ring.Pop()
	fmt.Printf("Popped %c (ok: %v)\n", val, ok)

	// Output:
	// Popped a (ok: true)
}
