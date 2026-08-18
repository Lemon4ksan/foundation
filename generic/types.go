// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generic

import "reflect"

// Option represents a functional option that configures a generic target.
type Option[T any] func(T)

// ApplyOptions sequentially applies a list of functional options to the target object.
// If any option is nil, it is silently skipped.
func ApplyOptions[T any](target *T, opts ...Option[*T]) {
	for _, opt := range opts {
		if opt != nil {
			opt(target)
		}
	}
}

// Ptr returns a pointer to the provided value.
// It is useful for inline initialization of pointer fields with literals.
func Ptr[T any](v T) *T {
	return &v
}

// PtrOrNil returns a pointer to the given value, or nil if the value is the zero value of its type.
func PtrOrNil[T comparable](v T) *T {
	var zero T
	if v == zero {
		return nil
	}

	return &v
}

// Zero returns the zero value of the given type T.
func Zero[T any]() T {
	var zero T
	return zero
}

// IsZero reports whether the provided value is the zero value of its type.
func IsZero[T comparable](v T) bool {
	var zero T
	return v == zero
}

// Deref safely dereferences the provided pointer.
// If the pointer is nil, it returns the zero value of type T.
func Deref[T any](ptr *T) T {
	if ptr == nil {
		var zero T
		return zero
	}

	return *ptr
}

// DerefOr safely dereferences the provided pointer.
// If the pointer is nil, it returns the provided fallback default value.
func DerefOr[T any](ptr *T, def T) T {
	if ptr == nil {
		return def
	}

	return *ptr
}

// Coalesce returns the first non-zero value from the provided list.
// If all values are the zero value of type T, it returns the zero value.
func Coalesce[T comparable](vals ...T) T {
	var zero T
	for _, v := range vals {
		if v != zero {
			return v
		}
	}

	return zero
}

// CoalesceNil returns the first non-nil value from the provided list.
//
// It supports all nillable types (pointers, interfaces, maps, slices, channels, and functions)
// by performing safe reflection checks to bypass Go's typed-nil interface comparison constraints.
// If all values are nil, it returns the zero value of type T.
func CoalesceNil[T any](vals ...T) T {
	for _, v := range vals {
		if !isNil(v) {
			return v
		}
	}

	var zero T

	return zero
}

// isNil safely determines if the provided interface value is nil.
//
// It bypasses the common Go pitfall where an interface wrapping a nil pointer is not considered nil.
func isNil(val any) bool {
	if val == nil {
		return true
	}

	v := reflect.ValueOf(val)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Pointer, reflect.UnsafePointer, reflect.Slice:
		return v.IsNil()
	}

	return false
}

// Ternary emulates a traditional ternary operator for generic types.
// It returns a if cond is true, otherwise it returns b.
func Ternary[T any](cond bool, a, b T) T {
	if cond {
		return a
	}

	return b
}
