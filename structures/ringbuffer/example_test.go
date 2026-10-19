// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ringbuffer_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/structures/ringbuffer"
)

func ExampleBuffer() {
	rb := ringbuffer.NewWithCapacity[int](2)
	rb.PushBack(1)
	rb.PushBack(2)

	v, _ := rb.PopFront()
	fmt.Printf("Popped: %d\n", v)
	// Output: Popped: 1
}
