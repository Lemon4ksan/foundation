// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto

import (
	"encoding/binary"
	"iter"
	"math"
	"unsafe"
)

// PackedVarints returns an iterator yielding uint64 varints from a packed repeated field.
func (f Field) PackedVarints() iter.Seq2[uint64, error] { return ScanPackedVarints(f.Data) }

// PackedFixed32 returns an iterator yielding uint32 fixed values from a packed repeated field.
func (f Field) PackedFixed32() iter.Seq2[uint32, error] { return ScanPackedFixed32(f.Data) }

// PackedFixed64 returns an iterator yielding uint64 fixed values from a packed repeated field.
func (f Field) PackedFixed64() iter.Seq2[uint64, error] { return ScanPackedFixed64(f.Data) }

// PackedFloat32 returns an iterator yielding float32 values from a packed repeated field.
func (f Field) PackedFloat32() iter.Seq2[float32, error] { return ScanPackedFloat32(f.Data) }

// PackedFloat64 returns an iterator yielding float64 values from a packed repeated field.
func (f Field) PackedFloat64() iter.Seq2[float64, error] { return ScanPackedFloat64(f.Data) }

// PackedSint32 returns an iterator yielding int32 ZigZag decoded values from a packed repeated field.
func (f Field) PackedSint32() iter.Seq2[int32, error] { return ScanPackedSint32(f.Data) }

// PackedSint64 returns an iterator yielding int64 ZigZag decoded values from a packed repeated field.
func (f Field) PackedSint64() iter.Seq2[int64, error] { return ScanPackedSint64(f.Data) }

// PackedBools returns an iterator yielding bool values from a packed repeated field.
func (f Field) PackedBools() iter.Seq2[bool, error] { return ScanPackedBools(f.Data) }

// PackedInt32 returns an iterator yielding int32 varints from a packed repeated field.
func (f Field) PackedInt32() iter.Seq2[int32, error] { return ScanPackedInt32(f.Data) }

// PackedInt64 returns an iterator yielding int64 varints from a packed repeated field.
func (f Field) PackedInt64() iter.Seq2[int64, error] { return ScanPackedInt64(f.Data) }

// PackedUint32 returns an iterator yielding uint32 varints from a packed repeated field.
func (f Field) PackedUint32() iter.Seq2[uint32, error] { return ScanPackedUint32(f.Data) }

// PackedUint64 returns an iterator yielding uint64 varints from a packed repeated field.
func (f Field) PackedUint64() iter.Seq2[uint64, error] { return ScanPackedUint64(f.Data) }

// ScanPackedVarints returns a zero-allocation iterator over packed uint64 varints.
// It leverages 64-bit SWAR batch checks to process 8 single-byte varints at once.
func ScanPackedVarints(data []byte) iter.Seq2[uint64, error] {
	return func(yield func(uint64, error) bool) {
		offset := 0
		n := len(data)

		for offset+8 <= n {
			v := binary.LittleEndian.Uint64(data[offset:])
			if (v & 0x8080808080808080) == 0 {
				offset += 8
				if !yield(uint64(byte(v)), nil) ||
					!yield(uint64(byte(v>>8)), nil) ||
					!yield(uint64(byte(v>>16)), nil) ||
					!yield(uint64(byte(v>>24)), nil) ||
					!yield(uint64(byte(v>>32)), nil) ||
					!yield(uint64(byte(v>>40)), nil) ||
					!yield(uint64(byte(v>>48)), nil) ||
					!yield(uint64(byte(v>>56)), nil) {
					return
				}
				continue
			}

			b := byte(v)
			if b < 0x80 {
				offset++
				if !yield(uint64(b), nil) {
					return
				}
				continue
			}

			val, readBytes, err := DecodeUvarint(data[offset:])
			if err != nil {
				yield(0, err)
				return
			}
			offset += readBytes
			if !yield(val, nil) {
				return
			}
		}

		for offset < n {
			b := data[offset]
			if b < 0x80 {
				offset++
				if !yield(uint64(b), nil) {
					return
				}
				continue
			}

			val, readBytes, err := DecodeUvarint(data[offset:])
			if err != nil {
				yield(0, err)
				return
			}
			offset += readBytes
			if !yield(val, nil) {
				return
			}
		}
	}
}

// ScanPackedUint64 returns a zero-allocation iterator over packed uint64 varints.
func ScanPackedUint64(data []byte) iter.Seq2[uint64, error] {
	return ScanPackedVarints(data)
}

// ScanPackedInt64 returns a zero-allocation iterator over packed int64 varints.
func ScanPackedInt64(data []byte) iter.Seq2[int64, error] {
	return func(yield func(int64, error) bool) {
		for v, err := range ScanPackedVarints(data) {
			if err != nil {
				yield(0, err)
				return
			}
			if !yield(int64(v), nil) {
				return
			}
		}
	}
}

// ScanPackedUint32 returns a zero-allocation iterator over packed uint32 varints.
func ScanPackedUint32(data []byte) iter.Seq2[uint32, error] {
	return func(yield func(uint32, error) bool) {
		for v, err := range ScanPackedVarints(data) {
			if err != nil {
				yield(0, err)
				return
			}
			if !yield(uint32(v), nil) {
				return
			}
		}
	}
}

// ScanPackedInt32 returns a zero-allocation iterator over packed int32 varints.
func ScanPackedInt32(data []byte) iter.Seq2[int32, error] {
	return func(yield func(int32, error) bool) {
		for v, err := range ScanPackedVarints(data) {
			if err != nil {
				yield(0, err)
				return
			}
			if !yield(int32(v), nil) {
				return
			}
		}
	}
}

// ScanPackedFixed32 returns an unrolled zero-allocation iterator over packed fixed32 values.
func ScanPackedFixed32(data []byte) iter.Seq2[uint32, error] {
	return func(yield func(uint32, error) bool) {
		n := len(data)
		offset := 0
		for offset+16 <= n {
			v0 := binary.LittleEndian.Uint32(data[offset:])
			v1 := binary.LittleEndian.Uint32(data[offset+4:])
			v2 := binary.LittleEndian.Uint32(data[offset+8:])
			v3 := binary.LittleEndian.Uint32(data[offset+12:])
			offset += 16
			if !yield(v0, nil) || !yield(v1, nil) || !yield(v2, nil) || !yield(v3, nil) {
				return
			}
		}
		for offset+4 <= n {
			v := binary.LittleEndian.Uint32(data[offset : offset+4])
			offset += 4
			if !yield(v, nil) {
				return
			}
		}
		if offset < n {
			yield(0, ErrTruncated)
		}
	}
}

// ScanPackedFixed64 returns an unrolled zero-allocation iterator over packed fixed64 values.
func ScanPackedFixed64(data []byte) iter.Seq2[uint64, error] {
	return func(yield func(uint64, error) bool) {
		n := len(data)
		offset := 0
		for offset+32 <= n {
			v0 := binary.LittleEndian.Uint64(data[offset:])
			v1 := binary.LittleEndian.Uint64(data[offset+8:])
			v2 := binary.LittleEndian.Uint64(data[offset+16:])
			v3 := binary.LittleEndian.Uint64(data[offset+24:])
			offset += 32
			if !yield(v0, nil) || !yield(v1, nil) || !yield(v2, nil) || !yield(v3, nil) {
				return
			}
		}
		for offset+8 <= n {
			v := binary.LittleEndian.Uint64(data[offset : offset+8])
			offset += 8
			if !yield(v, nil) {
				return
			}
		}
		if offset < n {
			yield(0, ErrTruncated)
		}
	}
}

// ScanPackedFloat32 returns an unrolled zero-allocation iterator over packed float32 values.
func ScanPackedFloat32(data []byte) iter.Seq2[float32, error] {
	return func(yield func(float32, error) bool) {
		n := len(data)
		offset := 0
		for offset+16 <= n {
			v0 := math.Float32frombits(binary.LittleEndian.Uint32(data[offset:]))
			v1 := math.Float32frombits(binary.LittleEndian.Uint32(data[offset+4:]))
			v2 := math.Float32frombits(binary.LittleEndian.Uint32(data[offset+8:]))
			v3 := math.Float32frombits(binary.LittleEndian.Uint32(data[offset+12:]))
			offset += 16
			if !yield(v0, nil) || !yield(v1, nil) || !yield(v2, nil) || !yield(v3, nil) {
				return
			}
		}
		for offset+4 <= n {
			v := math.Float32frombits(binary.LittleEndian.Uint32(data[offset : offset+4]))
			offset += 4
			if !yield(v, nil) {
				return
			}
		}
		if offset < n {
			yield(0, ErrTruncated)
		}
	}
}

// ScanPackedFloat64 returns an unrolled zero-allocation iterator over packed float64 values.
func ScanPackedFloat64(data []byte) iter.Seq2[float64, error] {
	return func(yield func(float64, error) bool) {
		n := len(data)
		offset := 0
		for offset+32 <= n {
			v0 := math.Float64frombits(binary.LittleEndian.Uint64(data[offset:]))
			v1 := math.Float64frombits(binary.LittleEndian.Uint64(data[offset+8:]))
			v2 := math.Float64frombits(binary.LittleEndian.Uint64(data[offset+16:]))
			v3 := math.Float64frombits(binary.LittleEndian.Uint64(data[offset+24:]))
			offset += 32
			if !yield(v0, nil) || !yield(v1, nil) || !yield(v2, nil) || !yield(v3, nil) {
				return
			}
		}
		for offset+8 <= n {
			v := math.Float64frombits(binary.LittleEndian.Uint64(data[offset : offset+8]))
			offset += 8
			if !yield(v, nil) {
				return
			}
		}
		if offset < n {
			yield(0, ErrTruncated)
		}
	}
}

// ScanPackedSint32 returns an iterator over packed ZigZag-encoded sint32 values.
func ScanPackedSint32(data []byte) iter.Seq2[int32, error] {
	return func(yield func(int32, error) bool) {
		n := len(data)
		offset := 0
		for offset+8 <= n {
			v := binary.LittleEndian.Uint64(data[offset:])
			if (v & 0x8080808080808080) == 0 {
				offset += 8
				if !yield(DecodeZigZag32(uint32(byte(v))), nil) ||
					!yield(DecodeZigZag32(uint32(byte(v>>8))), nil) ||
					!yield(DecodeZigZag32(uint32(byte(v>>16))), nil) ||
					!yield(DecodeZigZag32(uint32(byte(v>>24))), nil) ||
					!yield(DecodeZigZag32(uint32(byte(v>>32))), nil) ||
					!yield(DecodeZigZag32(uint32(byte(v>>40))), nil) ||
					!yield(DecodeZigZag32(uint32(byte(v>>48))), nil) ||
					!yield(DecodeZigZag32(uint32(byte(v>>56))), nil) {
					return
				}
				continue
			}

			b := byte(v)
			if b < 0x80 {
				offset++
				if !yield(DecodeZigZag32(uint32(b)), nil) {
					return
				}
				continue
			}

			raw, readBytes, err := DecodeUvarint(data[offset:])
			if err != nil {
				yield(0, err)
				return
			}
			offset += readBytes
			if !yield(DecodeZigZag32(uint32(raw)), nil) {
				return
			}
		}

		for offset < n {
			b := data[offset]
			if b < 0x80 {
				offset++
				if !yield(DecodeZigZag32(uint32(b)), nil) {
					return
				}
				continue
			}

			raw, readBytes, err := DecodeUvarint(data[offset:])
			if err != nil {
				yield(0, err)
				return
			}
			offset += readBytes
			if !yield(DecodeZigZag32(uint32(raw)), nil) {
				return
			}
		}
	}
}

// ScanPackedSint64 returns an iterator over packed ZigZag-encoded sint64 values.
func ScanPackedSint64(data []byte) iter.Seq2[int64, error] {
	return func(yield func(int64, error) bool) {
		n := len(data)
		offset := 0
		for offset+8 <= n {
			v := binary.LittleEndian.Uint64(data[offset:])
			if (v & 0x8080808080808080) == 0 {
				offset += 8
				if !yield(DecodeZigZag64(uint64(byte(v))), nil) ||
					!yield(DecodeZigZag64(uint64(byte(v>>8))), nil) ||
					!yield(DecodeZigZag64(uint64(byte(v>>16))), nil) ||
					!yield(DecodeZigZag64(uint64(byte(v>>24))), nil) ||
					!yield(DecodeZigZag64(uint64(byte(v>>32))), nil) ||
					!yield(DecodeZigZag64(uint64(byte(v>>40))), nil) ||
					!yield(DecodeZigZag64(uint64(byte(v>>48))), nil) ||
					!yield(DecodeZigZag64(uint64(byte(v>>56))), nil) {
					return
				}
				continue
			}

			b := byte(v)
			if b < 0x80 {
				offset++
				if !yield(DecodeZigZag64(uint64(b)), nil) {
					return
				}
				continue
			}

			raw, readBytes, err := DecodeUvarint(data[offset:])
			if err != nil {
				yield(0, err)
				return
			}
			offset += readBytes
			if !yield(DecodeZigZag64(raw), nil) {
				return
			}
		}

		for offset < n {
			b := data[offset]
			if b < 0x80 {
				offset++
				if !yield(DecodeZigZag64(uint64(b)), nil) {
					return
				}
				continue
			}

			raw, readBytes, err := DecodeUvarint(data[offset:])
			if err != nil {
				yield(0, err)
				return
			}
			offset += readBytes
			if !yield(DecodeZigZag64(raw), nil) {
				return
			}
		}
	}
}

// ScanPackedBools returns an iterator over packed bool values.
func ScanPackedBools(data []byte) iter.Seq2[bool, error] {
	return func(yield func(bool, error) bool) {
		n := len(data)
		offset := 0
		for offset+8 <= n {
			v := binary.LittleEndian.Uint64(data[offset:])
			if (v & 0x8080808080808080) == 0 {
				offset += 8
				if !yield(byte(v) != 0, nil) ||
					!yield(byte(v>>8) != 0, nil) ||
					!yield(byte(v>>16) != 0, nil) ||
					!yield(byte(v>>24) != 0, nil) ||
					!yield(byte(v>>32) != 0, nil) ||
					!yield(byte(v>>40) != 0, nil) ||
					!yield(byte(v>>48) != 0, nil) ||
					!yield(byte(v>>56) != 0, nil) {
					return
				}
				continue
			}

			b := byte(v)
			if b < 0x80 {
				offset++
				if !yield(b != 0, nil) {
					return
				}
				continue
			}

			raw, readBytes, err := DecodeUvarint(data[offset:])
			if err != nil {
				yield(false, err)
				return
			}
			offset += readBytes
			if !yield(raw != 0, nil) {
				return
			}
		}

		for offset < n {
			b := data[offset]
			if b < 0x80 {
				offset++
				if !yield(b != 0, nil) {
					return
				}
				continue
			}

			raw, readBytes, err := DecodeUvarint(data[offset:])
			if err != nil {
				yield(false, err)
				return
			}
			offset += readBytes
			if !yield(raw != 0, nil) {
				return
			}
		}
	}
}

// DecodePackedFixed32 decodes packed fixed32 values directly into dst with zero heap allocations.
func DecodePackedFixed32(data []byte, dst []uint32) ([]uint32, error) {
	n := len(data)
	if n%4 != 0 {
		return dst[:0], ErrTruncated
	}
	count := n / 4
	if cap(dst) < count {
		dst = make([]uint32, count)
	} else {
		dst = dst[:count]
	}
	if count == 0 {
		return dst, nil
	}
	if isLittleEndian {
		copy(unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(dst))), n), data)
	} else {
		for i := range count {
			dst[i] = binary.LittleEndian.Uint32(data[i*4 : (i+1)*4])
		}
	}
	return dst, nil
}

// DecodePackedFixed64 decodes packed fixed64 values directly into dst with zero heap allocations.
func DecodePackedFixed64(data []byte, dst []uint64) ([]uint64, error) {
	n := len(data)
	if n%8 != 0 {
		return dst[:0], ErrTruncated
	}
	count := n / 8
	if cap(dst) < count {
		dst = make([]uint64, count)
	} else {
		dst = dst[:count]
	}
	if count == 0 {
		return dst, nil
	}
	if isLittleEndian {
		copy(unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(dst))), n), data)
	} else {
		for i := range count {
			dst[i] = binary.LittleEndian.Uint64(data[i*8 : (i+1)*8])
		}
	}
	return dst, nil
}

// DecodePackedFloat32 decodes packed float32 values directly into dst with zero heap allocations.
func DecodePackedFloat32(data []byte, dst []float32) ([]float32, error) {
	n := len(data)
	if n%4 != 0 {
		return dst[:0], ErrTruncated
	}
	count := n / 4
	if cap(dst) < count {
		dst = make([]float32, count)
	} else {
		dst = dst[:count]
	}
	if count == 0 {
		return dst, nil
	}
	if isLittleEndian {
		copy(unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(dst))), n), data)
	} else {
		for i := range count {
			dst[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[i*4 : (i+1)*4]))
		}
	}
	return dst, nil
}

// DecodePackedFloat64 decodes packed float64 values directly into dst with zero heap allocations.
func DecodePackedFloat64(data []byte, dst []float64) ([]float64, error) {
	n := len(data)
	if n%8 != 0 {
		return dst[:0], ErrTruncated
	}
	count := n / 8
	if cap(dst) < count {
		dst = make([]float64, count)
	} else {
		dst = dst[:count]
	}
	if count == 0 {
		return dst, nil
	}
	if isLittleEndian {
		copy(unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(dst))), n), data)
	} else {
		for i := range count {
			dst[i] = math.Float64frombits(binary.LittleEndian.Uint64(data[i*8 : (i+1)*8]))
		}
	}
	return dst, nil
}

// DecodePackedBools decodes packed bool values directly into dst with zero heap allocations.
func DecodePackedBools(data []byte, dst []bool) ([]bool, error) {
	dst = dst[:0]
	n := len(data)
	offset := 0
	for offset+8 <= n {
		v := binary.LittleEndian.Uint64(data[offset:])
		if (v & 0x8080808080808080) == 0 {
			dst = append(dst,
				byte(v) != 0,
				byte(v>>8) != 0,
				byte(v>>16) != 0,
				byte(v>>24) != 0,
				byte(v>>32) != 0,
				byte(v>>40) != 0,
				byte(v>>48) != 0,
				byte(v>>56) != 0,
			)
			offset += 8
			continue
		}
		b := byte(v)
		if b < 0x80 {
			dst = append(dst, b != 0)
			offset++
			continue
		}
		raw, readBytes, err := DecodeUvarint(data[offset:])
		if err != nil {
			return dst, err
		}
		dst = append(dst, raw != 0)
		offset += readBytes
	}
	for offset < n {
		b := data[offset]
		if b < 0x80 {
			dst = append(dst, b != 0)
			offset++
			continue
		}
		raw, readBytes, err := DecodeUvarint(data[offset:])
		if err != nil {
			return dst, err
		}
		dst = append(dst, raw != 0)
		offset += readBytes
	}
	return dst, nil
}

// DecodePackedVarints decodes packed uint64 varints into dst with zero heap allocations.
func DecodePackedVarints(data []byte, dst []uint64) ([]uint64, error) {
	dst = dst[:0]
	n := len(data)
	offset := 0
	for offset+8 <= n {
		v := binary.LittleEndian.Uint64(data[offset:])
		if (v & 0x8080808080808080) == 0 {
			dst = append(dst,
				uint64(byte(v)),
				uint64(byte(v>>8)),
				uint64(byte(v>>16)),
				uint64(byte(v>>24)),
				uint64(byte(v>>32)),
				uint64(byte(v>>40)),
				uint64(byte(v>>48)),
				uint64(byte(v>>56)),
			)
			offset += 8
			continue
		}
		b := byte(v)
		if b < 0x80 {
			dst = append(dst, uint64(b))
			offset++
			continue
		}
		val, readBytes, err := DecodeUvarint(data[offset:])
		if err != nil {
			return dst, err
		}
		dst = append(dst, val)
		offset += readBytes
	}
	for offset < n {
		b := data[offset]
		if b < 0x80 {
			dst = append(dst, uint64(b))
			offset++
			continue
		}
		val, readBytes, err := DecodeUvarint(data[offset:])
		if err != nil {
			return dst, err
		}
		dst = append(dst, val)
		offset += readBytes
	}
	return dst, nil
}

// DecodePackedSint32 decodes packed ZigZag-encoded sint32 values into dst.
func DecodePackedSint32(data []byte, dst []int32) ([]int32, error) {
	dst = dst[:0]
	n := len(data)
	offset := 0
	for offset+8 <= n {
		v := binary.LittleEndian.Uint64(data[offset:])
		if (v & 0x8080808080808080) == 0 {
			dst = append(dst,
				DecodeZigZag32(uint32(byte(v))),
				DecodeZigZag32(uint32(byte(v>>8))),
				DecodeZigZag32(uint32(byte(v>>16))),
				DecodeZigZag32(uint32(byte(v>>24))),
				DecodeZigZag32(uint32(byte(v>>32))),
				DecodeZigZag32(uint32(byte(v>>40))),
				DecodeZigZag32(uint32(byte(v>>48))),
				DecodeZigZag32(uint32(byte(v>>56))),
			)
			offset += 8
			continue
		}
		b := byte(v)
		if b < 0x80 {
			dst = append(dst, DecodeZigZag32(uint32(b)))
			offset++
			continue
		}
		raw, readBytes, err := DecodeUvarint(data[offset:])
		if err != nil {
			return dst, err
		}
		dst = append(dst, DecodeZigZag32(uint32(raw)))
		offset += readBytes
	}
	for offset < n {
		b := data[offset]
		if b < 0x80 {
			dst = append(dst, DecodeZigZag32(uint32(b)))
			offset++
			continue
		}
		raw, readBytes, err := DecodeUvarint(data[offset:])
		if err != nil {
			return dst, err
		}
		dst = append(dst, DecodeZigZag32(uint32(raw)))
		offset += readBytes
	}
	return dst, nil
}

// DecodePackedSint64 decodes packed ZigZag-encoded sint64 values into dst.
func DecodePackedSint64(data []byte, dst []int64) ([]int64, error) {
	dst = dst[:0]
	n := len(data)
	offset := 0
	for offset+8 <= n {
		v := binary.LittleEndian.Uint64(data[offset:])
		if (v & 0x8080808080808080) == 0 {
			dst = append(dst,
				DecodeZigZag64(uint64(byte(v))),
				DecodeZigZag64(uint64(byte(v>>8))),
				DecodeZigZag64(uint64(byte(v>>16))),
				DecodeZigZag64(uint64(byte(v>>24))),
				DecodeZigZag64(uint64(byte(v>>32))),
				DecodeZigZag64(uint64(byte(v>>40))),
				DecodeZigZag64(uint64(byte(v>>48))),
				DecodeZigZag64(uint64(byte(v>>56))),
			)
			offset += 8
			continue
		}
		b := byte(v)
		if b < 0x80 {
			dst = append(dst, DecodeZigZag64(uint64(b)))
			offset++
			continue
		}
		raw, readBytes, err := DecodeUvarint(data[offset:])
		if err != nil {
			return dst, err
		}
		dst = append(dst, DecodeZigZag64(raw))
		offset += readBytes
	}
	for offset < n {
		b := data[offset]
		if b < 0x80 {
			dst = append(dst, DecodeZigZag64(uint64(b)))
			offset++
			continue
		}
		raw, readBytes, err := DecodeUvarint(data[offset:])
		if err != nil {
			return dst, err
		}
		dst = append(dst, DecodeZigZag64(raw))
		offset += readBytes
	}
	return dst, nil
}
