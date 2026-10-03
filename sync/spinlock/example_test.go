// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package spinlock_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/sync/spinlock"
)

func ExampleSpinLock() {
	var mu spinlock.SpinLock

	mu.Lock()
	fmt.Println("locked")
	mu.Unlock()

	if mu.TryLock() {
		fmt.Println("try-lock succeeded")
		mu.Unlock()
	}
	// Output:
	// locked
	// try-lock succeeded
}
