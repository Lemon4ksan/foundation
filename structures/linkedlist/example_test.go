// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package list_test

import (
	"fmt"

	list "github.com/lemon4ksan/foundation/structures/linkedlist"
)

func ExampleList() {
	l := list.New[string]()
	l.PushBack("hello")
	l.PushFront("world")

	for v := range l.Values() {
		fmt.Println(v)
	}
	// Output:
	// world
	// hello
}
