// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package offheap_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/silicon/offheap"
)

func ExampleSlabAllocator() {
	slab, err := offheap.NewSlabAllocator[[1500]byte](1024)
	if err != nil {
		panic(err)
	}
	defer slab.Release()

	packetBuf := slab.Alloc()

	copy(packetBuf[:], []byte("raw ethernet packet payload"))
	fmt.Printf("Off-heap packet stored: %s\n", packetBuf[:27])

	// Output:
	// Off-heap packet stored: raw ethernet packet payload
}
