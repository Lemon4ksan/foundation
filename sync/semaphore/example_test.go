// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package semaphore_test

import (
	"context"
	"fmt"

	"github.com/lemon4ksan/foundation/sync/semaphore"
)

func ExampleSemaphore() {
	sem := semaphore.New(3)
	ctx := context.Background()

	if err := sem.Acquire(ctx); err != nil {
		panic(err)
	}
	fmt.Println("acquired slot")

	sem.Release()
	sem.Resize(10)
	fmt.Println("resized to 10")

	// Output:
	// acquired slot
	// resized to 10
}
