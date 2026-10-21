// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package memkit_test

import (
	"fmt"
	"unsafe"

	"github.com/lemon4ksan/foundation/silicon/memkit"
)

func ExampleNoCopy() {
	type RingState struct {
		_    memkit.NoCopy
		_    memkit.NoCompare
		_    memkit.Align64
		head uint64
		_    memkit.CacheLinePad
		tail uint64
	}

	var s RingState
	fmt.Println("size:", unsafe.Sizeof(s), "align:", unsafe.Alignof(s))
	// Output: size: 80 align: 8
}

func ExampleIsNilInterface() {
	var p *int
	var v any = p

	fmt.Println("v == nil:", v == nil)
	fmt.Println("IsNilInterface(v):", memkit.IsNilInterface(v))
	// Output:
	// v == nil: false
	// IsNilInterface(v): true
}
