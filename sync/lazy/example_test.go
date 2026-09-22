// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lazy_test

import (
	"fmt"
	"sync"

	"github.com/lemon4ksan/foundation/sync/lazy"
)

func ExampleLazy() {
	initCount := 0

	l := lazy.New(func() (string, error) {
		initCount++
		return "expensive-resource", nil
	})

	var wg sync.WaitGroup
	// 5 concurrent readers
	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			val, err := l.Get()
			if err != nil || val != "expensive-resource" {
				panic("unexpected result")
			}
		}()
	}
	wg.Wait()

	val, _ := l.Get()
	fmt.Printf("Value: %s, Invocations: %d\n", val, initCount)

	// Reset cached value
	l.Reset()
	valAfterReset, _ := l.Get()
	fmt.Printf("After reset value: %s, Invocations: %d\n", valAfterReset, initCount)

	// Output:
	// Value: expensive-resource, Invocations: 1
	// After reset value: expensive-resource, Invocations: 2
}
