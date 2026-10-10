// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package refkit

import (
	"reflect"
)

// IsNil safely checks whether v is nil without panicking on non-nillable kinds.
//
// Returns true for invalid values and nil pointers, interfaces, channels, maps, slices, functions, or unsafe pointers.
// It executes in O(1) time and is safe for concurrent use.
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
//
// It executes in O(N) time where N is the size of the value.
// It is safe for concurrent use.
func IsZero(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}

	return v.IsZero()
}

// IsStruct reports whether v is a struct or a pointer to a struct.
//
// It executes in amortized O(1) time and is safe for concurrent use.
func IsStruct(v any) bool {
	return IndirectKind(v) == reflect.Struct
}

// IsCollection reports whether v is a slice, array, or map (or a pointer to one).
//
// It executes in amortized O(1) time and is safe for concurrent use.
func IsCollection(v any) bool {
	switch IndirectKind(v) {
	case reflect.Slice, reflect.Array, reflect.Map:
		return true
	default:
		return false
	}
}

// IsNumeric reports whether kind is an integer, unsigned integer, or floating-point type.
//
// It executes in O(1) time and is safe for concurrent use.
func IsNumeric(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
		return true
	default:
		return false
	}
}

// IsInteger reports whether kind is a signed or unsigned integer.
//
// It executes in O(1) time and is safe for concurrent use.
func IsInteger(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return true
	default:
		return false
	}
}

// IsFloat reports whether kind is a float32 or float64.
//
// It executes in O(1) time and is safe for concurrent use.
func IsFloat(k reflect.Kind) bool {
	return k == reflect.Float32 || k == reflect.Float64
}

// IsSigned reports whether kind is a signed integer.
//
// It executes in O(1) time and is safe for concurrent use.
func IsSigned(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return true
	default:
		return false
	}
}

// IsUnsigned reports whether kind is an unsigned integer.
//
// It executes in O(1) time and is safe for concurrent use.
func IsUnsigned(k reflect.Kind) bool {
	switch k {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return true
	default:
		return false
	}
}
