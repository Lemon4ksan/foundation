// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json

import (
	"encoding"
	"encoding/base64"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"sync"
	"unsafe"

	"github.com/lemon4ksan/foundation/silicon/bytesconv"
)

var (
	marshalerType     = reflect.TypeOf((*Marshaler)(nil)).Elem()
	textMarshalerType = reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()
	rawMessageType    = reflect.TypeOf(RawMessage(nil))
)

type encodeState struct {
	buf        []byte
	escapeHTML bool
}

var encPool = sync.Pool{
	New: func() any {
		return &encodeState{
			buf:        make([]byte, 0, 512),
			escapeHTML: true,
		}
	},
}

type encoderFunc func(e *encodeState, p unsafe.Pointer) error

type encEntry struct {
	done chan struct{}
	enc  encoderFunc
	err  error
}

var (
	encoderCache sync.Map // map[reflect.Type]encoderFunc
	encMu        sync.Mutex
	encCompiling = make(map[reflect.Type]*encEntry)
)

func getEncoder(t reflect.Type) (encoderFunc, error) {
	if enc, ok := encoderCache.Load(t); ok {
		return enc.(encoderFunc), nil
	}

	encMu.Lock()
	if enc, ok := encoderCache.Load(t); ok {
		encMu.Unlock()
		return enc.(encoderFunc), nil
	}

	if entry, ok := encCompiling[t]; ok {
		encMu.Unlock()
		return func(e *encodeState, p unsafe.Pointer) error {
			<-entry.done
			if entry.err != nil {
				return entry.err
			}
			return entry.enc(e, p)
		}, nil
	}

	entry := &encEntry{
		done: make(chan struct{}),
	}
	encCompiling[t] = entry
	encMu.Unlock()

	enc, err := compileEncoder(t)

	encMu.Lock()
	entry.enc = enc
	entry.err = err
	delete(encCompiling, t)
	if err == nil {
		encoderCache.Store(t, enc)
	}
	close(entry.done)
	encMu.Unlock()

	if err != nil {
		return nil, err
	}

	return enc, nil
}

func compileEncoder(t reflect.Type) (encoderFunc, error) {
	if t == rawMessageType {
		return encodeRawMessage, nil
	}

	if t.Implements(marshalerType) {
		return compileMarshalerEncoder(t), nil
	}

	if reflect.PointerTo(t).Implements(marshalerType) && t.Kind() != reflect.Pointer {
		return compileAddrMarshalerEncoder(t), nil
	}

	if t.Implements(textMarshalerType) {
		return compileTextMarshalerEncoder(t), nil
	}

	if reflect.PointerTo(t).Implements(textMarshalerType) && t.Kind() != reflect.Pointer {
		return compileAddrTextMarshalerEncoder(t), nil
	}

	switch t.Kind() {
	case reflect.Bool:
		return encodeBool, nil
	case reflect.Int:
		return encodeInt, nil
	case reflect.Int8:
		return encodeInt8, nil
	case reflect.Int16:
		return encodeInt16, nil
	case reflect.Int32:
		return encodeInt32, nil
	case reflect.Int64:
		return encodeInt64, nil
	case reflect.Uint:
		return encodeUint, nil
	case reflect.Uint8:
		return encodeUint8, nil
	case reflect.Uint16:
		return encodeUint16, nil
	case reflect.Uint32:
		return encodeUint32, nil
	case reflect.Uint64:
		return encodeUint64, nil
	case reflect.Float32:
		return encodeFloat32, nil
	case reflect.Float64:
		return encodeFloat64, nil
	case reflect.String:
		return encodeString, nil
	case reflect.Pointer:
		return compilePtrEncoder(t)
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			return encodeByteSlice, nil
		}

		return compileSliceEncoder(t)
	case reflect.Array:
		return compileArrayEncoder(t)
	case reflect.Struct:
		return compileStructEncoder(t)
	case reflect.Map:
		return compileMapEncoder(t)
	case reflect.Interface:
		return encodeInterface, nil
	default:
		return nil, fmt.Errorf("json: unsupported type: %s", t.String())
	}
}

func encodeRawMessage(e *encodeState, p unsafe.Pointer) error {
	val := reflect.NewAt(rawMessageType, p).Elem()
	msg := val.Interface().(RawMessage)
	if msg == nil {
		e.buf = append(e.buf, "null"...)
		return nil
	}

	e.buf = append(e.buf, msg...)
	return nil
}

func compileMarshalerEncoder(t reflect.Type) encoderFunc {
	return func(e *encodeState, p unsafe.Pointer) error {
		val := reflect.NewAt(t, p).Elem()
		m := val.Interface().(Marshaler)
		data, err := m.MarshalJSON()
		if err != nil {
			return err
		}

		e.buf = append(e.buf, data...)
		return nil
	}
}

func compileAddrMarshalerEncoder(t reflect.Type) encoderFunc {
	return func(e *encodeState, p unsafe.Pointer) error {
		val := reflect.NewAt(t, p)
		m := val.Interface().(Marshaler)
		data, err := m.MarshalJSON()
		if err != nil {
			return err
		}

		e.buf = append(e.buf, data...)
		return nil
	}
}

func compileTextMarshalerEncoder(t reflect.Type) encoderFunc {
	return func(e *encodeState, p unsafe.Pointer) error {
		val := reflect.NewAt(t, p).Elem()
		m := val.Interface().(encoding.TextMarshaler)
		text, err := m.MarshalText()
		if err != nil {
			return err
		}

		e.buf = appendString(e.buf, bytesconv.B2S(text), e.escapeHTML)
		return nil
	}
}

func compileAddrTextMarshalerEncoder(t reflect.Type) encoderFunc {
	return func(e *encodeState, p unsafe.Pointer) error {
		val := reflect.NewAt(t, p)
		m := val.Interface().(encoding.TextMarshaler)
		text, err := m.MarshalText()
		if err != nil {
			return err
		}

		e.buf = appendString(e.buf, bytesconv.B2S(text), e.escapeHTML)
		return nil
	}
}

func encodeBool(e *encodeState, p unsafe.Pointer) error {
	if *(*bool)(p) {
		e.buf = append(e.buf, "true"...)
	} else {
		e.buf = append(e.buf, "false"...)
	}

	return nil
}

func encodeInt(e *encodeState, p unsafe.Pointer) error {
	e.buf = strconv.AppendInt(e.buf, int64(*(*int)(p)), 10)
	return nil
}

func encodeInt8(e *encodeState, p unsafe.Pointer) error {
	e.buf = strconv.AppendInt(e.buf, int64(*(*int8)(p)), 10)
	return nil
}

func encodeInt16(e *encodeState, p unsafe.Pointer) error {
	e.buf = strconv.AppendInt(e.buf, int64(*(*int16)(p)), 10)
	return nil
}

func encodeInt32(e *encodeState, p unsafe.Pointer) error {
	e.buf = strconv.AppendInt(e.buf, int64(*(*int32)(p)), 10)
	return nil
}

func encodeInt64(e *encodeState, p unsafe.Pointer) error {
	e.buf = strconv.AppendInt(e.buf, *(*int64)(p), 10)
	return nil
}

func encodeUint(e *encodeState, p unsafe.Pointer) error {
	e.buf = strconv.AppendUint(e.buf, uint64(*(*uint)(p)), 10)
	return nil
}

func encodeUint8(e *encodeState, p unsafe.Pointer) error {
	e.buf = strconv.AppendUint(e.buf, uint64(*(*uint8)(p)), 10)
	return nil
}

func encodeUint16(e *encodeState, p unsafe.Pointer) error {
	e.buf = strconv.AppendUint(e.buf, uint64(*(*uint16)(p)), 10)
	return nil
}

func encodeUint32(e *encodeState, p unsafe.Pointer) error {
	e.buf = strconv.AppendUint(e.buf, uint64(*(*uint32)(p)), 10)
	return nil
}

func encodeUint64(e *encodeState, p unsafe.Pointer) error {
	e.buf = strconv.AppendUint(e.buf, *(*uint64)(p), 10)
	return nil
}

func encodeFloat32(e *encodeState, p unsafe.Pointer) error {
	f := float64(*(*float32)(p))
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return fmt.Errorf("json: unsupported float value: %f", f)
	}

	e.buf = strconv.AppendFloat(e.buf, f, 'f', -1, 32)

	return nil
}

func encodeFloat64(e *encodeState, p unsafe.Pointer) error {
	f := *(*float64)(p)
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return fmt.Errorf("json: unsupported float value: %f", f)
	}

	e.buf = strconv.AppendFloat(e.buf, f, 'f', -1, 64)

	return nil
}

func encodeString(e *encodeState, p unsafe.Pointer) error {
	s := *(*string)(p)
	e.buf = appendString(e.buf, s, e.escapeHTML)

	return nil
}

func encodeByteSlice(e *encodeState, p unsafe.Pointer) error {
	s := *(*[]byte)(p)
	if s == nil {
		e.buf = append(e.buf, "null"...)
		return nil
	}

	e.buf = append(e.buf, '"')
	encLen := base64.StdEncoding.EncodedLen(len(s))
	start := len(e.buf)
	if cap(e.buf)-start < encLen {
		newBuf := make([]byte, start+encLen)
		copy(newBuf, e.buf)
		e.buf = newBuf
	} else {
		e.buf = e.buf[:start+encLen]
	}

	base64.StdEncoding.Encode(e.buf[start:], s)
	e.buf = append(e.buf, '"')

	return nil
}

type structFieldEncoder struct {
	nameBytes []byte
	offset    uintptr
	encode    encoderFunc
	omitEmpty bool
	quoted    bool
	isZero    func(p unsafe.Pointer) bool
}

func compileStructFields(t reflect.Type, baseOffset uintptr) ([]structFieldEncoder, error) {
	numFields := t.NumField()
	var fields []structFieldEncoder

	for i := 0; i < numFields; i++ {
		sf := t.Field(i)
		if sf.PkgPath != "" && !sf.Anonymous { // private unexported field
			continue
		}

		opts := parseTag(sf)
		if opts.ignored {
			continue
		}

		ft := sf.Type
		if sf.Anonymous && opts.name == "" {
			if ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct {
				nested, err := compileStructFields(ft, baseOffset+sf.Offset)
				if err != nil {
					return nil, err
				}
				fields = append(fields, nested...)
				continue
			}
		}

		enc, err := compileEncoder(sf.Type)
		if err != nil {
			return nil, err
		}

		sfe := structFieldEncoder{
			nameBytes: []byte(sf.Name),
			offset:    baseOffset + sf.Offset,
			encode:    enc,
			omitEmpty: opts.omitEmpty,
			quoted:    opts.quoted,
			isZero:    buildIsZero(sf.Type),
		}

		if opts.name != "" {
			sfe.nameBytes = []byte(opts.name)
		}

		fields = append(fields, sfe)
	}

	return fields, nil
}

func compileStructEncoder(t reflect.Type) (encoderFunc, error) {
	fields, err := compileStructFields(t, 0)
	if err != nil {
		return nil, err
	}

	return func(e *encodeState, p unsafe.Pointer) error {
		e.buf = append(e.buf, '{')
		first := true

		for i := range fields {
			f := &fields[i]
			fp := unsafe.Pointer(uintptr(p) + f.offset)

			if f.omitEmpty && f.isZero(fp) {
				continue
			}

			if !first {
				e.buf = append(e.buf, ',')
			}
			first = false

			e.buf = appendString(e.buf, bytesconv.B2S(f.nameBytes), e.escapeHTML)
			e.buf = append(e.buf, ':')

			if f.quoted {
				e.buf = append(e.buf, '"')
			}

			if err := f.encode(e, fp); err != nil {
				return err
			}

			if f.quoted {
				e.buf = append(e.buf, '"')
			}
		}

		e.buf = append(e.buf, '}')
		return nil
	}, nil
}

func buildIsZero(t reflect.Type) func(p unsafe.Pointer) bool {
	switch t.Kind() {
	case reflect.Bool:
		return func(p unsafe.Pointer) bool { return !*(*bool)(p) }
	case reflect.Int:
		return func(p unsafe.Pointer) bool { return *(*int)(p) == 0 }
	case reflect.Int8:
		return func(p unsafe.Pointer) bool { return *(*int8)(p) == 0 }
	case reflect.Int16:
		return func(p unsafe.Pointer) bool { return *(*int16)(p) == 0 }
	case reflect.Int32:
		return func(p unsafe.Pointer) bool { return *(*int32)(p) == 0 }
	case reflect.Int64:
		return func(p unsafe.Pointer) bool { return *(*int64)(p) == 0 }
	case reflect.Uint:
		return func(p unsafe.Pointer) bool { return *(*uint)(p) == 0 }
	case reflect.Uint8:
		return func(p unsafe.Pointer) bool { return *(*uint8)(p) == 0 }
	case reflect.Uint16:
		return func(p unsafe.Pointer) bool { return *(*uint16)(p) == 0 }
	case reflect.Uint32:
		return func(p unsafe.Pointer) bool { return *(*uint32)(p) == 0 }
	case reflect.Uint64:
		return func(p unsafe.Pointer) bool { return *(*uint64)(p) == 0 }
	case reflect.Float32:
		return func(p unsafe.Pointer) bool { return *(*float32)(p) == 0 }
	case reflect.Float64:
		return func(p unsafe.Pointer) bool { return *(*float64)(p) == 0 }
	case reflect.String:
		return func(p unsafe.Pointer) bool { return *(*string)(p) == "" }
	case reflect.Pointer, reflect.Slice, reflect.Map, reflect.Interface:
		return func(p unsafe.Pointer) bool { return *(*unsafe.Pointer)(p) == nil }
	default:
		return func(p unsafe.Pointer) bool { return false }
	}
}

func compilePtrEncoder(t reflect.Type) (encoderFunc, error) {
	elemType := t.Elem()
	var (
		once    sync.Once
		elemEnc encoderFunc
		elemErr error
	)

	return func(e *encodeState, p unsafe.Pointer) error {
		ptr := *(*unsafe.Pointer)(p)
		if ptr == nil {
			e.buf = append(e.buf, "null"...)
			return nil
		}

		once.Do(func() {
			elemEnc, elemErr = getEncoder(elemType)
		})
		if elemErr != nil {
			return elemErr
		}

		return elemEnc(e, ptr)
	}, nil
}

func compileSliceEncoder(t reflect.Type) (encoderFunc, error) {
	elemType := t.Elem()
	var (
		once    sync.Once
		elemEnc encoderFunc
		elemErr error
	)

	return func(e *encodeState, p unsafe.Pointer) error {
		sliceVal := reflect.NewAt(t, p).Elem()
		if sliceVal.IsNil() {
			e.buf = append(e.buf, "null"...)
			return nil
		}

		once.Do(func() {
			elemEnc, elemErr = getEncoder(elemType)
		})
		if elemErr != nil {
			return elemErr
		}

		e.buf = append(e.buf, '[')
		n := sliceVal.Len()
		for i := 0; i < n; i++ {
			if i > 0 {
				e.buf = append(e.buf, ',')
			}

			elemVal := sliceVal.Index(i)
			elemPtr := unsafe.Pointer(elemVal.Addr().Pointer())
			if err := elemEnc(e, elemPtr); err != nil {
				return err
			}
		}

		e.buf = append(e.buf, ']')
		return nil
	}, nil
}

func compileArrayEncoder(t reflect.Type) (encoderFunc, error) {
	elemType := t.Elem()
	elemEnc, err := getEncoder(elemType)
	if err != nil {
		return nil, err
	}

	return func(e *encodeState, p unsafe.Pointer) error {
		arrVal := reflect.NewAt(t, p).Elem()
		e.buf = append(e.buf, '[')
		n := arrVal.Len()
		for i := 0; i < n; i++ {
			if i > 0 {
				e.buf = append(e.buf, ',')
			}

			elemVal := arrVal.Index(i)
			elemPtr := unsafe.Pointer(elemVal.Addr().Pointer())
			if err := elemEnc(e, elemPtr); err != nil {
				return err
			}
		}

		e.buf = append(e.buf, ']')
		return nil
	}, nil
}

func compileMapEncoder(t reflect.Type) (encoderFunc, error) {
	return func(e *encodeState, p unsafe.Pointer) error {
		val := reflect.NewAt(t, p).Elem()
		if val.IsNil() {
			e.buf = append(e.buf, "null"...)
			return nil
		}

		e.buf = append(e.buf, '{')
		first := true

		iter := val.MapRange()
		for iter.Next() {
			if !first {
				e.buf = append(e.buf, ',')
			}
			first = false

			k := iter.Key()
			v := iter.Value()

			kStr := fmt.Sprint(k.Interface())
			e.buf = appendString(e.buf, kStr, e.escapeHTML)
			e.buf = append(e.buf, ':')

			vBytes, err := Marshal(v.Interface())
			if err != nil {
				return err
			}

			e.buf = append(e.buf, vBytes...)
		}

		e.buf = append(e.buf, '}')
		return nil
	}, nil
}

func encodeInterface(e *encodeState, p unsafe.Pointer) error {
	val := *(*any)(p)
	if val == nil {
		e.buf = append(e.buf, "null"...)
		return nil
	}

	bytes, err := Marshal(val)
	if err != nil {
		return err
	}

	e.buf = append(e.buf, bytes...)

	return nil
}

const hexDigits = "0123456789abcdef"

func appendString(dst []byte, s string, escapeHTML bool) []byte {
	dst = append(dst, '"')
	start := 0

	for i := 0; i < len(s); i++ {
		b := s[i]
		if b >= 0x20 && b != '"' && b != '\\' && (!escapeHTML || (b != '<' && b != '>' && b != '&')) {
			continue
		}

		if start < i {
			dst = append(dst, s[start:i]...)
		}

		switch b {
		case '"':
			dst = append(dst, `\"`...)
		case '\\':
			dst = append(dst, `\\`...)
		case '\n':
			dst = append(dst, `\n`...)
		case '\r':
			dst = append(dst, `\r`...)
		case '\t':
			dst = append(dst, `\t`...)
		case '\b':
			dst = append(dst, `\b`...)
		case '\f':
			dst = append(dst, `\f`...)
		case '<':
			dst = append(dst, `\u003c`...)
		case '>':
			dst = append(dst, `\u003e`...)
		case '&':
			dst = append(dst, `\u0026`...)
		default:
			// Control character \u00XX
			dst = append(dst, `\u00`...)
			dst = append(dst, hexDigits[b>>4], hexDigits[b&0x0F])
		}

		start = i + 1
	}

	if start < len(s) {
		dst = append(dst, s[start:]...)
	}

	dst = append(dst, '"')

	return dst
}
