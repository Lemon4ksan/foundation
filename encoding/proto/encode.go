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
	return binary.AppendUvarint(dst, uint64(Tag(fieldNum, wireType)))
}

// AppendVarint appends a uint64 varint field to dst.
func AppendVarint(dst []byte, fieldNum uint32, val uint64) []byte {
	dst = AppendTag(dst, fieldNum, WireVarint)
	return binary.AppendUvarint(dst, val)
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
func AppendUint32(dst []byte, fieldNum uint32, val uint32) []byte {
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
func AppendFixed32(dst []byte, fieldNum uint32, val uint32) []byte {
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
	dst = binary.AppendUvarint(dst, uint64(len(val)))
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
