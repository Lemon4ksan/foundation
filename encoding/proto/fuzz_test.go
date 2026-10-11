// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto_test

import (
	"encoding/binary"
	"errors"
	"math"
	"testing"

	"github.com/lemon4ksan/foundation/encoding/proto"
)

// FuzzProtoScan stress tests proto.Scan with malformed wire buffers,
// boundary conditions, overflow varints, truncated prefixes, and invalid tags/wire types.
func FuzzProtoScan(f *testing.F) {
	// Seed 1: Empty buffer
	f.Add([]byte{})

	// Seed 2: Standard valid multi-field protobuf message
	var seedValid []byte
	seedValid = proto.AppendVarint(seedValid, 1, 42)
	seedValid = proto.AppendSint32(seedValid, 2, -100)
	seedValid = proto.AppendFixed32(seedValid, 3, 0x12345678)
	seedValid = proto.AppendFixed64(seedValid, 4, 0x0102030405060708)
	seedValid = proto.AppendString(seedValid, 5, "fuzz-testing-proto")
	seedValid = proto.AppendBytes(seedValid, 6, []byte{0xDE, 0xAD, 0xBE, 0xEF})
	f.Add(seedValid)

	// Seed 3: Varint overflow (>10 bytes)
	seedOverflow := make([]byte, 12)
	for i := range seedOverflow {
		seedOverflow[i] = 0x80
	}
	f.Add(seedOverflow)

	// Seed 4: 10 bytes with continuation bit set
	seed10 := make([]byte, 10)
	for i := range seed10 {
		seed10[i] = 0x80
	}
	f.Add(seed10)

	// Seed 5: Zero tag
	f.Add([]byte{0x00})

	// Seed 6: Tag > MaxUint32
	var seedBigTag []byte
	seedBigTag = binary.AppendUvarint(seedBigTag, (uint64(1)<<35)+8)
	seedBigTag = binary.AppendUvarint(seedBigTag, 100)
	f.Add(seedBigTag)

	// Seed 7: Truncated length prefix (MaxUint64 length)
	var seedMaxLen []byte
	seedMaxLen = append(seedMaxLen, 0x0a) // field 1, wire 2
	seedMaxLen = binary.AppendUvarint(seedMaxLen, math.MaxUint64)
	f.Add(seedMaxLen)

	// Seed 8: Invalid wire types (3, 4, 6, 7)
	f.Add(proto.AppendTag(nil, 1, proto.WireStartGroup))
	f.Add(proto.AppendTag(nil, 1, proto.WireEndGroup))
	f.Add(proto.AppendTag(nil, 1, proto.WireType(6)))
	f.Add(proto.AppendTag(nil, 1, proto.WireType(7)))

	// Seed 9: Packed varints payload
	var seedPacked []byte
	seedPacked = proto.AppendPackedVarints(seedPacked, 1, []uint64{10, 20, 300, 4000, 50000})
	f.Add(seedPacked)

	f.Fuzz(func(t *testing.T, data []byte) {
		fieldCount := 0
		var lastErr error

		for field, err := range proto.Scan(data) {
			if err != nil {
				lastErr = err
				// Verify error is one of the recognized protobuf wire errors
				if !errors.Is(err, proto.ErrTruncated) &&
					!errors.Is(err, proto.ErrVarintOverflow) &&
					!errors.Is(err, proto.ErrInvalidTag) &&
					!errors.Is(err, proto.ErrInvalidWireType) {
					t.Fatalf("unexpected scan error: %v", err)
				}
				// Once an error is encountered, iteration must terminate immediately
				break
			}

			fieldCount++

			// Invariant: Field number must be strictly positive and <= MaxFieldNumber
			if field.Num == 0 || field.Num > proto.MaxFieldNumber {
				t.Fatalf("field number %d violates proto bounds [1, %d]", field.Num, proto.MaxFieldNumber)
			}

			// Invariant: Wire type must be valid
			switch field.Type {
			case proto.WireVarint:
				_ = field.Varint()
				_ = field.Int32()
				_ = field.Int64()
				_ = field.Uint32()
				_ = field.Uint64()
				_ = field.Sint32()
				_ = field.Sint64()
				_ = field.Bool()

			case proto.WireFixed64:
				_ = field.Fixed64()
				_ = field.Float64()

			case proto.WireBytes:
				raw := field.Bytes()
				if len(raw) > len(data) {
					t.Fatalf("WireBytes payload length %d exceeds total buffer %d", len(raw), len(data))
				}
				_ = field.String()
				msg := field.Message()

				// If the byte field looks like an embedded message, scan one level deeper safely
				if len(msg) > 0 && len(msg) < 512 {
					for subField, subErr := range proto.Scan(msg) {
						if subErr != nil {
							break
						}
						if subField.Num == 0 || subField.Num > proto.MaxFieldNumber {
							t.Fatalf("subfield number %d out of bounds", subField.Num)
						}
					}
				}

			case proto.WireFixed32:
				_ = field.Fixed32()
				_ = field.Float32()

			default:
				t.Fatalf("scanner yielded invalid wire type: %d", field.Type)
			}
		}

		_ = lastErr
	})
}

// FuzzProtoSkip stress tests proto.SkipField for all wire types, truncated inputs,
// overflow lengths, and ensures it never infinite loops or returns non-positive progress.
func FuzzProtoSkip(f *testing.F) {
	// Seed with valid and truncated inputs across wire types
	f.Add([]byte{0x01}, uint8(proto.WireVarint))
	f.Add([]byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}, uint8(proto.WireVarint))
	f.Add([]byte{0x01, 0x02, 0x03, 0x04}, uint8(proto.WireFixed32))
	f.Add([]byte{0x01, 0x02}, uint8(proto.WireFixed32))
	f.Add([]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}, uint8(proto.WireFixed64))
	f.Add([]byte{0x01, 0x02, 0x03, 0x04}, uint8(proto.WireFixed64))

	var maxLenBytes []byte
	maxLenBytes = binary.AppendUvarint(maxLenBytes, math.MaxUint64)
	f.Add(maxLenBytes, uint8(proto.WireBytes))

	var validBytes []byte
	validBytes = binary.AppendUvarint(validBytes, 5)
	validBytes = append(validBytes, "hello"...)
	f.Add(validBytes, uint8(proto.WireBytes))

	// Invalid wire types
	f.Add([]byte{0x01}, uint8(3))
	f.Add([]byte{0x01}, uint8(4))
	f.Add([]byte{0x01}, uint8(6))
	f.Add([]byte{0x01}, uint8(7))
	f.Add([]byte{0x01}, uint8(255))

	f.Fuzz(func(t *testing.T, data []byte, wt uint8) {
		wireType := proto.WireType(wt)
		n, err := proto.SkipField(data, wireType)

		if err == nil {
			// Invariant 1: If err == nil, wireType must be a recognized valid wire type
			if wireType != proto.WireVarint &&
				wireType != proto.WireFixed64 &&
				wireType != proto.WireBytes &&
				wireType != proto.WireFixed32 {
				t.Fatalf("SkipField succeeded with invalid wire type: %d", wireType)
			}

			// Invariant 2: Progress must be strictly positive (prevent infinite loops)
			if n <= 0 {
				t.Fatalf("SkipField returned non-positive byte count %d without error (infinite loop vector)", n)
			}

			// Invariant 3: Consumed bytes must not exceed input length
			if n > len(data) {
				t.Fatalf("SkipField consumed %d bytes from %d byte buffer", n, len(data))
			}
		} else {
			// Invariant 4: On error, consumed count must be 0
			if n != 0 {
				t.Fatalf("SkipField returned non-zero n=%d on error: %v", n, err)
			}

			// Invariant 5: Error must be recognized
			if !errors.Is(err, proto.ErrTruncated) &&
				!errors.Is(err, proto.ErrVarintOverflow) &&
				!errors.Is(err, proto.ErrInvalidWireType) {
				t.Fatalf("unexpected SkipField error: %v", err)
			}
		}
	})
}

// FuzzProtoPacked stress tests packed repeated field scanning and encoding.
func FuzzProtoPacked(f *testing.F) {
	// Seed 1: Empty slice
	f.Add([]byte{})

	// Seed 2: Valid packed varints
	var seedVarints []byte
	seedVarints = binary.AppendUvarint(seedVarints, 0)
	seedVarints = binary.AppendUvarint(seedVarints, 1)
	seedVarints = binary.AppendUvarint(seedVarints, 127)
	seedVarints = binary.AppendUvarint(seedVarints, 128)
	seedVarints = binary.AppendUvarint(seedVarints, math.MaxUint32)
	seedVarints = binary.AppendUvarint(seedVarints, math.MaxUint64)
	f.Add(seedVarints)

	// Seed 3: Packed fixed32
	var seedFixed32 []byte
	for i := uint32(0); i < 5; i++ {
		seedFixed32 = binary.LittleEndian.AppendUint32(seedFixed32, i*100)
	}
	f.Add(seedFixed32)

	// Seed 4: Packed fixed64
	var seedFixed64 []byte
	for i := uint64(0); i < 4; i++ {
		seedFixed64 = binary.LittleEndian.AppendUint64(seedFixed64, i*1000)
	}
	f.Add(seedFixed64)

	// Seed 5: Overflow varint in packed stream
	seedOverflow := make([]byte, 15)
	for i := range seedOverflow {
		seedOverflow[i] = 0x80
	}
	f.Add(seedOverflow)

	f.Fuzz(func(t *testing.T, data []byte) {
		// 1. Stress ScanPackedVarints
		var varintVals []uint64
		hasVarintErr := false
		for val, err := range proto.ScanPackedVarints(data) {
			if err != nil {
				hasVarintErr = true
				if !errors.Is(err, proto.ErrTruncated) && !errors.Is(err, proto.ErrVarintOverflow) {
					t.Fatalf("unexpected packed varint error: %v", err)
				}
				break
			}
			varintVals = append(varintVals, val)
		}

		// If no error, roundtrip through AppendPackedVarints and verify identical re-scan
		if !hasVarintErr && len(varintVals) > 0 && len(varintVals) <= 64 {
			reencoded := proto.AppendPackedVarints(nil, 1, varintVals)
			for field, err := range proto.Scan(reencoded) {
				if err != nil {
					t.Fatalf("unexpected error scanning re-encoded packed field: %v", err)
				}
				idx := 0
				for v, vErr := range field.PackedVarints() {
					if vErr != nil {
						t.Fatalf("unexpected error unpacking re-encoded varints: %v", vErr)
					}
					if idx >= len(varintVals) || v != varintVals[idx] {
						t.Fatalf("mismatch at index %d: expected %d, got %d", idx, varintVals[idx], v)
					}
					idx++
				}
				if idx != len(varintVals) {
					t.Fatalf("expected %d elements, unpacked %d", len(varintVals), idx)
				}
			}
		}

		// 2. Stress ScanPackedFixed32
		fixed32Count := 0
		for _, err := range proto.ScanPackedFixed32(data) {
			if err != nil {
				if !errors.Is(err, proto.ErrTruncated) {
					t.Fatalf("unexpected packed fixed32 error: %v", err)
				}
				break
			}
			fixed32Count++
		}
		if len(data)%4 == 0 && fixed32Count != len(data)/4 {
			t.Fatalf("expected %d fixed32 items, got %d", len(data)/4, fixed32Count)
		}

		// 3. Stress ScanPackedFixed64
		fixed64Count := 0
		for _, err := range proto.ScanPackedFixed64(data) {
			if err != nil {
				if !errors.Is(err, proto.ErrTruncated) {
					t.Fatalf("unexpected packed fixed64 error: %v", err)
				}
				break
			}
			fixed64Count++
		}
		if len(data)%8 == 0 && fixed64Count != len(data)/8 {
			t.Fatalf("expected %d fixed64 items, got %d", len(data)/8, fixed64Count)
		}
	})
}

// FuzzProtoRecursive stresses recursive submessage traversal with arbitrary byte inputs,
// circular structures, and deeply nested frames.
func FuzzProtoRecursive(f *testing.F) {
	// Seed 1: Empty
	f.Add([]byte{})

	// Seed 2: Nested message
	sub := proto.AppendVarint(nil, 1, 42)
	msg := proto.AppendMessage(nil, 1, sub)
	f.Add(msg)

	// Seed 3: Truncated submessage
	var truncatedInner []byte
	truncatedInner = proto.AppendTag(truncatedInner, 1, proto.WireBytes)
	truncatedInner = binary.AppendUvarint(truncatedInner, 100)
	f.Add(proto.AppendMessage(nil, 1, truncatedInner))

	// Seed 4: Overflow submessage
	overflow := make([]byte, 11)
	for i := range overflow {
		overflow[i] = 0x80
	}
	f.Add(proto.AppendMessage(nil, 1, overflow))

	f.Fuzz(func(t *testing.T, data []byte) {
		visitedCount := 0
		_ = proto.ScanMessage(data, 16, func(depth int, field proto.Field) (bool, error) {
			visitedCount++
			if field.Num == 0 || field.Num > proto.MaxFieldNumber {
				t.Fatalf("invalid field number %d in recursive scan", field.Num)
			}
			if field.Type == proto.WireBytes {
				return true, nil
			}
			return false, nil
		})
	})
}
