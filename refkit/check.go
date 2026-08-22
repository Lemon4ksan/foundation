// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package refkit

import (
	"reflect"
)

// IsNil safely checks whether v is nil without panicking on non-nillable kinds.
// Returns true for invalid values and nil pointers, interfaces, channels, maps, slices, functions, or unsafe pointers.
func IsNil(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}

	switch v.Kind() {
	case reflect.Chan,
		reflect.Func,
		reflect.Interface,
		reflect.Map,
		reflect.Pointer,
		reflect.Slice,
		reflect.UnsafePointer:
		return v.IsNil()
	default:
		return false
	}
}

// IsZero safely checks whether v is the zero value for its type, returning true for invalid values.
func IsZero(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}

	return v.IsZero()
}

// IsStruct reports whether v is a struct or a pointer to a struct.
func IsStruct(v any) bool {
	return IndirectKind(v) == reflect.Struct
}

// IsCollection reports whether v is a slice, array, or map (or a pointer to one).
func IsCollection(v any) bool {
	switch IndirectKind(v) {
	case reflect.Slice, reflect.Array, reflect.Map:
		return true
	default:
		return false
	}
}
