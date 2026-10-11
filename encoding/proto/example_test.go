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

func ExampleAppendPackedVarints() {
	vals := []uint64{10, 20, 30, 40}
	buf := proto.AppendPackedVarints(nil, 1, vals)

	fmt.Printf("Encoded %d packed varints in %d bytes\n", len(vals), len(buf))

	// Output:
	// Encoded 4 packed varints in 6 bytes
}

func ExampleScanPackedVarints() {
	// Encode packed varints into wire format
	vals := []uint64{100, 200, 300}
	buf := proto.AppendPackedVarints(nil, 1, vals)

	// Scan through message fields and iterate packed varints
	for field, err := range proto.Scan(buf) {
		if err != nil {
			break
		}
		if field.Num == 1 {
			for v, vErr := range field.PackedVarints() {
				if vErr != nil {
					break
				}
				fmt.Println(v)
			}
		}
	}

	// Output:
	// 100
	// 200
	// 300
}

func ExampleFindField() {
	var buf []byte
	buf = proto.AppendVarint(buf, 1, 100)
	buf = proto.AppendString(buf, 2, "target-payload")
	buf = proto.AppendFixed32(buf, 3, 999)

	field, found, err := proto.FindField(buf, 2)
	if err != nil || !found {
		return
	}
	fmt.Printf("Field %d: %s\n", field.Num, field.String())

	// Output:
	// Field 2: target-payload
}

func ExampleScanMapEntry() {
	// Map entry submessage payload: key (field 1) = "status", value (field 2) = 200
	var entryData []byte
	entryData = proto.AppendString(entryData, 1, "status")
	entryData = proto.AppendVarint(entryData, 2, 200)

	key, val, err := proto.ScanMapEntry(entryData)
	if err != nil {
		return
	}
	fmt.Printf("Key: %s, Value: %d\n", key.String(), val.Varint())

	// Output:
	// Key: status, Value: 200
}

func ExampleWalkMessage() {
	var inner []byte
	inner = proto.AppendVarint(inner, 1, 100)

	var outer []byte
	outer = proto.AppendMessage(outer, 2, inner)
	outer = proto.AppendVarint(outer, 3, 200)

	_ = proto.WalkMessage(outer, func(depth int, f proto.Field) bool {
		return f.Num == 2 // Descend into field 2
	}, func(depth int, f proto.Field) error {
		if f.Type == proto.WireVarint {
			fmt.Printf("depth=%d field=%d val=%d\n", depth, f.Num, f.Varint())
		}
		return nil
	})

	// Output:
	// depth=1 field=1 val=100
	// depth=0 field=3 val=200
}

func ExampleScan_validation() {
	// Malformed payload: length prefix specifies 100 bytes, but buffer only has 2 bytes
	var malformed []byte
	malformed = proto.AppendTag(malformed, 1, proto.WireBytes)
	malformed = append(malformed, 100) // length = 100
	malformed = append(malformed, 0xDE, 0xAD)

	for _, err := range proto.Scan(malformed) {
		if err != nil {
			fmt.Printf("Validation error: %v\n", err)
			return
		}
	}

	// Output:
	// Validation error: proto: unexpected end of buffer
}

func ExampleSkipField() {
	// A wire payload containing an unknown field to skip
	var buf []byte
	buf = proto.AppendString(buf, 999, "unknown-extension-data")

	// Read tag to identify wire type
	_, wt := proto.SplitTag(proto.Tag(999, proto.WireBytes))
	tagBytes := 2 // encoded size of tag 999 (WireBytes)

	n, err := proto.SkipField(buf[tagBytes:], wt)
	if err == nil {
		fmt.Printf("Skipped bytes: %d\n", n)
	}

	// Output:
	// Skipped bytes: 23
}
