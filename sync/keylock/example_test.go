// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package keylock_test

import (
	"fmt"
	"sync"
	"time"

	"github.com/lemon4ksan/foundation/sync/keylock"
)

func ExampleMutex() {
	kl := keylock.New[string]()
	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()
		kl.Lock("user-alice")
		defer kl.Unlock("user-alice")
		fmt.Println("Locked alice")
		time.Sleep(50 * time.Millisecond)
	}()

	go func() {
		defer wg.Done()
		kl.Lock("user-bob")
		defer kl.Unlock("user-bob")
		fmt.Println("Locked bob")
		time.Sleep(50 * time.Millisecond)
	}()

	wg.Wait()
	// Unordered output:
	// Locked alice
	// Locked bob
}
