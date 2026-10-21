// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bin

import (
	"encoding/binary"
	"errors"
	"fmt"
	"iter"
	"reflect"
	"runtime"
	"sync"
	"unsafe"

	"github.com/lemon4ksan/foundation/silicon/memkit"
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

func unpackStructPointer(v any) (reflect.Type, unsafe.Pointer, error) {
	if v == nil {
		return nil, nil, errors.New("bin: target must be a pointer to struct, got <nil>")
	}
	rt := reflect.TypeOf(v)
	e := memkit.EfaceOf(&v)
	if rt.Kind() != reflect.Pointer || e.Data == nil {
		return nil, nil, fmt.Errorf("bin: target must be a pointer to struct, got %v", rt)
	}
	return rt, e.Data, nil
}

func getLayout(t reflect.Type) (*structLayout, error) {
	if t == nil || t.Kind() != reflect.Pointer || t.Elem().Kind() != reflect.Struct {
		return nil, fmt.Errorf("bin: target must be a pointer to struct, got %v", t)
	}

	elemType := t.Elem()
	if v, ok := layoutCache.Load(elemType); ok {
		return v.(*structLayout), nil
	}

	layout := &structLayout{}
	numFields := elemType.NumField()

	for i := range numFields {
		field := elemType.Field(i)
		if !field.IsExported() {
			continue
		}
		tag := field.Tag.Get("bin")
		if tag == "" {
			tag = field.Tag.Get("binkit")
		}
		if tag == "-" || tag == "skip" {
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

func unmarshalStruct(b []byte, dest any, order binary.ByteOrder) error {
	rt, ptr, err := unpackStructPointer(dest)
	if err != nil {
		return err
	}
	layout, err := getLayout(rt)
	if err != nil {
		return err
	}

	if len(b) < layout.wireTotal {
		return ErrBufferTooShort
	}

	wirePos := 0

	for _, op := range layout.ops {
		targetPtr := unsafe.Add(ptr, op.structOffset)
		src := b[wirePos:]

		switch op.kind {
		case kindU8:
			*(*uint8)(targetPtr) = src[0]
		case kindU16:
			*(*uint16)(targetPtr) = order.Uint16(src)
		case kindU32:
			*(*uint32)(targetPtr) = order.Uint32(src)
		case kindU64:
			*(*uint64)(targetPtr) = order.Uint64(src)
		case kindI8:
			*(*int8)(targetPtr) = int8(src[0])
		case kindI16:
			*(*int16)(targetPtr) = int16(order.Uint16(src))
		case kindI32:
			*(*int32)(targetPtr) = int32(order.Uint32(src))
		case kindI64:
			*(*int64)(targetPtr) = int64(order.Uint64(src))
		case kindBytes:
			copy(unsafe.Slice((*byte)(targetPtr), op.wireSize), src[:op.wireSize])
		}

		wirePos += op.wireSize
	}

	runtime.KeepAlive(dest)
	return nil
}

func marshalStruct(src any, buf []byte, order binary.ByteOrder) ([]byte, error) {
	rt, ptr, err := unpackStructPointer(src)
	if err != nil {
		return nil, err
	}
	layout, err := getLayout(rt)
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
	wirePos := start

	for _, op := range layout.ops {
		targetPtr := unsafe.Add(ptr, op.structOffset)
		dst := buf[wirePos:]

		switch op.kind {
		case kindU8:
			dst[0] = *(*uint8)(targetPtr)
		case kindU16:
			order.PutUint16(dst, *(*uint16)(targetPtr))
		case kindU32:
			order.PutUint32(dst, *(*uint32)(targetPtr))
		case kindU64:
			order.PutUint64(dst, *(*uint64)(targetPtr))
		case kindI8:
			dst[0] = byte(*(*int8)(targetPtr))
		case kindI16:
			order.PutUint16(dst, uint16(*(*int16)(targetPtr)))
		case kindI32:
			order.PutUint32(dst, uint32(*(*int32)(targetPtr)))
		case kindI64:
			order.PutUint64(dst, uint64(*(*int64)(targetPtr)))
		case kindBytes:
			copy(dst[:op.wireSize], unsafe.Slice((*byte)(targetPtr), op.wireSize))
		}

		wirePos += op.wireSize
	}

	runtime.KeepAlive(src)
	return buf, nil
}

// UnmarshalLE automatically decodes little-endian binary fields from b into the target struct pointer.
func UnmarshalLE(b []byte, dest any) error {
	return unmarshalStruct(b, dest, binary.LittleEndian)
}

// MarshalLE automatically encodes the target struct pointer's fields into little-endian binary bytes.
func MarshalLE(src any, buf []byte) ([]byte, error) {
	return marshalStruct(src, buf, binary.LittleEndian)
}

// UnmarshalBE automatically decodes big-endian binary fields from b into the target struct pointer.
func UnmarshalBE(b []byte, dest any) error {
	return unmarshalStruct(b, dest, binary.BigEndian)
}

// MarshalBE automatically encodes the target struct pointer's fields into big-endian binary bytes.
func MarshalBE(src any, buf []byte) ([]byte, error) {
	return marshalStruct(src, buf, binary.BigEndian)
}

// UnmarshalSeqBE returns a push iterator decoding consecutive big-endian struct records of type T from data.
// Iteration stops when data is exhausted, when remaining bytes are insufficient, or when yield returns false.
func UnmarshalSeqBE[T any](data []byte) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		var zero T
		t := reflect.TypeOf(&zero)
		layout, err := getLayout(t)
		if err != nil {
			yield(zero, err)
			return
		}

		wireSize := layout.wireTotal
		if wireSize <= 0 {
			return
		}

		for len(data) >= wireSize {
			var val T
			if err := UnmarshalBE(data[:wireSize], &val); err != nil {
				yield(val, err)
				return
			}
			data = data[wireSize:]
			if !yield(val, nil) {
				return
			}
		}
	}
}

// UnmarshalSeqLE returns a push iterator decoding consecutive little-endian struct records of type T from data.
// Iteration stops when data is exhausted, when remaining bytes are insufficient, or when yield returns false.
func UnmarshalSeqLE[T any](data []byte) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		var zero T
		t := reflect.TypeOf(&zero)
		layout, err := getLayout(t)
		if err != nil {
			yield(zero, err)
			return
		}

		wireSize := layout.wireTotal
		if wireSize <= 0 {
			return
		}

		for len(data) >= wireSize {
			var val T
			if err := UnmarshalLE(data[:wireSize], &val); err != nil {
				yield(val, err)
				return
			}
			data = data[wireSize:]
			if !yield(val, nil) {
				return
			}
		}
	}
}
