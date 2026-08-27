// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json

import (
	"encoding"
	"encoding/base64"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"unsafe"

	"github.com/lemon4ksan/foundation/silicon/bytesconv"
)

var (
	unmarshalerType     = reflect.TypeOf((*Unmarshaler)(nil)).Elem()
	textUnmarshalerType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
)

// DecoderConfig specifies decoding flags and options.
type DecoderConfig struct {
	DisallowUnknownFields bool
	UseNumber             bool
	NoCopy                bool
}

type decodeFunc func(data []byte, cursor int, p unsafe.Pointer, cfg *DecoderConfig) (int, error)

var decoderCache sync.Map // map[reflect.Type]decodeFunc

func getDecoder(t reflect.Type) (decodeFunc, error) {
	if dec, ok := decoderCache.Load(t); ok {
		return dec.(decodeFunc), nil
	}

	dec, err := compileDecoder(t)
	if err != nil {
		return nil, err
	}

	decoderCache.Store(t, dec)

	return dec, nil
}

func compileDecoder(t reflect.Type) (decodeFunc, error) {
	if reflect.PointerTo(t).Implements(unmarshalerType) {
		return decodeUnmarshalerAddr(t), nil
	}

	if t.Implements(unmarshalerType) {
		return decodeUnmarshaler(t), nil
	}

	if reflect.PointerTo(t).Implements(textUnmarshalerType) {
		return decodeTextUnmarshalerAddr(t), nil
	}

	if t.Implements(textUnmarshalerType) {
		return decodeTextUnmarshaler(t), nil
	}

	if t == rawMessageType {
		return decodeRawMessage, nil
	}

	switch t.Kind() {
	case reflect.Bool:
		return decodeBool, nil
	case reflect.Int:
		return decodeInt, nil
	case reflect.Int8:
		return decodeInt8, nil
	case reflect.Int16:
		return decodeInt16, nil
	case reflect.Int32:
		return decodeInt32, nil
	case reflect.Int64:
		return decodeInt64, nil
	case reflect.Uint:
		return decodeUint, nil
	case reflect.Uint8:
		return decodeUint8, nil
	case reflect.Uint16:
		return decodeUint16, nil
	case reflect.Uint32:
		return decodeUint32, nil
	case reflect.Uint64:
		return decodeUint64, nil
	case reflect.Float32:
		return decodeFloat32, nil
	case reflect.Float64:
		return decodeFloat64, nil
	case reflect.String:
		return decodeString, nil
	case reflect.Pointer:
		return compilePtrDecoder(t)
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			return decodeByteSlice, nil
		}

		return compileSliceDecoder(t)
	case reflect.Array:
		return compileArrayDecoder(t)
	case reflect.Struct:
		return compileStructDecoder(t)
	case reflect.Map:
		return compileMapDecoder(t)
	case reflect.Interface:
		return decodeInterface, nil
	default:
		return nil, fmt.Errorf("json: unsupported decode type: %s", t.String())
	}
}

func decodeBool(data []byte, cursor int, p unsafe.Pointer, cfg *DecoderConfig) (int, error) {
	cursor = skipWhitespace(data, cursor)
	if cursor >= len(data) {
		return cursor, errUnexpectedEnd
	}

	if cursor+4 <= len(data) && string(data[cursor:cursor+4]) == "true" {
		*(*bool)(p) = true
		return cursor + 4, nil
	}

	if cursor+5 <= len(data) && string(data[cursor:cursor+5]) == "false" {
		*(*bool)(p) = false
		return cursor + 5, nil
	}

	if cursor+4 <= len(data) && string(data[cursor:cursor+4]) == "null" {
		return cursor + 4, nil
	}

	return cursor, errInvalidChar
}

func decodeInt(data []byte, cursor int, p unsafe.Pointer, cfg *DecoderConfig) (int, error) {
	cursor = skipWhitespace(data, cursor)
	num, newCursor, _, err := scanNumber(data, cursor)
	if err != nil {
		return cursor, err
	}

	n, err := parseInt(num)
	if err != nil {
		return cursor, err
	}

	*(*int)(p) = int(n)

	return newCursor, nil
}

func decodeInt8(data []byte, cursor int, p unsafe.Pointer, cfg *DecoderConfig) (int, error) {
	cursor = skipWhitespace(data, cursor)
	num, newCursor, _, err := scanNumber(data, cursor)
	if err != nil {
		return cursor, err
	}

	n, err := parseInt(num)
	if err != nil {
		return cursor, err
	}

	*(*int8)(p) = int8(n)

	return newCursor, nil
}

func decodeInt16(data []byte, cursor int, p unsafe.Pointer, cfg *DecoderConfig) (int, error) {
	cursor = skipWhitespace(data, cursor)
	num, newCursor, _, err := scanNumber(data, cursor)
	if err != nil {
		return cursor, err
	}

	n, err := parseInt(num)
	if err != nil {
		return cursor, err
	}

	*(*int16)(p) = int16(n)

	return newCursor, nil
}

func decodeInt32(data []byte, cursor int, p unsafe.Pointer, cfg *DecoderConfig) (int, error) {
	cursor = skipWhitespace(data, cursor)
	num, newCursor, _, err := scanNumber(data, cursor)
	if err != nil {
		return cursor, err
	}

	n, err := parseInt(num)
	if err != nil {
		return cursor, err
	}

	*(*int32)(p) = int32(n)

	return newCursor, nil
}

func decodeInt64(data []byte, cursor int, p unsafe.Pointer, cfg *DecoderConfig) (int, error) {
	cursor = skipWhitespace(data, cursor)
	num, newCursor, _, err := scanNumber(data, cursor)
	if err != nil {
		return cursor, err
	}

	n, err := parseInt(num)
	if err != nil {
		return cursor, err
	}

	*(*int64)(p) = n

	return newCursor, nil
}

func decodeUint(data []byte, cursor int, p unsafe.Pointer, cfg *DecoderConfig) (int, error) {
	cursor = skipWhitespace(data, cursor)
	num, newCursor, _, err := scanNumber(data, cursor)
	if err != nil {
		return cursor, err
	}

	n, err := parseUint(num)
	if err != nil {
		return cursor, err
	}

	*(*uint)(p) = uint(n)

	return newCursor, nil
}

func decodeUint8(data []byte, cursor int, p unsafe.Pointer, cfg *DecoderConfig) (int, error) {
	cursor = skipWhitespace(data, cursor)
	num, newCursor, _, err := scanNumber(data, cursor)
	if err != nil {
		return cursor, err
	}

	n, err := parseUint(num)
	if err != nil {
		return cursor, err
	}

	*(*uint8)(p) = uint8(n)

	return newCursor, nil
}

func decodeUint16(data []byte, cursor int, p unsafe.Pointer, cfg *DecoderConfig) (int, error) {
	cursor = skipWhitespace(data, cursor)
	num, newCursor, _, err := scanNumber(data, cursor)
	if err != nil {
		return cursor, err
	}

	n, err := parseUint(num)
	if err != nil {
		return cursor, err
	}

	*(*uint16)(p) = uint16(n)

	return newCursor, nil
}

func decodeUint32(data []byte, cursor int, p unsafe.Pointer, cfg *DecoderConfig) (int, error) {
	cursor = skipWhitespace(data, cursor)
	num, newCursor, _, err := scanNumber(data, cursor)
	if err != nil {
		return cursor, err
	}

	n, err := parseUint(num)
	if err != nil {
		return cursor, err
	}

	*(*uint32)(p) = uint32(n)

	return newCursor, nil
}

func decodeUint64(data []byte, cursor int, p unsafe.Pointer, cfg *DecoderConfig) (int, error) {
	cursor = skipWhitespace(data, cursor)
	num, newCursor, _, err := scanNumber(data, cursor)
	if err != nil {
		return cursor, err
	}

	n, err := parseUint(num)
	if err != nil {
		return cursor, err
	}

	*(*uint64)(p) = n

	return newCursor, nil
}

func decodeFloat32(data []byte, cursor int, p unsafe.Pointer, cfg *DecoderConfig) (int, error) {
	cursor = skipWhitespace(data, cursor)
	num, newCursor, _, err := scanNumber(data, cursor)
	if err != nil {
		return cursor, err
	}

	f, err := parseFloat(num)
	if err != nil {
		return cursor, err
	}

	*(*float32)(p) = float32(f)

	return newCursor, nil
}

func decodeFloat64(data []byte, cursor int, p unsafe.Pointer, cfg *DecoderConfig) (int, error) {
	cursor = skipWhitespace(data, cursor)
	num, newCursor, _, err := scanNumber(data, cursor)
	if err != nil {
		return cursor, err
	}

	f, err := parseFloat(num)
	if err != nil {
		return cursor, err
	}

	*(*float64)(p) = f

	return newCursor, nil
}

func decodeString(data []byte, cursor int, p unsafe.Pointer, cfg *DecoderConfig) (int, error) {
	cursor = skipWhitespace(data, cursor)
	if cursor >= len(data) {
		return cursor, errUnexpectedEnd
	}

	if cursor+4 <= len(data) && string(data[cursor:cursor+4]) == "null" {
		*(*string)(p) = ""
		return cursor + 4, nil
	}

	raw, newCursor, hasEscape, err := scanString(data, cursor)
	if err != nil {
		return cursor, err
	}

	if !hasEscape {
		if cfg != nil && cfg.NoCopy {
			*(*string)(p) = bytesconv.B2S(raw)
		} else {
			*(*string)(p) = string(raw)
		}

		return newCursor, nil
	}

	unescaped, err := unescapeString(make([]byte, 0, len(raw)), raw)
	if err != nil {
		return cursor, err
	}

	*(*string)(p) = string(unescaped)

	return newCursor, nil
}

func decodeByteSlice(data []byte, cursor int, p unsafe.Pointer, cfg *DecoderConfig) (int, error) {
	cursor = skipWhitespace(data, cursor)
	if cursor >= len(data) {
		return cursor, errUnexpectedEnd
	}

	if cursor+4 <= len(data) && string(data[cursor:cursor+4]) == "null" {
		*(*[]byte)(p) = nil
		return cursor + 4, nil
	}

	raw, newCursor, _, err := scanString(data, cursor)
	if err != nil {
		return cursor, err
	}

	decLen := base64.StdEncoding.DecodedLen(len(raw))
	buf := make([]byte, decLen)
	n, err := base64.StdEncoding.Decode(buf, raw)
	if err != nil {
		return cursor, err
	}

	*(*[]byte)(p) = buf[:n]

	return newCursor, nil
}

func decodeRawMessage(data []byte, cursor int, p unsafe.Pointer, cfg *DecoderConfig) (int, error) {
	cursor = skipWhitespace(data, cursor)
	start := cursor
	newCursor, err := skipValue(data, cursor)
	if err != nil {
		return cursor, err
	}

	msg := make(RawMessage, newCursor-start)
	copy(msg, data[start:newCursor])
	val := reflect.NewAt(rawMessageType, p).Elem()
	val.Set(reflect.ValueOf(msg))

	return newCursor, nil
}

func decodeUnmarshaler(t reflect.Type) decodeFunc {
	return func(data []byte, cursor int, p unsafe.Pointer, cfg *DecoderConfig) (int, error) {
		cursor = skipWhitespace(data, cursor)
		start := cursor
		newCursor, err := skipValue(data, cursor)
		if err != nil {
			return cursor, err
		}

		val := reflect.NewAt(t, p).Elem()
		if val.Kind() == reflect.Pointer && val.IsNil() {
			val.Set(reflect.New(t.Elem()))
		}
		u, ok := val.Interface().(Unmarshaler)
		if !ok {
			return cursor, errors.New("json: type does not implement Unmarshaler")
		}

		err = u.UnmarshalJSON(data[start:newCursor])
		if err != nil {
			return cursor, err
		}

		return newCursor, nil
	}
}

func decodeUnmarshalerAddr(t reflect.Type) decodeFunc {
	return func(data []byte, cursor int, p unsafe.Pointer, cfg *DecoderConfig) (int, error) {
		cursor = skipWhitespace(data, cursor)
		start := cursor
		newCursor, err := skipValue(data, cursor)
		if err != nil {
			return cursor, err
		}

		val := reflect.NewAt(t, p)
		u, ok := val.Interface().(Unmarshaler)
		if !ok {
			return cursor, errors.New("json: type pointer does not implement Unmarshaler")
		}

		err = u.UnmarshalJSON(data[start:newCursor])
		if err != nil {
			return cursor, err
		}

		return newCursor, nil
	}
}

func decodeTextUnmarshaler(t reflect.Type) decodeFunc {
	return func(data []byte, cursor int, p unsafe.Pointer, cfg *DecoderConfig) (int, error) {
		cursor = skipWhitespace(data, cursor)
		raw, newCursor, hasEscape, err := scanString(data, cursor)
		if err != nil {
			return cursor, err
		}

		valBytes := raw
		if hasEscape {
			valBytes, err = unescapeString(make([]byte, 0, len(raw)), raw)
			if err != nil {
				return cursor, err
			}
		}

		val := reflect.NewAt(t, p).Elem()
		u, ok := val.Interface().(encoding.TextUnmarshaler)
		if !ok {
			return cursor, errors.New("json: type does not implement TextUnmarshaler")
		}

		err = u.UnmarshalText(valBytes)
		if err != nil {
			return cursor, err
		}

		return newCursor, nil
	}
}

func decodeTextUnmarshalerAddr(t reflect.Type) decodeFunc {
	return func(data []byte, cursor int, p unsafe.Pointer, cfg *DecoderConfig) (int, error) {
		cursor = skipWhitespace(data, cursor)
		raw, newCursor, hasEscape, err := scanString(data, cursor)
		if err != nil {
			return cursor, err
		}

		valBytes := raw
		if hasEscape {
			valBytes, err = unescapeString(make([]byte, 0, len(raw)), raw)
			if err != nil {
				return cursor, err
			}
		}

		val := reflect.NewAt(t, p)
		u, ok := val.Interface().(encoding.TextUnmarshaler)
		if !ok {
			return cursor, errors.New("json: type pointer does not implement TextUnmarshaler")
		}

		err = u.UnmarshalText(valBytes)
		if err != nil {
			return cursor, err
		}

		return newCursor, nil
	}
}

type structFieldDecoder struct {
	name   string
	offset uintptr
	decode decodeFunc
	quoted bool
}

func compileStructFieldDecoders(t reflect.Type, baseOffset uintptr, fieldMap map[string]structFieldDecoder) error {
	numFields := t.NumField()
	for i := 0; i < numFields; i++ {
		sf := t.Field(i)
		if sf.PkgPath != "" && !sf.Anonymous {
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
				if err := compileStructFieldDecoders(ft, baseOffset+sf.Offset, fieldMap); err != nil {
					return err
				}
				continue
			}
		}

		dec, err := compileDecoder(sf.Type)
		if err != nil {
			return err
		}

		sfd := structFieldDecoder{
			name:   opts.name,
			offset: baseOffset + sf.Offset,
			decode: dec,
			quoted: opts.quoted,
		}

		if opts.name != "" {
			fieldMap[opts.name] = sfd
		}
		fieldMap[sf.Name] = sfd
		fieldMap[strings.ToLower(sf.Name)] = sfd
	}
	return nil
}

func compileStructDecoder(t reflect.Type) (decodeFunc, error) {
	fieldMap := make(map[string]structFieldDecoder)
	if err := compileStructFieldDecoders(t, 0, fieldMap); err != nil {
		return nil, err
	}

	return func(data []byte, cursor int, p unsafe.Pointer, cfg *DecoderConfig) (int, error) {
		cursor = skipWhitespace(data, cursor)
		if cursor >= len(data) {
			return cursor, errUnexpectedEnd
		}

		if cursor+4 <= len(data) && string(data[cursor:cursor+4]) == "null" {
			return cursor + 4, nil
		}

		if data[cursor] != '{' {
			return cursor, fmt.Errorf("json: expected '{', got %q", data[cursor])
		}
		cursor++

		first := true
		for {
			cursor = skipWhitespace(data, cursor)
			if cursor >= len(data) {
				return cursor, errUnexpectedEnd
			}

			if data[cursor] == '}' {
				return cursor + 1, nil
			}

			if !first {
				if cursor >= len(data) {
					return cursor, errUnexpectedEnd
				}
				if data[cursor] != ',' {
					return cursor, fmt.Errorf("json: expected ',', got %q", data[cursor])
				}
				cursor++
				cursor = skipWhitespace(data, cursor)
			}
			first = false

			if cursor >= len(data) {
				return cursor, errUnexpectedEnd
			}

			keyRaw, newCursor, hasEscape, err := scanString(data, cursor)
			if err != nil {
				return cursor, err
			}
			cursor = newCursor

			key := bytesconv.B2S(keyRaw)
			if hasEscape {
				unescaped, err := unescapeString(make([]byte, 0, len(keyRaw)), keyRaw)
				if err != nil {
					return cursor, err
				}
				key = string(unescaped)
			}

			cursor = skipWhitespace(data, cursor)
			if cursor >= len(data) {
				return cursor, errUnexpectedEnd
			}
			if data[cursor] != ':' {
				return cursor, fmt.Errorf("json: expected ':', got %q", data[cursor])
			}
			cursor++

			field, exists := fieldMap[key]
			if !exists {
				field, exists = fieldMap[strings.ToLower(key)]
			}
			if exists {
				fieldPtr := unsafe.Pointer(uintptr(p) + field.offset)
				if field.quoted {
					cursor = skipWhitespace(data, cursor)
					if cursor < len(data) && data[cursor] == '"' {
						strVal, strNewCursor, _, strErr := scanString(data, cursor)
						if strErr != nil {
							return cursor, strErr
						}
						if _, decErr := field.decode(strVal, 0, fieldPtr, cfg); decErr != nil {
							return cursor, decErr
						}
						cursor = strNewCursor
					} else {
						newCursor, err := field.decode(data, cursor, fieldPtr, cfg)
						if err != nil {
							return cursor, err
						}
						cursor = newCursor
					}
				} else {
					newCursor, err := field.decode(data, cursor, fieldPtr, cfg)
					if err != nil {
						return cursor, err
					}
					cursor = newCursor
				}
			} else {
				if cfg != nil && cfg.DisallowUnknownFields {
					return cursor, fmt.Errorf("json: unknown field %q", key)
				}

				newCursor, err := skipValue(data, cursor)
				if err != nil {
					return cursor, err
				}
				cursor = newCursor
			}
		}
	}, nil
}

func compilePtrDecoder(t reflect.Type) (decodeFunc, error) {
	elemType := t.Elem()
	elemDec, err := getDecoder(elemType)
	if err != nil {
		return nil, err
	}

	return func(data []byte, cursor int, p unsafe.Pointer, cfg *DecoderConfig) (int, error) {
		cursor = skipWhitespace(data, cursor)
		if cursor >= len(data) {
			return cursor, errUnexpectedEnd
		}

		if cursor+4 <= len(data) && string(data[cursor:cursor+4]) == "null" {
			*(*unsafe.Pointer)(p) = nil
			return cursor + 4, nil
		}

		elemPtr := *(*unsafe.Pointer)(p)
		if elemPtr == nil {
			val := reflect.New(elemType)
			elemPtr = unsafe.Pointer(val.Pointer())
			*(*unsafe.Pointer)(p) = elemPtr
		}

		return elemDec(data, cursor, elemPtr, cfg)
	}, nil
}

func compileSliceDecoder(t reflect.Type) (decodeFunc, error) {
	elemType := t.Elem()
	elemDec, err := getDecoder(elemType)
	if err != nil {
		return nil, err
	}

	return func(data []byte, cursor int, p unsafe.Pointer, cfg *DecoderConfig) (int, error) {
		cursor = skipWhitespace(data, cursor)
		if cursor >= len(data) {
			return cursor, errUnexpectedEnd
		}

		if cursor+4 <= len(data) && string(data[cursor:cursor+4]) == "null" {
			sliceVal := reflect.NewAt(t, p).Elem()
			sliceVal.Set(reflect.Zero(t))
			return cursor + 4, nil
		}

		if data[cursor] != '[' {
			return cursor, fmt.Errorf("json: expected '[', got %q", data[cursor])
		}
		cursor++

		sliceVal := reflect.NewAt(t, p).Elem()
		sliceVal.Set(reflect.MakeSlice(t, 0, 4))

		first := true
		for {
			cursor = skipWhitespace(data, cursor)
			if cursor >= len(data) {
				return cursor, errUnexpectedEnd
			}

			if data[cursor] == ']' {
				return cursor + 1, nil
			}

			if !first {
				if cursor >= len(data) {
					return cursor, errUnexpectedEnd
				}
				if data[cursor] != ',' {
					return cursor, fmt.Errorf("json: expected ',', got %q", data[cursor])
				}
				cursor++
			}
			first = false

			elemVal := reflect.New(elemType)
			elemPtr := unsafe.Pointer(elemVal.Pointer())

			newCursor, err := elemDec(data, cursor, elemPtr, cfg)
			if err != nil {
				return cursor, err
			}
			cursor = newCursor

			sliceVal.Set(reflect.Append(sliceVal, elemVal.Elem()))
		}
	}, nil
}

func compileArrayDecoder(t reflect.Type) (decodeFunc, error) {
	elemType := t.Elem()
	elemDec, err := getDecoder(elemType)
	if err != nil {
		return nil, err
	}
	arrayLen := t.Len()

	return func(data []byte, cursor int, p unsafe.Pointer, cfg *DecoderConfig) (int, error) {
		cursor = skipWhitespace(data, cursor)
		if cursor >= len(data) {
			return cursor, errUnexpectedEnd
		}

		if data[cursor] != '[' {
			return cursor, fmt.Errorf("json: expected '[', got %q", data[cursor])
		}
		cursor++

		arrVal := reflect.NewAt(t, p).Elem()
		idx := 0
		first := true
		for {
			cursor = skipWhitespace(data, cursor)
			if cursor >= len(data) {
				return cursor, errUnexpectedEnd
			}

			if data[cursor] == ']' {
				return cursor + 1, nil
			}

			if !first {
				if cursor >= len(data) {
					return cursor, errUnexpectedEnd
				}
				if data[cursor] != ',' {
					return cursor, fmt.Errorf("json: expected ',', got %q", data[cursor])
				}
				cursor++
			}
			first = false

			if idx < arrayLen {
				elemVal := arrVal.Index(idx)
				elemPtr := unsafe.Pointer(elemVal.Addr().Pointer())
				newCursor, err := elemDec(data, cursor, elemPtr, cfg)
				if err != nil {
					return cursor, err
				}
				cursor = newCursor
				idx++
			} else {
				newCursor, err := skipValue(data, cursor)
				if err != nil {
					return cursor, err
				}
				cursor = newCursor
			}
		}
	}, nil
}

func compileMapDecoder(t reflect.Type) (decodeFunc, error) {
	keyType := t.Key()
	elemType := t.Elem()

	var parseKey func(string) (reflect.Value, error)
	switch keyType.Kind() {
	case reflect.String:
		parseKey = func(s string) (reflect.Value, error) {
			return reflect.ValueOf(s).Convert(keyType), nil
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		parseKey = func(s string) (reflect.Value, error) {
			n, err := strconv.ParseInt(s, 10, keyType.Bits())
			if err != nil {
				return reflect.Value{}, err
			}
			v := reflect.New(keyType).Elem()
			v.SetInt(n)
			return v, nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		parseKey = func(s string) (reflect.Value, error) {
			n, err := strconv.ParseUint(s, 10, keyType.Bits())
			if err != nil {
				return reflect.Value{}, err
			}
			v := reflect.New(keyType).Elem()
			v.SetUint(n)
			return v, nil
		}
	default:
		return nil, fmt.Errorf("json: unsupported map key type: %s", keyType.String())
	}

	elemDec, err := getDecoder(elemType)
	if err != nil {
		return nil, err
	}

	return func(data []byte, cursor int, p unsafe.Pointer, cfg *DecoderConfig) (int, error) {
		cursor = skipWhitespace(data, cursor)
		if cursor >= len(data) {
			return cursor, errUnexpectedEnd
		}

		if cursor+4 <= len(data) && string(data[cursor:cursor+4]) == "null" {
			*(*unsafe.Pointer)(p) = nil
			return cursor + 4, nil
		}

		if data[cursor] != '{' {
			return cursor, fmt.Errorf("json: expected '{', got %q", data[cursor])
		}
		cursor++

		mapVal := reflect.NewAt(t, p).Elem()
		if mapVal.IsNil() {
			mapVal.Set(reflect.MakeMap(t))
		}

		first := true
		for {
			cursor = skipWhitespace(data, cursor)
			if cursor >= len(data) {
				return cursor, errUnexpectedEnd
			}

			if data[cursor] == '}' {
				return cursor + 1, nil
			}

			if !first {
				if cursor >= len(data) {
					return cursor, errUnexpectedEnd
				}
				if data[cursor] != ',' {
					return cursor, fmt.Errorf("json: expected ',', got %q", data[cursor])
				}
				cursor++
				cursor = skipWhitespace(data, cursor)
			}
			first = false

			if cursor >= len(data) {
				return cursor, errUnexpectedEnd
			}

			keyRaw, newCursor, hasEscape, err := scanString(data, cursor)
			if err != nil {
				return cursor, err
			}
			cursor = newCursor

			keyStr := string(keyRaw)
			if hasEscape {
				unescaped, err := unescapeString(make([]byte, 0, len(keyRaw)), keyRaw)
				if err != nil {
					return cursor, err
				}
				keyStr = string(unescaped)
			}

			cursor = skipWhitespace(data, cursor)
			if cursor >= len(data) {
				return cursor, errUnexpectedEnd
			}
			if data[cursor] != ':' {
				return cursor, fmt.Errorf("json: expected ':', got %q", data[cursor])
			}
			cursor++

			elemVal := reflect.New(elemType).Elem()
			elemPtr := unsafe.Pointer(elemVal.Addr().Pointer())

			newCursor, err = elemDec(data, cursor, elemPtr, cfg)
			if err != nil {
				return cursor, err
			}
			cursor = newCursor

			kv, err := parseKey(keyStr)
			if err != nil {
				return cursor, err
			}

			mapVal.SetMapIndex(kv, elemVal)
		}
	}, nil
}

func decodeInterface(data []byte, cursor int, p unsafe.Pointer, cfg *DecoderConfig) (int, error) {
	cursor = skipWhitespace(data, cursor)
	if cursor >= len(data) {
		return cursor, errUnexpectedEnd
	}

	existing := *(*any)(p)
	if existing != nil {
		rv := reflect.ValueOf(existing)
		if rv.Kind() == reflect.Pointer && !rv.IsNil() {
			elemType := rv.Type().Elem()
			dec, err := getDecoder(elemType)
			if err == nil {
				return dec(data, cursor, unsafe.Pointer(rv.Pointer()), cfg)
			}
		}
	}

	val, newCursor, err := decodeDynamic(data, cursor, cfg)
	if err != nil {
		return cursor, err
	}

	*(*any)(p) = val

	return newCursor, nil
}

func decodeDynamic(data []byte, cursor int, cfg *DecoderConfig) (any, int, error) {
	cursor = skipWhitespace(data, cursor)
	if cursor >= len(data) {
		return nil, cursor, errUnexpectedEnd
	}

	switch data[cursor] {
	case '"':
		raw, newCursor, hasEscape, err := scanString(data, cursor)
		if err != nil {
			return nil, cursor, err
		}
		if hasEscape {
			unescaped, err := unescapeString(make([]byte, 0, len(raw)), raw)
			if err != nil {
				return nil, cursor, err
			}
			return string(unescaped), newCursor, nil
		}
		return string(raw), newCursor, nil

	case '{':
		cursor++
		m := make(map[string]any)
		first := true
		for {
			cursor = skipWhitespace(data, cursor)
			if cursor >= len(data) {
				return nil, cursor, errUnexpectedEnd
			}

			if data[cursor] == '}' {
				return m, cursor + 1, nil
			}

			if !first {
				if data[cursor] != ',' {
					return nil, cursor, fmt.Errorf("json: expected ',', got %q", data[cursor])
				}
				cursor++
				cursor = skipWhitespace(data, cursor)
			}
			first = false

			keyRaw, newCursor, hasEscape, err := scanString(data, cursor)
			if err != nil {
				return nil, cursor, err
			}
			cursor = newCursor

			keyStr := string(keyRaw)
			if hasEscape {
				unescaped, err := unescapeString(make([]byte, 0, len(keyRaw)), keyRaw)
				if err != nil {
					return nil, cursor, err
				}
				keyStr = string(unescaped)
			}

			cursor = skipWhitespace(data, cursor)
			if cursor >= len(data) || data[cursor] != ':' {
				return nil, cursor, fmt.Errorf("json: expected ':', got %q", data[cursor])
			}
			cursor++

			v, valNewCursor, err := decodeDynamic(data, cursor, cfg)
			if err != nil {
				return nil, cursor, err
			}
			cursor = valNewCursor

			m[keyStr] = v
		}

	case '[':
		cursor++
		arr := make([]any, 0, 4)
		first := true
		for {
			cursor = skipWhitespace(data, cursor)
			if cursor >= len(data) {
				return nil, cursor, errUnexpectedEnd
			}

			if data[cursor] == ']' {
				return arr, cursor + 1, nil
			}

			if !first {
				if data[cursor] != ',' {
					return nil, cursor, fmt.Errorf("json: expected ',', got %q", data[cursor])
				}
				cursor++
			}
			first = false

			elem, newCursor, err := decodeDynamic(data, cursor, cfg)
			if err != nil {
				return nil, cursor, err
			}
			cursor = newCursor
			arr = append(arr, elem)
		}

	case 't':
		if cursor+4 <= len(data) && string(data[cursor:cursor+4]) == "true" {
			return true, cursor + 4, nil
		}
		return nil, cursor, errInvalidChar

	case 'f':
		if cursor+5 <= len(data) && string(data[cursor:cursor+5]) == "false" {
			return false, cursor + 5, nil
		}
		return nil, cursor, errInvalidChar

	case 'n':
		if cursor+4 <= len(data) && string(data[cursor:cursor+4]) == "null" {
			return nil, cursor + 4, nil
		}
		return nil, cursor, errInvalidChar

	default:
		numBytes, newCursor, isFloat, err := scanNumber(data, cursor)
		if err != nil {
			return nil, cursor, err
		}

		if cfg != nil && cfg.UseNumber {
			return Number(string(numBytes)), newCursor, nil
		}

		if isFloat {
			f, err := parseFloat(numBytes)
			return f, newCursor, err
		}

		n, err := parseInt(numBytes)
		if err == nil {
			return float64(n), newCursor, nil
		}

		f, err := parseFloat(numBytes)
		return f, newCursor, err
	}
}
