// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package refkit_test

import (
	"fmt"
	"io"
	"reflect"

	"github.com/lemon4ksan/foundation/refkit"
)

type User struct {
	Age int `json:"age,omitempty" validate:"age,min=18,max=120"`
}

type Config struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type Endpoint struct {
	Path    string `route:"/api/v1"`
	Timeout int    `route:"timeout"`
}

type StringWrapper struct {
	text string
}

func (s StringWrapper) String() string {
	return s.text
}

func ExampleVisibleFields() {
	t := reflect.TypeFor[Config]()
	for field := range refkit.VisibleFields(t) {
		fmt.Printf("%s: %s\n", field.Name, field.Type.Name())
	}

	// Output:
	// Host: string
	// Port: int
}

func ExampleAsInt64() {
	val := reflect.ValueOf(int32(42))
	num, ok := refkit.AsInt64(val)
	fmt.Printf("Value: %d (ok: %t)\n", num, ok)

	strVal := reflect.ValueOf("not a number")
	_, okStr := refkit.AsInt64(strVal)
	fmt.Printf("Mismatched: %t\n", okStr)

	// Output:
	// Value: 42 (ok: true)
	// Mismatched: false
}

func ExampleSchemaCache() {
	cache := refkit.NewSchemaCache()
	t := reflect.TypeFor[Endpoint]()

	schema, err := cache.Get(t)
	if err != nil {
		return
	}

	fmt.Printf("Type: %s, Fields: %d\n", schema.Type.Name(), len(schema.Fields))
	for _, f := range schema.Fields {
		fmt.Printf("Field: %s (tag: %s)\n", f.Field.Name, f.Tag.Value("route"))
	}

	// Output:
	// Type: Endpoint, Fields: 2
	// Field: Path (tag: /api/v1)
	// Field: Timeout (tag: timeout)
}

func ExampleImplements() {
	t := reflect.TypeFor[StringWrapper]()
	isStringer := refkit.Implements[fmt.Stringer](t)
	isReader := refkit.Implements[io.Reader](t)

	fmt.Printf("Implements fmt.Stringer: %t\n", isStringer)
	fmt.Printf("Implements io.Reader: %t\n", isReader)

	// Output:
	// Implements fmt.Stringer: true
	// Implements io.Reader: false
}

func ExampleTagOf() {
	field, _ := reflect.TypeOf(User{}).FieldByName("Age")

	tag := refkit.TagOf(field, "validate")
	minAge, ok := tag.Int("min")
	fmt.Printf("Min age required: %d (found: %v)\n", minAge, ok)

	// Output:
	// Min age required: 18 (found: true)
}

func ExampleIsNil() {
	var ch chan int
	val := reflect.ValueOf(ch)

	if refkit.IsNil(val) {
		fmt.Println("Channel is nil")
	}

	// Output:
	// Channel is nil
}
