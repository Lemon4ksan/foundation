// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto

import (
	"encoding/binary"
	"math"

	"github.com/lemon4ksan/foundation/silicon/bytesconv"
)

// AppendTag appends the encoded field tag (field number + wire type) to dst.
func AppendTag(dst []byte, fieldNum uint32, wireType WireType) []byte {
	return AppendUvarint(dst, uint64(Tag(fieldNum, wireType)))
}

// AppendVarint appends a uint64 varint field to dst.
func AppendVarint(dst []byte, fieldNum uint32, val uint64) []byte {
	dst = AppendTag(dst, fieldNum, WireVarint)
	return AppendUvarint(dst, val)
}

// AppendInt32 appends an int32 field to dst.
func AppendInt32(dst []byte, fieldNum uint32, val int32) []byte {
	return AppendVarint(dst, fieldNum, uint64(val))
}

// AppendInt64 appends an int64 field to dst.
func AppendInt64(dst []byte, fieldNum uint32, val int64) []byte {
	return AppendVarint(dst, fieldNum, uint64(val))
}

// AppendUint32 appends a uint32 field to dst.
func AppendUint32(dst []byte, fieldNum, val uint32) []byte {
	return AppendVarint(dst, fieldNum, uint64(val))
}

// AppendUint64 appends a uint64 field to dst.
func AppendUint64(dst []byte, fieldNum uint32, val uint64) []byte {
	return AppendVarint(dst, fieldNum, val)
}

// AppendSint32 appends an sint32 field encoded with ZigZag to dst.
func AppendSint32(dst []byte, fieldNum uint32, val int32) []byte {
	return AppendVarint(dst, fieldNum, uint64(EncodeZigZag32(val)))
}

// AppendSint64 appends an sint64 field encoded with ZigZag to dst.
func AppendSint64(dst []byte, fieldNum uint32, val int64) []byte {
	return AppendVarint(dst, fieldNum, EncodeZigZag64(val))
}

// AppendBool appends a bool field to dst.
func AppendBool(dst []byte, fieldNum uint32, val bool) []byte {
	var b uint64
	if val {
		b = 1
	}
	return AppendVarint(dst, fieldNum, b)
}

// AppendFixed32 appends a fixed 32-bit field to dst in little-endian order.
func AppendFixed32(dst []byte, fieldNum, val uint32) []byte {
	dst = AppendTag(dst, fieldNum, WireFixed32)
	return binary.LittleEndian.AppendUint32(dst, val)
}

// AppendFixed64 appends a fixed 64-bit field to dst in little-endian order.
func AppendFixed64(dst []byte, fieldNum uint32, val uint64) []byte {
	dst = AppendTag(dst, fieldNum, WireFixed64)
	return binary.LittleEndian.AppendUint64(dst, val)
}

// AppendFloat32 appends a 32-bit floating point field to dst.
func AppendFloat32(dst []byte, fieldNum uint32, val float32) []byte {
	return AppendFixed32(dst, fieldNum, math.Float32bits(val))
}

// AppendFloat64 appends a 64-bit floating point field to dst.
func AppendFloat64(dst []byte, fieldNum uint32, val float64) []byte {
	return AppendFixed64(dst, fieldNum, math.Float64bits(val))
}

// AppendBytes appends a length-delimited byte slice field to dst.
func AppendBytes(dst []byte, fieldNum uint32, val []byte) []byte {
	dst = AppendTag(dst, fieldNum, WireBytes)
	dst = AppendUvarint(dst, uint64(len(val)))
	return append(dst, val...)
}

// AppendString appends a length-delimited string field to dst.
func AppendString(dst []byte, fieldNum uint32, val string) []byte {
	return AppendBytes(dst, fieldNum, bytesconv.S2B(val))
}

// AppendMessage appends an embedded Protocol Buffers message to dst.
func AppendMessage(dst []byte, fieldNum uint32, msg []byte) []byte {
	return AppendBytes(dst, fieldNum, msg)
}

// AppendField appends a pre-constructed Field to dst.
func AppendField(dst []byte, f Field) []byte {
	dst = AppendTag(dst, f.Num, f.Type)
	switch f.Type {
	case WireVarint:
		return AppendUvarint(dst, f.Raw)
	case WireFixed64:
		return binary.LittleEndian.AppendUint64(dst, f.Raw)
	case WireBytes:
		dst = AppendUvarint(dst, uint64(len(f.Data)))
		return append(dst, f.Data...)
	case WireFixed32:
		return binary.LittleEndian.AppendUint32(dst, uint32(f.Raw))
	default:
		return dst
	}
}

// FieldSize returns the wire-encoded size of f in bytes, including tag.
func FieldSize(f Field) int {
	tagSize := TagSize(f.Num, f.Type)
	switch f.Type {
	case WireVarint:
		return tagSize + VarintLen(f.Raw)
	case WireFixed64:
		return tagSize + 8
	case WireBytes:
		return tagSize + VarintLen(uint64(len(f.Data))) + len(f.Data)
	case WireFixed32:
		return tagSize + 4
	default:
		return tagSize
	}
}

// VarintField constructs a WireVarint Field.
//
//go:inline
func VarintField(num uint32, val uint64) Field {
	return Field{Num: num, Type: WireVarint, Raw: val}
}

// Int64Field constructs a WireVarint Field from an int64.
//
//go:inline
func Int64Field(num uint32, val int64) Field {
	return Field{Num: num, Type: WireVarint, Raw: uint64(val)}
}

// Int32Field constructs a WireVarint Field from an int32.
//
//go:inline
func Int32Field(num uint32, val int32) Field {
	return Field{Num: num, Type: WireVarint, Raw: uint64(val)}
}

// Uint64Field constructs a WireVarint Field from a uint64.
//
//go:inline
func Uint64Field(num uint32, val uint64) Field {
	return Field{Num: num, Type: WireVarint, Raw: val}
}

// Uint32Field constructs a WireVarint Field from a uint32.
//
//go:inline
func Uint32Field(num, val uint32) Field {
	return Field{Num: num, Type: WireVarint, Raw: uint64(val)}
}

// Sint32Field constructs a WireVarint Field from an sint32 using ZigZag encoding.
//
//go:inline
func Sint32Field(num uint32, val int32) Field {
	return Field{Num: num, Type: WireVarint, Raw: uint64(EncodeZigZag32(val))}
}

// Sint64Field constructs a WireVarint Field from an sint64 using ZigZag encoding.
//
//go:inline
func Sint64Field(num uint32, val int64) Field {
	return Field{Num: num, Type: WireVarint, Raw: EncodeZigZag64(val)}
}

// BoolField constructs a WireVarint Field from a bool.
//
//go:inline
func BoolField(num uint32, val bool) Field {
	var raw uint64
	if val {
		raw = 1
	}
	return Field{Num: num, Type: WireVarint, Raw: raw}
}

// Fixed32Field constructs a WireFixed32 Field.
//
//go:inline
func Fixed32Field(num, val uint32) Field {
	return Field{Num: num, Type: WireFixed32, Raw: uint64(val)}
}

// Fixed64Field constructs a WireFixed64 Field.
//
//go:inline
func Fixed64Field(num uint32, val uint64) Field {
	return Field{Num: num, Type: WireFixed64, Raw: val}
}

// Float32Field constructs a WireFixed32 Field from a float32.
//
//go:inline
func Float32Field(num uint32, val float32) Field {
	return Field{Num: num, Type: WireFixed32, Raw: uint64(math.Float32bits(val))}
}

// Float64Field constructs a WireFixed64 Field from a float64.
//
//go:inline
func Float64Field(num uint32, val float64) Field {
	return Field{Num: num, Type: WireFixed64, Raw: math.Float64bits(val)}
}

// BytesField constructs a WireBytes Field from a byte slice (zero-copy).
//
//go:inline
func BytesField(num uint32, val []byte) Field {
	return Field{Num: num, Type: WireBytes, Data: val}
}

// StringField constructs a WireBytes Field from a string (zero-copy view).
//
//go:inline
func StringField(num uint32, val string) Field {
	return Field{Num: num, Type: WireBytes, Data: bytesconv.S2B(val)}
}

// MessageField constructs a WireBytes Field representing an embedded submessage.
//
//go:inline
func MessageField(num uint32, val []byte) Field {
	return Field{Num: num, Type: WireBytes, Data: val}
}
