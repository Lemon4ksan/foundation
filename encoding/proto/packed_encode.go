// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto

import (
	"encoding/binary"
	"math"
	"unsafe"
)

var isLittleEndian = true

func init() {
	var x uint16 = 1
	isLittleEndian = *(*byte)(unsafe.Pointer(&x)) == 1
}

// growSlice ensures dst has capacity for extra bytes, reusing dst without allocations if capacity suffices.
//
//go:inline
func growSlice(dst []byte, extra int) []byte {
	n := len(dst)
	needed := n + extra
	if needed <= cap(dst) {
		return dst[:needed]
	}
	newCap := max(2*cap(dst), needed)
	newDst := make([]byte, needed, newCap)
	copy(newDst, dst)
	return newDst
}

// AppendPackedVarints appends a packed repeated uint64 varint field to dst.
func AppendPackedVarints(dst []byte, fieldNum uint32, vals []uint64) []byte {
	if len(vals) == 0 {
		return dst
	}

	totalLen := 0
	allSingle := true
	i := 0
	for i+8 <= len(vals) {
		orSum := vals[i] | vals[i+1] | vals[i+2] | vals[i+3] |
			vals[i+4] | vals[i+5] | vals[i+6] | vals[i+7]
		if (orSum &^ 0x7f) == 0 {
			totalLen += 8
			i += 8
			continue
		}
		allSingle = false
		for j := range 8 {
			totalLen += SizeVarint(vals[i+j])
		}
		i += 8
	}
	for ; i < len(vals); i++ {
		v := vals[i]
		if v >= 0x80 {
			allSingle = false
			totalLen += SizeVarint(v)
		} else {
			totalLen++
		}
	}

	dst = AppendTag(dst, fieldNum, WireBytes)
	dst = AppendUvarint(dst, uint64(totalLen))
	n := len(dst)
	dst = growSlice(dst, totalLen)

	if allSingle {
		w := n
		idx := 0
		for idx+8 <= len(vals) {
			packed := (vals[idx+0] & 0xff) |
				((vals[idx+1] & 0xff) << 8) |
				((vals[idx+2] & 0xff) << 16) |
				((vals[idx+3] & 0xff) << 24) |
				((vals[idx+4] & 0xff) << 32) |
				((vals[idx+5] & 0xff) << 40) |
				((vals[idx+6] & 0xff) << 48) |
				((vals[idx+7] & 0xff) << 56)
			binary.LittleEndian.PutUint64(dst[w:], packed)
			w += 8
			idx += 8
		}
		for ; idx < len(vals); idx++ {
			dst[w] = byte(vals[idx])
			w++
		}
		return dst
	}

	w := n
	for _, v := range vals {
		w += PutUvarint(dst[w:], v)
	}
	return dst
}

// AppendPackedUint64 appends a packed repeated uint64 field to dst.
func AppendPackedUint64(dst []byte, fieldNum uint32, vals []uint64) []byte {
	return AppendPackedVarints(dst, fieldNum, vals)
}

// AppendPackedInt64 appends a packed repeated int64 field to dst.
func AppendPackedInt64(dst []byte, fieldNum uint32, vals []int64) []byte {
	if len(vals) == 0 {
		return dst
	}
	return AppendPackedVarints(dst, fieldNum, *(*[]uint64)(unsafe.Pointer(&vals)))
}

// AppendPackedUint32 appends a packed repeated uint32 field to dst.
func AppendPackedUint32(dst []byte, fieldNum uint32, vals []uint32) []byte {
	if len(vals) == 0 {
		return dst
	}
	totalLen := 0
	allSingle := true
	for _, v := range vals {
		if v >= 0x80 {
			allSingle = false
			totalLen += SizeVarint(uint64(v))
		} else {
			totalLen++
		}
	}

	dst = AppendTag(dst, fieldNum, WireBytes)
	dst = AppendUvarint(dst, uint64(totalLen))
	n := len(dst)
	dst = growSlice(dst, totalLen)

	if allSingle {
		w := n
		for _, v := range vals {
			dst[w] = byte(v)
			w++
		}
		return dst
	}

	w := n
	for _, v := range vals {
		w += PutUvarint(dst[w:], uint64(v))
	}
	return dst
}

// AppendPackedInt32 appends a packed repeated int32 field to dst.
// Negative values are sign-extended to 10-byte varints per protobuf specification.
func AppendPackedInt32(dst []byte, fieldNum uint32, vals []int32) []byte {
	if len(vals) == 0 {
		return dst
	}
	totalLen := 0
	allSingle := true
	for _, v := range vals {
		u := uint64(int64(v))
		if u >= 0x80 {
			allSingle = false
			totalLen += SizeVarint(u)
		} else {
			totalLen++
		}
	}

	dst = AppendTag(dst, fieldNum, WireBytes)
	dst = AppendUvarint(dst, uint64(totalLen))
	n := len(dst)
	dst = growSlice(dst, totalLen)

	if allSingle {
		w := n
		for _, v := range vals {
			dst[w] = byte(v)
			w++
		}
		return dst
	}

	w := n
	for _, v := range vals {
		w += PutUvarint(dst[w:], uint64(int64(v)))
	}
	return dst
}

// AppendPackedFixed32 appends a packed repeated fixed32 (uint32) field to dst.
func AppendPackedFixed32(dst []byte, fieldNum uint32, vals []uint32) []byte {
	if len(vals) == 0 {
		return dst
	}
	byteLen := len(vals) * 4
	dst = AppendTag(dst, fieldNum, WireBytes)
	dst = AppendUvarint(dst, uint64(byteLen))
	n := len(dst)
	dst = growSlice(dst, byteLen)

	if isLittleEndian {
		src := unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(vals))), byteLen)
		copy(dst[n:], src)
	} else {
		for i, v := range vals {
			binary.LittleEndian.PutUint32(dst[n+i*4:], v)
		}
	}
	return dst
}

// AppendPackedSfixed32 appends a packed repeated sfixed32 (int32) field to dst.
func AppendPackedSfixed32(dst []byte, fieldNum uint32, vals []int32) []byte {
	if len(vals) == 0 {
		return dst
	}
	return AppendPackedFixed32(dst, fieldNum, *(*[]uint32)(unsafe.Pointer(&vals)))
}

// AppendPackedFixed64 appends a packed repeated fixed64 (uint64) field to dst.
func AppendPackedFixed64(dst []byte, fieldNum uint32, vals []uint64) []byte {
	if len(vals) == 0 {
		return dst
	}
	byteLen := len(vals) * 8
	dst = AppendTag(dst, fieldNum, WireBytes)
	dst = AppendUvarint(dst, uint64(byteLen))
	n := len(dst)
	dst = growSlice(dst, byteLen)

	if isLittleEndian {
		src := unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(vals))), byteLen)
		copy(dst[n:], src)
	} else {
		for i, v := range vals {
			binary.LittleEndian.PutUint64(dst[n+i*8:], v)
		}
	}
	return dst
}

// AppendPackedSfixed64 appends a packed repeated sfixed64 (int64) field to dst.
func AppendPackedSfixed64(dst []byte, fieldNum uint32, vals []int64) []byte {
	if len(vals) == 0 {
		return dst
	}
	return AppendPackedFixed64(dst, fieldNum, *(*[]uint64)(unsafe.Pointer(&vals)))
}

// AppendPackedFloat32 appends a packed repeated float32 field to dst.
func AppendPackedFloat32(dst []byte, fieldNum uint32, vals []float32) []byte {
	if len(vals) == 0 {
		return dst
	}
	byteLen := len(vals) * 4
	dst = AppendTag(dst, fieldNum, WireBytes)
	dst = AppendUvarint(dst, uint64(byteLen))
	n := len(dst)
	dst = growSlice(dst, byteLen)

	if isLittleEndian {
		src := unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(vals))), byteLen)
		copy(dst[n:], src)
	} else {
		for i, v := range vals {
			binary.LittleEndian.PutUint32(dst[n+i*4:], math.Float32bits(v))
		}
	}
	return dst
}

// AppendPackedFloat64 appends a packed repeated float64 (double) field to dst.
func AppendPackedFloat64(dst []byte, fieldNum uint32, vals []float64) []byte {
	if len(vals) == 0 {
		return dst
	}
	byteLen := len(vals) * 8
	dst = AppendTag(dst, fieldNum, WireBytes)
	dst = AppendUvarint(dst, uint64(byteLen))
	n := len(dst)
	dst = growSlice(dst, byteLen)

	if isLittleEndian {
		src := unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(vals))), byteLen)
		copy(dst[n:], src)
	} else {
		for i, v := range vals {
			binary.LittleEndian.PutUint64(dst[n+i*8:], math.Float64bits(v))
		}
	}
	return dst
}

// AppendPackedSint32 appends a packed repeated sint32 field encoded with ZigZag to dst.
func AppendPackedSint32(dst []byte, fieldNum uint32, vals []int32) []byte {
	if len(vals) == 0 {
		return dst
	}
	totalLen := 0
	allSingle := true
	for _, v := range vals {
		zz := EncodeZigZag32(v)
		if zz >= 0x80 {
			allSingle = false
			totalLen += SizeVarint(uint64(zz))
		} else {
			totalLen++
		}
	}

	dst = AppendTag(dst, fieldNum, WireBytes)
	dst = AppendUvarint(dst, uint64(totalLen))
	n := len(dst)
	dst = growSlice(dst, totalLen)

	if allSingle {
		w := n
		for _, v := range vals {
			dst[w] = byte(EncodeZigZag32(v))
			w++
		}
		return dst
	}

	w := n
	for _, v := range vals {
		w += PutUvarint(dst[w:], uint64(EncodeZigZag32(v)))
	}
	return dst
}

// AppendPackedSint64 appends a packed repeated sint64 field encoded with ZigZag to dst.
func AppendPackedSint64(dst []byte, fieldNum uint32, vals []int64) []byte {
	if len(vals) == 0 {
		return dst
	}
	totalLen := 0
	allSingle := true
	for _, v := range vals {
		zz := EncodeZigZag64(v)
		if zz >= 0x80 {
			allSingle = false
			totalLen += SizeVarint(zz)
		} else {
			totalLen++
		}
	}

	dst = AppendTag(dst, fieldNum, WireBytes)
	dst = AppendUvarint(dst, uint64(totalLen))
	n := len(dst)
	dst = growSlice(dst, totalLen)

	if allSingle {
		w := n
		for _, v := range vals {
			dst[w] = byte(EncodeZigZag64(v))
			w++
		}
		return dst
	}

	w := n
	for _, v := range vals {
		w += PutUvarint(dst[w:], EncodeZigZag64(v))
	}
	return dst
}

// AppendPackedBools appends a packed repeated bool field to dst.
func AppendPackedBools(dst []byte, fieldNum uint32, vals []bool) []byte {
	if len(vals) == 0 {
		return dst
	}
	byteLen := len(vals)
	dst = AppendTag(dst, fieldNum, WireBytes)
	dst = AppendUvarint(dst, uint64(byteLen))
	n := len(dst)
	dst = growSlice(dst, byteLen)

	src := unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(vals))), byteLen)
	copy(dst[n:], src)
	return dst
}
