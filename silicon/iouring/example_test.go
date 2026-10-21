// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package iouring_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/silicon/iouring"
)

func ExampleSQE_PrepRead() {
	buf := make([]byte, 4096)
	var sqe iouring.SQE
	sqe.PrepRead(3, buf, 0, 99)

	fmt.Println(sqe.Opcode == iouring.OpRead, sqe.Fd, sqe.Len, sqe.UserData)
	// Output:
	// true 3 4096 99
}
