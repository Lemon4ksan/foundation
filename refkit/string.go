// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package refkit

import (
	"encoding"
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"github.com/lemon4ksan/foundation/silicon/bytesconv"
)

// ErrUnsupportedType is returned when a type cannot be converted to a string representation.
var ErrUnsupportedType = errors.New("refkit: unsupported type")

// ToString converts v into its string representation.
// It uses fast non-reflective paths for primitive types, [encoding.TextMarshaler],
// and [fmt.Stringer], falling back to reflection for pointers and complex types.
func ToString(v any) (string, error) {
	if v == nil {
		return "", nil
	}

	switch val := v.(type) {
	case string:
		return val, nil
	case int:
		return strconv.FormatInt(int64(val), 10), nil
	case int64:
		return strconv.FormatInt(val, 10), nil
	case int32:
		return strconv.FormatInt(int64(val), 10), nil
	case int16:
		return strconv.FormatInt(int64(val), 10), nil
	case int8:
		return strconv.FormatInt(int64(val), 10), nil
	case uint:
		return strconv.FormatUint(uint64(val), 10), nil
	case uint64:
		return strconv.FormatUint(val, 10), nil
	case uint32:
		return strconv.FormatUint(uint64(val), 10), nil
	case uint16:
		return strconv.FormatUint(uint64(val), 10), nil
	case uint8:
		return strconv.FormatUint(uint64(val), 10), nil
	case bool:
		return strconv.FormatBool(val), nil
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64), nil
	case float32:
		return strconv.FormatFloat(float64(val), 'f', -1, 32), nil
	case []byte:
		return bytesconv.B2S(val), nil
	case encoding.TextMarshaler:
		b, err := val.MarshalText()
		if err != nil {
			return "", err
		}

		return bytesconv.B2S(b), nil
	case fmt.Stringer:
		return val.String(), nil
	default:
		return ValueToString(reflect.ValueOf(v))
	}
}

// ValueToString unwraps consecutive pointers and interfaces in v and formats
// primitive scalars, [encoding.TextMarshaler], and [fmt.Stringer] implementations into a string.
func ValueToString(v reflect.Value) (string, error) {
	for v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return "", nil
		}

		v = v.Elem()
	}

	if !v.IsValid() {
		return "", nil
	}

	switch v.Kind() {
	case reflect.String:
		return v.String(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10), nil
	case reflect.Bool:
		return strconv.FormatBool(v.Bool()), nil
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64), nil
	}

	if v.CanInterface() {
		val := v.Interface()

		if tm, ok := val.(encoding.TextMarshaler); ok {
			b, err := tm.MarshalText()
			if err != nil {
				return "", err
			}

			return bytesconv.B2S(b), nil
		}

		if s, ok := val.(fmt.Stringer); ok {
			return s.String(), nil
		}
	}

	return "", ErrUnsupportedType
}

// HasTextRepresentation reports whether v implements [encoding.TextMarshaler] or [fmt.Stringer].
func HasTextRepresentation(v any) bool {
	if v == nil {
		return false
	}

	if _, ok := v.(encoding.TextMarshaler); ok {
		return true
	}

	if _, ok := v.(fmt.Stringer); ok {
		return true
	}

	if rv, ok := v.(reflect.Value); ok {
		return ValueHasTextRepresentation(rv)
	}

	return ValueHasTextRepresentation(reflect.ValueOf(v))
}

// ValueHasTextRepresentation reports whether [reflect.Value] v implements [encoding.TextMarshaler] or [fmt.Stringer].
func ValueHasTextRepresentation(v reflect.Value) bool {
	if !v.IsValid() || !v.CanInterface() {
		return false
	}

	val := v.Interface()
	_, hasText := val.(encoding.TextMarshaler)
	_, hasStringer := val.(fmt.Stringer)

	return hasText || hasStringer
}

// DerefPointer unwraps consecutive pointer levels until reaching a concrete value or a nil pointer.
func DerefPointer(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return reflect.Value{}
		}

		v = v.Elem()
	}

	return v
}
