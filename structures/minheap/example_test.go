// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package minheap_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/structures/minheap"
)

func ExampleHeap() {
	h := new(minheap.Heap[int, string])
	h.Push(10, "Task C")
	h.Push(1, "Task A")
	h.Push(5, "Task B")

	k, v := h.Pop()
	fmt.Printf("Highest priority: %d - %s\n", k, v)
	// Output: Highest priority: 1 - Task A
}
