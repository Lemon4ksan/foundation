// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/encoding/proto"
)

func ExampleScan() {
	// Encode fields into a buffer
	var buf []byte
	buf = proto.AppendVarint(buf, 1, 42)
	buf = proto.AppendString(buf, 2, "hello foundation")

	// Scan fields without heap allocations
	for field, err := range proto.Scan(buf) {
		if err != nil {
			break
		}
		switch field.Num {
		case 1:
			fmt.Printf("Field 1 (Varint): %d\n", field.Varint())
		case 2:
			fmt.Printf("Field 2 (String): %s\n", field.String())
		}
	}

	// Output:
	// Field 1 (Varint): 42
	// Field 2 (String): hello foundation
}

func ExampleAppendString() {
	buf := proto.AppendString(nil, 1, "test")
	fmt.Printf("Bytes encoded: %d\n", len(buf))

	// Output:
	// Bytes encoded: 6
}
