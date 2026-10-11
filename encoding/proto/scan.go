// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto

import (
	"encoding/binary"
	"iter"
	"math"

	"github.com/lemon4ksan/foundation/silicon/bytesconv"
)

// Field represents a single decoded Protocol Buffers field from a raw stream.
type Field struct {
	Num  uint32
	Type WireType
	Raw  uint64 // Value for varint, fixed32, fixed64
	Data []byte // Sliced directly from source buffer for WireBytes (zero-copy)
}

// Varint returns the field value as an unsigned 64-bit varint.
//
//go:inline
func (f Field) Varint() uint64 { return f.Raw }

// Int32 returns the field value as a signed 32-bit integer.
//
//go:inline
func (f Field) Int32() int32 { return int32(f.Raw) }

// Int64 returns the field value as a signed 64-bit integer.
//
//go:inline
func (f Field) Int64() int64 { return int64(f.Raw) }

// Uint32 returns the field value as an unsigned 32-bit integer.
//
//go:inline
func (f Field) Uint32() uint32 { return uint32(f.Raw) }

// Uint64 returns the field value as an unsigned 64-bit integer.
//
//go:inline
func (f Field) Uint64() uint64 { return f.Raw }

// Sint32 returns the field value decoded from ZigZag format.
//
//go:inline
func (f Field) Sint32() int32 { return DecodeZigZag32(uint32(f.Raw)) }

// Sint64 returns the field value decoded from ZigZag format.
//
//go:inline
func (f Field) Sint64() int64 { return DecodeZigZag64(f.Raw) }

// Bool returns the field value as a boolean.
//
//go:inline
func (f Field) Bool() bool { return f.Raw != 0 }

// Fixed32 returns the field value as a 32-bit fixed integer.
//
//go:inline
func (f Field) Fixed32() uint32 { return uint32(f.Raw) }

// Fixed64 returns the field value as a 64-bit fixed integer.
//
//go:inline
func (f Field) Fixed64() uint64 { return f.Raw }

// Float32 returns the field value as a 32-bit floating point number.
//
//go:inline
func (f Field) Float32() float32 { return math.Float32frombits(uint32(f.Raw)) }

// Float64 returns the field value as a 64-bit floating point number.
//
//go:inline
func (f Field) Float64() float64 { return math.Float64frombits(f.Raw) }

// Bytes returns the length-delimited payload without heap allocations.
//
//go:inline
func (f Field) Bytes() []byte { return f.Data }

// String returns the length-delimited payload as a string view without copying bytes.
//
//go:inline
func (f Field) String() string { return bytesconv.B2S(f.Data) }

// Message returns the length-delimited payload representing an embedded sub-message.
//
//go:inline
func (f Field) Message() []byte { return f.Data }

// Scan returns a push iterator over fields in the Protocol Buffers wire buffer.
//
// It performs zero allocations and yields fields until the buffer is exhausted
// or an error occurs.
func Scan(data []byte) iter.Seq2[Field, error] {
	return func(yield func(Field, error) bool) {
		offset := 0
		n := len(data)

		for offset < n {
			tag, tagBytes := binary.Uvarint(data[offset:])
			if tagBytes <= 0 {
				yield(Field{}, ErrTruncated)
				return
			}
			offset += tagBytes

			fieldNum, wireType := SplitTag(uint32(tag))
			if fieldNum == 0 {
				yield(Field{}, ErrInvalidTag)
				return
			}

			var f Field
			f.Num = fieldNum
			f.Type = wireType

			switch wireType {
			case WireVarint:
				val, valBytes := binary.Uvarint(data[offset:])
				if valBytes <= 0 {
					yield(Field{}, ErrTruncated)
					return
				}
				offset += valBytes
				f.Raw = val

			case WireFixed64:
				if offset+8 > n {
					yield(Field{}, ErrTruncated)
					return
				}
				f.Raw = binary.LittleEndian.Uint64(data[offset : offset+8])
				offset += 8

			case WireBytes:
				length, lenBytes := binary.Uvarint(data[offset:])
				if lenBytes <= 0 {
					yield(Field{}, ErrTruncated)
					return
				}
				offset += lenBytes

				if uint64(offset)+length > uint64(n) {
					yield(Field{}, ErrTruncated)
					return
				}
				end := offset + int(length)
				f.Data = data[offset:end]
				offset = end

			case WireFixed32:
				if offset+4 > n {
					yield(Field{}, ErrTruncated)
					return
				}
				f.Raw = uint64(binary.LittleEndian.Uint32(data[offset : offset+4]))
				offset += 4

			case WireStartGroup, WireEndGroup:
				// Groups are deprecated in proto3; skip remaining wire
				yield(Field{}, ErrInvalidWireType)
				return

			default:
				yield(Field{}, ErrInvalidWireType)
				return
			}

			if !yield(f, nil) {
				return
			}
		}
	}
}

// SkipField determines the number of bytes consumed by a field of wireType in data.
func SkipField(data []byte, wireType WireType) (int, error) {
	n := len(data)
	switch wireType {
	case WireVarint:
		_, read := binary.Uvarint(data)
		if read <= 0 {
			return 0, ErrTruncated
		}
		return read, nil

	case WireFixed64:
		if n < 8 {
			return 0, ErrTruncated
		}
		return 8, nil

	case WireBytes:
		length, read := binary.Uvarint(data)
		if read <= 0 {
			return 0, ErrTruncated
		}
		total := read + int(length)
		if total > n || total < 0 {
			return 0, ErrTruncated
		}
		return total, nil

	case WireFixed32:
		if n < 4 {
			return 0, ErrTruncated
		}
		return 4, nil

	default:
		return 0, ErrInvalidWireType
	}
}
