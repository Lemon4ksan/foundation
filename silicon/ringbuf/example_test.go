// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ringbuf_test

import (
	"fmt"
	"sync"

	"github.com/lemon4ksan/foundation/silicon/ringbuf"
)

func ExampleSPSCRingBuffer() {
	rb := ringbuf.NewSPSCRingBuffer[uint64](2048)
	var wg sync.WaitGroup
	wg.Add(2)

	// Producer Thread
	go func() {
		defer wg.Done()
		for i := uint64(1); i <= 1000; i++ {
			val := i
			for !rb.Push(&val) {
				// Buffer full, yield
			}
		}
	}()

	// Consumer Thread
	go func() {
		defer wg.Done()
		received := uint64(0)
		for received < 1000 {
			if val := rb.Pop(); val != nil {
				received = *val
			}
		}
		fmt.Println("Consumed all items, last:", received)
	}()

	wg.Wait()

	// Output:
	// Consumed all items, last: 1000
}
