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

func ExampleChain_Peek() {
	chain := bufkit.NewChain()
	defer chain.Release()

	chain.WriteString("PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n")
	peeked, _ := chain.Peek(3)
	fmt.Printf("Peeked prefix: %s (Len remains %d)\n", string(peeked), chain.Len())

	// Output:
	// Peeked prefix: PRI (Len remains 24)
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

func ExampleRing_PushBatch() {
	ring := bufkit.NewRing[int](16)

	items := []int{10, 20, 30, 40}
	pushed := ring.PushBatch(items)

	dest := make([]int, 2)
	popped := ring.PopBatch(dest)

	fmt.Printf("Pushed: %d, Popped: %d, Values: %v\n", pushed, popped, dest)

	// Output:
	// Pushed: 4, Popped: 2, Values: [10 20]
}
