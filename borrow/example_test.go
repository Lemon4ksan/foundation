// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package borrow_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/borrow"
)

type Config struct {
	Workers int
	Name    string
}

func ExampleBox() {
	// Create an exclusively owned Box with linear ownership semantics
	box := borrow.NewBox(Config{Workers: 4, Name: "worker-pool"})

	// Immutable shared borrow
	ref := box.Borrow()
	fmt.Printf("Config name: %s, workers: %d\n", ref.Get().Name, ref.Get().Workers)

	// Exclusive mutable borrow
	mut := box.BorrowMut()
	mut.Get().Workers = 8

	fmt.Printf("Updated workers: %d\n", box.Get().Workers)

	// Release box (recycles memory and invalidates all active borrows)
	box.Release()
	fmt.Printf("Box is valid after release: %v\n", box.IsValid())

	// Output:
	// Config name: worker-pool, workers: 4
	// Updated workers: 8
	// Box is valid after release: false
}
