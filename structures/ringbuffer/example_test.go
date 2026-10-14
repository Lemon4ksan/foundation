// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ringbuffer_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/structures/ringbuffer"
)

func ExampleBuffer() {
	rb := new(ringbuffer.Buffer[int])
	rb.Init(2)
	rb.PushBack(1)
	rb.PushBack(2)

	fmt.Printf("Popped: %d\n", rb.PopFront())
	// Output: Popped: 1
}
