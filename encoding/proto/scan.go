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

// Scan returns a push iterator over fields if this field contains an embedded submessage.
//
//go:inline
func (f Field) Scan() iter.Seq2[Field, error] {
	return Scan(f.Data)
}

// Scan returns a push iterator over fields in the Protocol Buffers wire buffer.
//
// It performs zero allocations and yields fields until the buffer is exhausted
// or an error occurs.
func Scan(data []byte) iter.Seq2[Field, error] {
	return func(yield func(Field, error) bool) {
		offset := 0
		n := len(data)

		for offset < n {
			b0 := data[offset]
			var tag uint64
			var tagBytes int

			if b0 < 0x80 {
				tag = uint64(b0)
				tagBytes = 1
			} else {
				var err error
				tag, tagBytes, err = DecodeUvarint(data[offset:])
				if err != nil {
					yield(Field{}, err)
					return
				}
			}
			offset += tagBytes

			if tag > math.MaxUint32 {
				yield(Field{}, ErrInvalidTag)
				return
			}

			fieldNum, wireType := SplitTag(uint32(tag))
			if fieldNum == 0 || fieldNum > MaxFieldNumber {
				yield(Field{}, ErrInvalidTag)
				return
			}

			var f Field
			f.Num = fieldNum
			f.Type = wireType

			switch wireType {
			case WireVarint:
				if offset < n && data[offset] < 0x80 {
					f.Raw = uint64(data[offset])
					offset++
				} else {
					val, valBytes, err := DecodeUvarint(data[offset:])
					if err != nil {
						yield(Field{}, err)
						return
					}
					offset += valBytes
					f.Raw = val
				}

			case WireFixed64:
				if n-offset < 8 {
					yield(Field{}, ErrTruncated)
					return
				}
				f.Raw = binary.LittleEndian.Uint64(data[offset : offset+8])
				offset += 8

			case WireBytes:
				var length uint64
				if offset < n && data[offset] < 0x80 {
					length = uint64(data[offset])
					offset++
				} else {
					var lenBytes int
					var err error
					length, lenBytes, err = DecodeUvarint(data[offset:])
					if err != nil {
						yield(Field{}, err)
						return
					}
					offset += lenBytes
				}

				rem := n - offset
				if length > uint64(rem) {
					yield(Field{}, ErrTruncated)
					return
				}
				end := offset + int(length)
				f.Data = data[offset:end]
				offset = end

			case WireFixed32:
				if n-offset < 4 {
					yield(Field{}, ErrTruncated)
					return
				}
				f.Raw = uint64(binary.LittleEndian.Uint32(data[offset : offset+4]))
				offset += 4

			case WireStartGroup, WireEndGroup:
				// Groups are deprecated in proto3; invalid wire type
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
		if n > 0 && data[0] < 0x80 {
			return 1, nil
		}
		_, read, err := DecodeUvarint(data)
		if err != nil {
			return 0, err
		}
		return read, nil

	case WireFixed64:
		if n < 8 {
			return 0, ErrTruncated
		}
		return 8, nil

	case WireBytes:
		var length uint64
		var read int
		if n > 0 && data[0] < 0x80 {
			length = uint64(data[0])
			read = 1
		} else {
			var err error
			length, read, err = DecodeUvarint(data)
			if err != nil {
				return 0, err
			}
		}
		rem := n - read
		if length > uint64(rem) {
			return 0, ErrTruncated
		}
		return read + int(length), nil

	case WireFixed32:
		if n < 4 {
			return 0, ErrTruncated
		}
		return 4, nil

	default:
		return 0, ErrInvalidWireType
	}
}

// SkipTagAndField determines the total number of bytes consumed by a tag and its field value in data.
func SkipTagAndField(data []byte) (int, error) {
	tag, tagBytes, err := DecodeUvarint(data)
	if err != nil {
		return 0, err
	}
	if tag > math.MaxUint32 {
		return 0, ErrInvalidTag
	}
	fieldNum, wireType := SplitTag(uint32(tag))
	if fieldNum == 0 || fieldNum > MaxFieldNumber {
		return 0, ErrInvalidTag
	}
	if wireType == WireStartGroup {
		groupLen, err := SkipGroup(data[tagBytes:], fieldNum)
		if err != nil {
			return 0, err
		}
		return tagBytes + groupLen, nil
	}
	valBytes, err := SkipField(data[tagBytes:], wireType)
	if err != nil {
		return 0, err
	}
	return tagBytes + valBytes, nil
}

// SkipGroup skips a group from data until the matching WireEndGroup tag with groupFieldNum is encountered.
func SkipGroup(data []byte, groupFieldNum uint32) (int, error) {
	return skipGroupDepth(data, groupFieldNum, 0, DefaultMaxDepth)
}

func skipGroupDepth(data []byte, groupFieldNum uint32, depth, maxDepth int) (int, error) {
	if depth >= maxDepth {
		return 0, ErrMaxDepthExceeded
	}
	offset := 0
	n := len(data)

	for offset < n {
		tag, tagBytes, err := DecodeUvarint(data[offset:])
		if err != nil {
			return 0, err
		}
		offset += tagBytes

		if tag > math.MaxUint32 {
			return 0, ErrInvalidTag
		}

		fieldNum, wireType := SplitTag(uint32(tag))
		if fieldNum == 0 || fieldNum > MaxFieldNumber {
			return 0, ErrInvalidTag
		}

		if wireType == WireEndGroup {
			if fieldNum == groupFieldNum {
				return offset, nil
			}
			return 0, ErrInvalidWireType
		}

		if wireType == WireStartGroup {
			skipped, err := skipGroupDepth(data[offset:], fieldNum, depth+1, maxDepth)
			if err != nil {
				return 0, err
			}
			offset += skipped
			continue
		}

		skipped, err := SkipField(data[offset:], wireType)
		if err != nil {
			return 0, err
		}
		offset += skipped
	}

	return 0, ErrTruncated
}
