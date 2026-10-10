// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package refkit

import (
	"math"
	"reflect"
)

// AsInt64 extracts a 64-bit signed integer from v without interface boxing allocations.
//
// It succeeds for signed integer kinds ([reflect.Int], [reflect.Int8], [reflect.Int16],
// [reflect.Int32], [reflect.Int64]) and for unsigned integer kinds whose values fit
// within [math.MaxInt64]. For unsupported kinds, invalid values, or unsigned values that
// overflow [math.MaxInt64], it returns (0, false).
//
//go:inline
func AsInt64(v reflect.Value) (int64, bool) {
	if !v.IsValid() {
		return 0, false
	}

	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int(), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		u := v.Uint()
		if u <= math.MaxInt64 {
			return int64(u), true
		}
		return 0, false
	default:
		return 0, false
	}
}

// AsUint64 extracts a 64-bit unsigned integer from v without interface boxing allocations.
//
// It succeeds for unsigned integer kinds ([reflect.Uint], [reflect.Uint8], [reflect.Uint16],
// [reflect.Uint32], [reflect.Uint64], [reflect.Uintptr]) and for non-negative signed integers.
// For unsupported kinds, invalid values, or negative signed integers, it returns (0, false).
//
//go:inline
func AsUint64(v reflect.Value) (uint64, bool) {
	if !v.IsValid() {
		return 0, false
	}

	switch v.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint(), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i := v.Int()
		if i >= 0 {
			return uint64(i), true
		}
		return 0, false
	default:
		return 0, false
	}
}

// AsFloat64 extracts a 64-bit floating point number from v without interface boxing allocations.
//
// It succeeds for [reflect.Float32] and [reflect.Float64]. For other kinds or invalid values,
// it returns (0, false).
//
//go:inline
func AsFloat64(v reflect.Value) (float64, bool) {
	if !v.IsValid() {
		return 0, false
	}

	switch v.Kind() {
	case reflect.Float32, reflect.Float64:
		return v.Float(), true
	default:
		return 0, false
	}
}

// AsBool extracts a boolean from v without interface boxing allocations.
//
// It succeeds for [reflect.Bool]. For other kinds or invalid values, it returns (false, false).
//
//go:inline
func AsBool(v reflect.Value) (bool, bool) {
	if !v.IsValid() {
		return false, false
	}

	if v.Kind() == reflect.Bool {
		return v.Bool(), true
	}

	return false, false
}

// AsBytes extracts the underlying byte slice from v without copying or interface boxing allocations.
//
// It succeeds for byte slices ([]byte) and addressable byte arrays ([N]byte). For other kinds,
// non-byte slices/arrays, or invalid values, it returns (nil, false).
//
//go:inline
func AsBytes(v reflect.Value) ([]byte, bool) {
	if !v.IsValid() {
		return nil, false
	}

	if v.Kind() == reflect.Slice && v.Type().Elem().Kind() == reflect.Uint8 {
		return v.Bytes(), true
	}

	if v.Kind() == reflect.Array && v.Type().Elem().Kind() == reflect.Uint8 && v.CanAddr() {
		return v.Bytes(), true
	}

	return nil, false
}

// AsString extracts the string from v without copying or interface boxing allocations.
//
// It succeeds for [reflect.String]. For other kinds or invalid values, it returns ("", false).
//
//go:inline
func AsString(v reflect.Value) (string, bool) {
	if !v.IsValid() {
		return "", false
	}

	if v.Kind() == reflect.String {
		return v.String(), true
	}

	return "", false
}
