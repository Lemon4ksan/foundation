// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package list_test

import (
	"fmt"

	list "github.com/lemon4ksan/foundation/structures/linkedlist"
)

func ExampleList() {
	// Create an array-backed doubly linked list with preallocated capacity
	l := list.NewCapacity[string](8)

	l.PushBack("alpha")
	l.PushBack("beta")
	l.PushFront("root")

	// Standard Go iterator traversal
	for v := range l.Values() {
		fmt.Println(v)
	}

	fmt.Printf("Length: %d\n", l.Len())

	// Pop front
	front := l.Front()
	l.Remove(front)
	fmt.Printf("After pop front, new front: %s\n", l.Front().Value)

	// Output:
	// root
	// alpha
	// beta
	// Length: 3
	// After pop front, new front: alpha
}
