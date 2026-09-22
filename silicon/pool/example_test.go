// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pool_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/silicon/pool"
)

func ExamplePerPStorage() {
	// Create a per-CPU sharded memory pool with CPU cache-line padding
	// to prevent false sharing and lock contention across cores.
	storage := pool.NewPerPStorage(func() []byte {
		return make([]byte, 256)
	})

	// Retrieve a buffer local to the calling processor
	buf := storage.Get()
	buf[0] = 42

	fmt.Printf("Buffer length: %d, first byte: %d\n", len(buf), buf[0])

	// Output:
	// Buffer length: 256, first byte: 42
}
