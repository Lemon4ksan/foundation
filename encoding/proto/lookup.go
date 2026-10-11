// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto

import (
	"encoding/binary"
	"math"

	"github.com/lemon4ksan/foundation/silicon/bytesconv"
)

// FindField searches data for the first occurrence of targetFieldNum without decoding the entire message.
// It returns (field, true, nil) if found, (Field{}, false, nil) if not found, or an error if data is malformed.
func FindField(data []byte, targetFieldNum uint32) (Field, bool, error) {
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
			tag, tagBytes := binary.Uvarint(data[offset:])
			if tagBytes <= 0 {
				return Field{}, false, ErrTruncated
			}
			offset += tagBytes
			fieldNum, wireType = SplitTag(uint32(tag))
		}
		if fieldNum == 0 {
			return Field{}, false, ErrInvalidTag
		}

		if fieldNum == targetFieldNum {
			var f Field
			f.Num = fieldNum
			f.Type = wireType

			switch wireType {
			case WireVarint:
				if offset < n && data[offset] < 0x80 {
					f.Raw = uint64(data[offset])
					return f, true, nil
				}
				val, valBytes := binary.Uvarint(data[offset:])
				if valBytes <= 0 {
					return Field{}, false, ErrTruncated
				}
				f.Raw = val
				return f, true, nil

			case WireFixed64:
				if offset+8 > n {
					return Field{}, false, ErrTruncated
				}
				f.Raw = binary.LittleEndian.Uint64(data[offset : offset+8])
				return f, true, nil

			case WireBytes:
				var length uint64
				if offset < n && data[offset] < 0x80 {
					length = uint64(data[offset])
					offset++
				} else {
					var lenBytes int
					length, lenBytes = binary.Uvarint(data[offset:])
					if lenBytes <= 0 {
						return Field{}, false, ErrTruncated
					}
					offset += lenBytes
				}
				rem := n - offset
				if length > uint64(rem) {
					return Field{}, false, ErrTruncated
				}
				end := offset + int(length)
				f.Data = data[offset:end]
				return f, true, nil

			case WireFixed32:
				if offset+4 > n {
					return Field{}, false, ErrTruncated
				}
				f.Raw = uint64(binary.LittleEndian.Uint32(data[offset : offset+4]))
				return f, true, nil

			default:
				return Field{}, false, ErrInvalidWireType
			}
		}

		// Fast skip for non-target fields
		switch wireType {
		case WireVarint:
			if offset < n && data[offset] < 0x80 {
				offset++
				continue
			}
			_, read := binary.Uvarint(data[offset:])
			if read <= 0 {
				return Field{}, false, ErrTruncated
			}
			offset += read

		case WireFixed64:
			if offset+8 > n {
				return Field{}, false, ErrTruncated
			}
			offset += 8

		case WireBytes:
			if offset < n && data[offset] < 0x80 {
				length := int(data[offset])
				offset++
				if offset+length > n {
					return Field{}, false, ErrTruncated
				}
				offset += length
				continue
			}
			length, read := binary.Uvarint(data[offset:])
			if read <= 0 {
				return Field{}, false, ErrTruncated
			}
			offset += read
			rem := n - offset
			if length > uint64(rem) {
				return Field{}, false, ErrTruncated
			}
			offset += int(length)

		case WireFixed32:
			if offset+4 > n {
				return Field{}, false, ErrTruncated
			}
			offset += 4

		default:
			skip, err := SkipField(data[offset:], wireType)
			if err != nil {
				return Field{}, false, err
			}
			offset += skip
		}
	}

	return Field{}, false, nil
}

// FindLastField searches data for targetFieldNum, returning the last occurrence in the message.
// This matches standard Protobuf merging behavior where the last occurrence wins.
func FindLastField(data []byte, targetFieldNum uint32) (Field, bool, error) {
	var (
		lastField Field
		found     bool
	)

	offset := 0
	n := len(data)

	for offset < n {
		tag, tagBytes := binary.Uvarint(data[offset:])
		if tagBytes <= 0 {
			return Field{}, false, ErrTruncated
		}
		offset += tagBytes

		fieldNum, wireType := SplitTag(uint32(tag))
		if fieldNum == 0 {
			return Field{}, false, ErrInvalidTag
		}

		if fieldNum == targetFieldNum {
			var f Field
			f.Num = fieldNum
			f.Type = wireType

			switch wireType {
			case WireVarint:
				val, valBytes := binary.Uvarint(data[offset:])
				if valBytes <= 0 {
					return Field{}, false, ErrTruncated
				}
				offset += valBytes
				f.Raw = val

			case WireFixed64:
				if offset+8 > n {
					return Field{}, false, ErrTruncated
				}
				f.Raw = binary.LittleEndian.Uint64(data[offset : offset+8])
				offset += 8

			case WireBytes:
				length, lenBytes := binary.Uvarint(data[offset:])
				if lenBytes <= 0 {
					return Field{}, false, ErrTruncated
				}
				offset += lenBytes
				rem := n - offset
				if length > uint64(rem) {
					return Field{}, false, ErrTruncated
				}
				end := offset + int(length)
				f.Data = data[offset:end]
				offset = end

			case WireFixed32:
				if offset+4 > n {
					return Field{}, false, ErrTruncated
				}
				f.Raw = uint64(binary.LittleEndian.Uint32(data[offset : offset+4]))
				offset += 4

			default:
				return Field{}, false, ErrInvalidWireType
			}

			lastField = f
			found = true
			continue
		}

		skip, err := SkipField(data[offset:], wireType)
		if err != nil {
			return Field{}, false, err
		}
		offset += skip
	}

	return lastField, found, nil
}

// FindVarint extracts the uint64 varint value of targetFieldNum.
// If the field is found but is not WireVarint, ErrInvalidWireType is returned.
func FindVarint(data []byte, targetFieldNum uint32) (uint64, bool, error) {
	f, found, err := FindField(data, targetFieldNum)
	if err != nil || !found {
		return 0, found, err
	}
	if f.Type != WireVarint {
		return 0, false, ErrInvalidWireType
	}
	return f.Raw, true, nil
}

// FindInt32 extracts the int32 varint value of targetFieldNum.
func FindInt32(data []byte, targetFieldNum uint32) (int32, bool, error) {
	v, found, err := FindVarint(data, targetFieldNum)
	return int32(v), found, err
}

// FindInt64 extracts the int64 varint value of targetFieldNum.
func FindInt64(data []byte, targetFieldNum uint32) (int64, bool, error) {
	v, found, err := FindVarint(data, targetFieldNum)
	return int64(v), found, err
}

// FindUint32 extracts the uint32 varint value of targetFieldNum.
func FindUint32(data []byte, targetFieldNum uint32) (uint32, bool, error) {
	v, found, err := FindVarint(data, targetFieldNum)
	return uint32(v), found, err
}

// FindUint64 extracts the uint64 varint value of targetFieldNum.
func FindUint64(data []byte, targetFieldNum uint32) (uint64, bool, error) {
	return FindVarint(data, targetFieldNum)
}

// FindSint32 extracts the ZigZag-encoded sint32 value of targetFieldNum.
func FindSint32(data []byte, targetFieldNum uint32) (int32, bool, error) {
	v, found, err := FindVarint(data, targetFieldNum)
	return DecodeZigZag32(uint32(v)), found, err
}

// FindSint64 extracts the ZigZag-encoded sint64 value of targetFieldNum.
func FindSint64(data []byte, targetFieldNum uint32) (int64, bool, error) {
	v, found, err := FindVarint(data, targetFieldNum)
	return DecodeZigZag64(v), found, err
}

// FindBool extracts the bool varint value of targetFieldNum.
func FindBool(data []byte, targetFieldNum uint32) (bool, bool, error) {
	v, found, err := FindVarint(data, targetFieldNum)
	return v != 0, found, err
}

// FindFixed32 extracts the uint32 fixed32 value of targetFieldNum.
func FindFixed32(data []byte, targetFieldNum uint32) (uint32, bool, error) {
	f, found, err := FindField(data, targetFieldNum)
	if err != nil || !found {
		return 0, found, err
	}
	if f.Type != WireFixed32 {
		return 0, false, ErrInvalidWireType
	}
	return uint32(f.Raw), true, nil
}

// FindFixed64 extracts the uint64 fixed64 value of targetFieldNum.
func FindFixed64(data []byte, targetFieldNum uint32) (uint64, bool, error) {
	f, found, err := FindField(data, targetFieldNum)
	if err != nil || !found {
		return 0, found, err
	}
	if f.Type != WireFixed64 {
		return 0, false, ErrInvalidWireType
	}
	return f.Raw, true, nil
}

// FindFloat32 extracts the float32 value of targetFieldNum.
func FindFloat32(data []byte, targetFieldNum uint32) (float32, bool, error) {
	f, found, err := FindField(data, targetFieldNum)
	if err != nil || !found {
		return 0, found, err
	}
	if f.Type != WireFixed32 {
		return 0, false, ErrInvalidWireType
	}
	return math.Float32frombits(uint32(f.Raw)), true, nil
}

// FindFloat64 extracts the float64 value of targetFieldNum.
func FindFloat64(data []byte, targetFieldNum uint32) (float64, bool, error) {
	f, found, err := FindField(data, targetFieldNum)
	if err != nil || !found {
		return 0, found, err
	}
	if f.Type != WireFixed64 {
		return 0, false, ErrInvalidWireType
	}
	return math.Float64frombits(f.Raw), true, nil
}

// FindBytes extracts the length-delimited byte slice of targetFieldNum without copying.
func FindBytes(data []byte, targetFieldNum uint32) ([]byte, bool, error) {
	f, found, err := FindField(data, targetFieldNum)
	if err != nil || !found {
		return nil, found, err
	}
	if f.Type != WireBytes {
		return nil, false, ErrInvalidWireType
	}
	return f.Data, true, nil
}

// FindString extracts the length-delimited string value of targetFieldNum without copying.
func FindString(data []byte, targetFieldNum uint32) (string, bool, error) {
	b, found, err := FindBytes(data, targetFieldNum)
	if err != nil || !found {
		return "", found, err
	}
	return bytesconv.B2S(b), true, nil
}

// FindMessage extracts the submessage payload bytes of targetFieldNum without copying.
func FindMessage(data []byte, targetFieldNum uint32) ([]byte, bool, error) {
	return FindBytes(data, targetFieldNum)
}

// FindNestedField traverses a path of submessage field numbers and returns the target field.
// E.g. FindNestedField(data, 1, 2, 3) finds field 1 at root, then field 2 in that submessage,
// then field 3 in that submessage.
func FindNestedField(data []byte, fieldPath ...uint32) (Field, bool, error) {
	if len(fieldPath) == 0 {
		return Field{}, false, nil
	}
	if len(fieldPath) > DefaultMaxDepth {
		return Field{}, false, ErrMaxDepthExceeded
	}

	curr := data
	for i, num := range fieldPath {
		f, found, err := FindField(curr, num)
		if err != nil || !found {
			return Field{}, found, err
		}
		if i == len(fieldPath)-1 {
			return f, true, nil
		}
		if f.Type != WireBytes {
			return Field{}, false, ErrInvalidWireType
		}
		curr = f.Data
	}
	return Field{}, false, nil
}

// FindNestedVarint traverses a path of submessages and returns the target uint64 varint value.
func FindNestedVarint(data []byte, fieldPath ...uint32) (uint64, bool, error) {
	f, found, err := FindNestedField(data, fieldPath...)
	if err != nil || !found {
		return 0, found, err
	}
	if f.Type != WireVarint {
		return 0, false, ErrInvalidWireType
	}
	return f.Raw, true, nil
}

// FindNestedBytes traverses a path of submessages and returns the target byte slice without copying.
func FindNestedBytes(data []byte, fieldPath ...uint32) ([]byte, bool, error) {
	f, found, err := FindNestedField(data, fieldPath...)
	if err != nil || !found {
		return nil, found, err
	}
	if f.Type != WireBytes {
		return nil, false, ErrInvalidWireType
	}
	return f.Data, true, nil
}

// FindNestedString traverses a path of submessages and returns the target string without copying.
func FindNestedString(data []byte, fieldPath ...uint32) (string, bool, error) {
	b, found, err := FindNestedBytes(data, fieldPath...)
	if err != nil || !found {
		return "", found, err
	}
	return bytesconv.B2S(b), true, nil
}

// FindNestedMessage traverses a path of submessages and returns the submessage payload bytes.
func FindNestedMessage(data []byte, fieldPath ...uint32) ([]byte, bool, error) {
	return FindNestedBytes(data, fieldPath...)
}
