// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto

import (
	"encoding/binary"
	"iter"
	"math"
)

// MapEntry represents a decoded key-value pair from a Protocol Buffers map entry submessage.
type MapEntry struct {
	Key   Field
	Value Field
}

// ScanMapEntry parses a single Protocol Buffers map entry submessage payload.
// In protobuf, field 1 is the key and field 2 is the value.
// It yields (key, val, err) without heap allocations and zero copies.
//
// If a key or value field is omitted in data, the returned Field has Num=1 or Num=2
// with default zero values.
func ScanMapEntry(data []byte) (key, val Field, err error) {
	key.Num = 1
	val.Num = 2

	offset := 0
	n := len(data)

	for offset < n {
		b0 := data[offset]
		var fieldNum uint32
		var wireType WireType
		if b0 < 0x80 {
			fieldNum = uint32(b0 >> 3)
			wireType = WireType(b0 & 7)
			offset++
		} else {
			tag, tagBytes, err := DecodeUvarint(data[offset:])
			if err != nil {
				return Field{}, Field{}, err
			}
			if tag > math.MaxUint32 {
				return Field{}, Field{}, ErrInvalidTag
			}
			offset += tagBytes
			fieldNum, wireType = SplitTag(uint32(tag))
		}
		if fieldNum == 0 || fieldNum > MaxFieldNumber {
			return Field{}, Field{}, ErrInvalidTag
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
				valUvar, valBytes := binary.Uvarint(data[offset:])
				if valBytes <= 0 {
					return Field{}, Field{}, ErrTruncated
				}
				offset += valBytes
				f.Raw = valUvar
			}

		case WireFixed64:
			if offset+8 > n {
				return Field{}, Field{}, ErrTruncated
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
				length, lenBytes = binary.Uvarint(data[offset:])
				if lenBytes <= 0 {
					return Field{}, Field{}, ErrTruncated
				}
				offset += lenBytes
			}
			if length > math.MaxInt || uint64(offset)+length > uint64(n) {
				return Field{}, Field{}, ErrTruncated
			}
			f.Data = data[offset : offset+int(length)]
			offset += int(length)

		case WireFixed32:
			if offset+4 > n {
				return Field{}, Field{}, ErrTruncated
			}
			f.Raw = uint64(binary.LittleEndian.Uint32(data[offset : offset+4]))
			offset += 4

		default:
			return Field{}, Field{}, ErrInvalidWireType
		}

		switch fieldNum {
		case 1:
			key = f
		case 2:
			val = f
		default:
			// Protobuf spec ignores unknown fields in map entries
		}
	}

	return key, val, nil
}

// ScanMapEntries returns a push iterator over all map entries for mapFieldNum in data.
// It yields (MapEntry, error) with zero heap allocations.
func ScanMapEntries(data []byte, mapFieldNum uint32) iter.Seq2[MapEntry, error] {
	return func(yield func(MapEntry, error) bool) {
		for f, err := range Scan(data) {
			if err != nil {
				yield(MapEntry{}, err)
				return
			}
			if f.Num == mapFieldNum && f.Type == WireBytes {
				k, v, entryErr := ScanMapEntry(f.Data)
				if entryErr != nil {
					yield(MapEntry{}, entryErr)
					return
				}
				if !yield(MapEntry{Key: k, Value: v}, nil) {
					return
				}
			}
		}
	}
}

// ScanMap is an alias for ScanMapEntries.
func ScanMap(data []byte, mapFieldNum uint32) iter.Seq2[MapEntry, error] {
	return ScanMapEntries(data, mapFieldNum)
}

// ForEachMapEntry iterates over all map entries for mapFieldNum in data.
func ForEachMapEntry(data []byte, mapFieldNum uint32, fn func(key, val Field) error) error {
	for entry, err := range ScanMapEntries(data, mapFieldNum) {
		if err != nil {
			return err
		}
		if err := fn(entry.Key, entry.Value); err != nil {
			return err
		}
	}
	return nil
}

// MapEntry parses f.Data as a map entry submessage, returning (key, val, err).
// Returns ErrInvalidWireType if f.Type != WireBytes.
func (f Field) MapEntry() (key, val Field, err error) {
	if f.Type != WireBytes {
		return Field{}, Field{}, ErrInvalidWireType
	}
	return ScanMapEntry(f.Data)
}

// AppendMapEntry appends a map entry submessage (field 1 = key, field 2 = val) under mapFieldNum to dst.
func AppendMapEntry(dst []byte, mapFieldNum uint32, key, val Field) []byte {
	key.Num = 1
	val.Num = 2

	entrySize := FieldSize(key) + FieldSize(val)
	dst = AppendTag(dst, mapFieldNum, WireBytes)
	dst = binary.AppendUvarint(dst, uint64(entrySize))
	dst = AppendField(dst, key)
	dst = AppendField(dst, val)
	return dst
}

// AppendMapEntryPayload encodes only the map entry payload (field 1 = key, field 2 = val) without outer tag/length.
func AppendMapEntryPayload(dst []byte, key, val Field) []byte {
	key.Num = 1
	val.Num = 2
	dst = AppendField(dst, key)
	dst = AppendField(dst, val)
	return dst
}

// AppendMapEntryStringString appends a map entry with string key and string value to dst.
func AppendMapEntryStringString(dst []byte, mapFieldNum uint32, key, val string) []byte {
	kLen := len(key)
	vLen := len(val)
	entryLen := 1 + VarintLen(uint64(kLen)) + kLen + 1 + VarintLen(uint64(vLen)) + vLen

	dst = AppendTag(dst, mapFieldNum, WireBytes)
	dst = binary.AppendUvarint(dst, uint64(entryLen))
	dst = append(dst, 0x0a) // Tag(1, WireBytes)
	dst = binary.AppendUvarint(dst, uint64(kLen))
	dst = append(dst, key...)
	dst = append(dst, 0x12) // Tag(2, WireBytes)
	dst = binary.AppendUvarint(dst, uint64(vLen))
	dst = append(dst, val...)
	return dst
}

// AppendMapEntryStringBytes appends a map entry with string key and byte slice value to dst.
func AppendMapEntryStringBytes(dst []byte, mapFieldNum uint32, key string, val []byte) []byte {
	kLen := len(key)
	vLen := len(val)
	entryLen := 1 + VarintLen(uint64(kLen)) + kLen + 1 + VarintLen(uint64(vLen)) + vLen

	dst = AppendTag(dst, mapFieldNum, WireBytes)
	dst = binary.AppendUvarint(dst, uint64(entryLen))
	dst = append(dst, 0x0a) // Tag(1, WireBytes)
	dst = binary.AppendUvarint(dst, uint64(kLen))
	dst = append(dst, key...)
	dst = append(dst, 0x12) // Tag(2, WireBytes)
	dst = binary.AppendUvarint(dst, uint64(vLen))
	dst = append(dst, val...)
	return dst
}

// AppendMapEntryStringInt64 appends a map entry with string key and int64 value to dst.
func AppendMapEntryStringInt64(dst []byte, mapFieldNum uint32, key string, val int64) []byte {
	kLen := len(key)
	vLen := VarintLen(uint64(val))
	entryLen := 1 + VarintLen(uint64(kLen)) + kLen + 1 + vLen

	dst = AppendTag(dst, mapFieldNum, WireBytes)
	dst = binary.AppendUvarint(dst, uint64(entryLen))
	dst = append(dst, 0x0a) // Tag(1, WireBytes)
	dst = binary.AppendUvarint(dst, uint64(kLen))
	dst = append(dst, key...)
	dst = append(dst, 0x10) // Tag(2, WireVarint)
	dst = binary.AppendUvarint(dst, uint64(val))
	return dst
}

// AppendMapEntryStringUint64 appends a map entry with string key and uint64 value to dst.
func AppendMapEntryStringUint64(dst []byte, mapFieldNum uint32, key string, val uint64) []byte {
	return AppendMapEntryStringInt64(dst, mapFieldNum, key, int64(val))
}

// AppendMapEntryInt64String appends a map entry with int64 key and string value to dst.
func AppendMapEntryInt64String(dst []byte, mapFieldNum uint32, key int64, val string) []byte {
	kLen := VarintLen(uint64(key))
	vLen := len(val)
	entryLen := 1 + kLen + 1 + VarintLen(uint64(vLen)) + vLen

	dst = AppendTag(dst, mapFieldNum, WireBytes)
	dst = binary.AppendUvarint(dst, uint64(entryLen))
	dst = append(dst, 0x08) // Tag(1, WireVarint)
	dst = binary.AppendUvarint(dst, uint64(key))
	dst = append(dst, 0x12) // Tag(2, WireBytes)
	dst = binary.AppendUvarint(dst, uint64(vLen))
	dst = append(dst, val...)
	return dst
}

// AppendMapEntryVarintVarint appends a map entry with varint key and varint value to dst.
func AppendMapEntryVarintVarint(dst []byte, mapFieldNum uint32, key, val uint64) []byte {
	kLen := VarintLen(key)
	vLen := VarintLen(val)
	entryLen := 1 + kLen + 1 + vLen

	dst = AppendTag(dst, mapFieldNum, WireBytes)
	dst = binary.AppendUvarint(dst, uint64(entryLen))
	dst = append(dst, 0x08) // Tag(1, WireVarint)
	dst = binary.AppendUvarint(dst, key)
	dst = append(dst, 0x10) // Tag(2, WireVarint)
	dst = binary.AppendUvarint(dst, val)
	return dst
}

// AppendMapEntryInt64Int64 appends a map entry with int64 key and int64 value to dst.
func AppendMapEntryInt64Int64(dst []byte, mapFieldNum uint32, key, val int64) []byte {
	return AppendMapEntryVarintVarint(dst, mapFieldNum, uint64(key), uint64(val))
}
