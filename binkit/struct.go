// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package binkit

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"sync"
	"unsafe"
)

type fieldKind uint8

const (
	kindU8 fieldKind = iota + 1
	kindU16
	kindU32
	kindU64
	kindI8
	kindI16
	kindI32
	kindI64
	kindBytes
)

type fieldOp struct {
	structOffset uintptr
	wireSize     int
	kind         fieldKind
}

type structLayout struct {
	wireTotal int
	ops       []fieldOp
}

var layoutCache sync.Map // map[reflect.Type]*structLayout

func getLayout(t reflect.Type) (*structLayout, error) {
	if t.Kind() != reflect.Pointer || t.Elem().Kind() != reflect.Struct {
		return nil, fmt.Errorf("binkit: target must be a pointer to struct, got %v", t)
	}

	elemType := t.Elem()
	if v, ok := layoutCache.Load(elemType); ok {
		return v.(*structLayout), nil
	}

	layout := &structLayout{}
	numFields := elemType.NumField()

	for i := 0; i < numFields; i++ {
		field := elemType.Field(i)
		if !field.IsExported() {
			continue
		}
		if tag := field.Tag.Get("binkit"); tag == "-" || tag == "skip" {
			continue
		}

		op := fieldOp{structOffset: field.Offset}
		switch field.Type.Kind() {
		case reflect.Uint8:
			op.kind = kindU8
			op.wireSize = 1
		case reflect.Uint16:
			op.kind = kindU16
			op.wireSize = 2
		case reflect.Uint32:
			op.kind = kindU32
			op.wireSize = 4
		case reflect.Uint64:
			op.kind = kindU64
			op.wireSize = 8
		case reflect.Int8:
			op.kind = kindI8
			op.wireSize = 1
		case reflect.Int16:
			op.kind = kindI16
			op.wireSize = 2
		case reflect.Int32:
			op.kind = kindI32
			op.wireSize = 4
		case reflect.Int64:
			op.kind = kindI64
			op.wireSize = 8
		case reflect.Array:
			if field.Type.Elem().Kind() == reflect.Uint8 {
				op.kind = kindBytes
				op.wireSize = field.Type.Len()
			} else {
				continue
			}
		default:
			// Non-fixed fields (strings, slices, pointers) are skipped for automatic fixed-header unpacking
			continue
		}

		layout.ops = append(layout.ops, op)
		layout.wireTotal += op.wireSize
	}

	layoutCache.Store(elemType, layout)
	return layout, nil
}

// UnmarshalLE automatically decodes little-endian binary fields from b into the target struct pointer.
func UnmarshalLE(b []byte, dest any) error {
	val := reflect.ValueOf(dest)
	layout, err := getLayout(val.Type())
	if err != nil {
		return err
	}

	if len(b) < layout.wireTotal {
		return ErrBufferTooShort
	}

	ptr := val.UnsafePointer()
	wirePos := 0

	for _, op := range layout.ops {
		targetPtr := unsafe.Add(ptr, op.structOffset)
		src := b[wirePos:]

		switch op.kind {
		case kindU8:
			*(*uint8)(targetPtr) = src[0]
		case kindU16:
			*(*uint16)(targetPtr) = binary.LittleEndian.Uint16(src)
		case kindU32:
			*(*uint32)(targetPtr) = binary.LittleEndian.Uint32(src)
		case kindU64:
			*(*uint64)(targetPtr) = binary.LittleEndian.Uint64(src)
		case kindI8:
			*(*int8)(targetPtr) = int8(src[0])
		case kindI16:
			*(*int16)(targetPtr) = int16(binary.LittleEndian.Uint16(src))
		case kindI32:
			*(*int32)(targetPtr) = int32(binary.LittleEndian.Uint32(src))
		case kindI64:
			*(*int64)(targetPtr) = int64(binary.LittleEndian.Uint64(src))
		case kindBytes:
			copy(unsafe.Slice((*byte)(targetPtr), op.wireSize), src[:op.wireSize])
		}

		wirePos += op.wireSize
	}

	return nil
}

// MarshalLE automatically encodes the target struct pointer's fields into little-endian binary bytes.
func MarshalLE(src any, buf []byte) ([]byte, error) {
	val := reflect.ValueOf(src)
	layout, err := getLayout(val.Type())
	if err != nil {
		return nil, err
	}

	needed := layout.wireTotal
	if cap(buf)-len(buf) < needed {
		newBuf := make([]byte, len(buf), len(buf)+needed)
		copy(newBuf, buf)
		buf = newBuf
	}

	start := len(buf)
	buf = buf[:start+needed]
	ptr := val.UnsafePointer()
	wirePos := start

	for _, op := range layout.ops {
		targetPtr := unsafe.Add(ptr, op.structOffset)
		dst := buf[wirePos:]

		switch op.kind {
		case kindU8:
			dst[0] = *(*uint8)(targetPtr)
		case kindU16:
			binary.LittleEndian.PutUint16(dst, *(*uint16)(targetPtr))
		case kindU32:
			binary.LittleEndian.PutUint32(dst, *(*uint32)(targetPtr))
		case kindU64:
			binary.LittleEndian.PutUint64(dst, *(*uint64)(targetPtr))
		case kindI8:
			dst[0] = byte(*(*int8)(targetPtr))
		case kindI16:
			binary.LittleEndian.PutUint16(dst, uint16(*(*int16)(targetPtr)))
		case kindI32:
			binary.LittleEndian.PutUint32(dst, uint32(*(*int32)(targetPtr)))
		case kindI64:
			binary.LittleEndian.PutUint64(dst, uint64(*(*int64)(targetPtr)))
		case kindBytes:
			copy(dst[:op.wireSize], unsafe.Slice((*byte)(targetPtr), op.wireSize))
		}

		wirePos += op.wireSize
	}

	return buf, nil
}

// UnmarshalBE automatically decodes big-endian binary fields from b into the target struct pointer.
func UnmarshalBE(b []byte, dest any) error {
	val := reflect.ValueOf(dest)
	layout, err := getLayout(val.Type())
	if err != nil {
		return err
	}

	if len(b) < layout.wireTotal {
		return ErrBufferTooShort
	}

	ptr := val.UnsafePointer()
	wirePos := 0

	for _, op := range layout.ops {
		targetPtr := unsafe.Add(ptr, op.structOffset)
		src := b[wirePos:]

		switch op.kind {
		case kindU8:
			*(*uint8)(targetPtr) = src[0]
		case kindU16:
			*(*uint16)(targetPtr) = binary.BigEndian.Uint16(src)
		case kindU32:
			*(*uint32)(targetPtr) = binary.BigEndian.Uint32(src)
		case kindU64:
			*(*uint64)(targetPtr) = binary.BigEndian.Uint64(src)
		case kindI8:
			*(*int8)(targetPtr) = int8(src[0])
		case kindI16:
			*(*int16)(targetPtr) = int16(binary.BigEndian.Uint16(src))
		case kindI32:
			*(*int32)(targetPtr) = int32(binary.BigEndian.Uint32(src))
		case kindI64:
			*(*int64)(targetPtr) = int64(binary.BigEndian.Uint64(src))
		case kindBytes:
			copy(unsafe.Slice((*byte)(targetPtr), op.wireSize), src[:op.wireSize])
		}

		wirePos += op.wireSize
	}

	return nil
}

// MarshalBE automatically encodes the target struct pointer's fields into big-endian binary bytes.
func MarshalBE(src any, buf []byte) ([]byte, error) {
	val := reflect.ValueOf(src)
	layout, err := getLayout(val.Type())
	if err != nil {
		return nil, err
	}

	needed := layout.wireTotal
	if cap(buf)-len(buf) < needed {
		newBuf := make([]byte, len(buf), len(buf)+needed)
		copy(newBuf, buf)
		buf = newBuf
	}

	start := len(buf)
	buf = buf[:start+needed]
	ptr := val.UnsafePointer()
	wirePos := start

	for _, op := range layout.ops {
		targetPtr := unsafe.Add(ptr, op.structOffset)
		dst := buf[wirePos:]

		switch op.kind {
		case kindU8:
			dst[0] = *(*uint8)(targetPtr)
		case kindU16:
			binary.BigEndian.PutUint16(dst, *(*uint16)(targetPtr))
		case kindU32:
			binary.BigEndian.PutUint32(dst, *(*uint32)(targetPtr))
		case kindU64:
			binary.BigEndian.PutUint64(dst, *(*uint64)(targetPtr))
		case kindI8:
			dst[0] = byte(*(*int8)(targetPtr))
		case kindI16:
			binary.BigEndian.PutUint16(dst, uint16(*(*int16)(targetPtr)))
		case kindI32:
			binary.BigEndian.PutUint32(dst, uint32(*(*int32)(targetPtr)))
		case kindI64:
			binary.BigEndian.PutUint64(dst, uint64(*(*int64)(targetPtr)))
		case kindBytes:
			copy(dst[:op.wireSize], unsafe.Slice((*byte)(targetPtr), op.wireSize))
		}

		wirePos += op.wireSize
	}

	return buf, nil
}
