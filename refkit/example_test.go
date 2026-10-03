// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package refkit_test

import (
	"fmt"
	"reflect"

	"github.com/lemon4ksan/foundation/refkit"
)

type User struct {
	Age int `json:"age,omitempty" validate:"age,min=18,max=120"`
}

func ExampleGetTag() {
	field, _ := reflect.TypeOf(User{}).FieldByName("Age")

	tag := refkit.GetTag(field, "validate")
	minAge, ok := tag.GetInt("min")
	fmt.Printf("Min age required: %d (found: %v)\n", minAge, ok)

	// Output:
	// Min age required: 18 (found: true)
}

func ExampleIsNil() {
	var ch chan int
	val := reflect.ValueOf(ch)

	// Checks nil status safely without panicking on unsupported kinds
	if refkit.IsNil(val) {
		fmt.Println("Channel is nil")
	}

	// Output:
	// Channel is nil
}
