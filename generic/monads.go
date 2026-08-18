// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generic

import (
	"errors"
	"reflect"
)

// Optional represents a type-safe container that may or may not contain a valid
// non-nil value.
//
// It is the functional equivalent of Swift's Optional type, protecting applications
// from nil-pointer dereference panics.
//
// The zero value of Optional is ready to use and represents the absence of a value,
// structurally equivalent to [None].
type Optional[T any] struct {
	val   T
	valid bool
}

// Some instantiates an [Optional] wrapping a valid, non-empty value of type T.
func Some[T any](v T) Optional[T] {
	return Optional[T]{val: v, valid: true}
}

// None instantiates an empty [Optional] representing the absence of a value.
func None[T any]() Optional[T] {
	return Optional[T]{}
}

// IsPresent returns true if the optional contains a valid wrapped value.
func (o Optional[T]) IsPresent() bool {
	return o.valid
}

// Value returns the wrapped value and a boolean indicating if the value is present.
//
// If the optional is empty, it returns the zero value of type T and false.
func (o Optional[T]) Value() (T, bool) {
	return o.val, o.valid
}

// ValueOr returns the wrapped value if present, otherwise returning the fallback value.
func (o Optional[T]) ValueOr(fallback T) T {
	if !o.valid {
		return fallback
	}

	return o.val
}

// Filter returns the [Optional] containing the value if present and matching
// the predicate, otherwise returning [None].
//
// If the predicate function is nil, Filter returns [None] to prevent runtime panics.
func (o Optional[T]) Filter(predicate func(T) bool) Optional[T] {
	if predicate == nil || !o.valid || !predicate(o.val) {
		return None[T]()
	}

	return o
}

// MapOptional transforms the value inside o using f if it is present, returning
// a new [Optional] wrapping the result.
//
// If the transformer function f is nil, MapOptional returns [None] of type U.
// Due to Go's generic limitations where struct methods cannot introduce new
// type parameters, this is implemented as a package-level function.
func MapOptional[T, U any](o Optional[T], f func(T) U) Optional[U] {
	if f == nil || !o.valid {
		return None[U]()
	}

	return Some(f(o.val))
}

// FlatMapOptional transforms the value inside o using f, returning another [Optional]
// if the value is present.
//
// If the transformer function f is nil, FlatMapOptional returns [None] of type U.
func FlatMapOptional[T, U any](o Optional[T], f func(T) Optional[U]) Optional[U] {
	if f == nil || !o.valid {
		return None[U]()
	}

	return f(o.val)
}

// Result represents the outcome of an operation that either succeeds with a value
// of type T or fails with an error.
//
// It is the functional equivalent of Swift's Result type, providing an explicit,
// structured alternative to returning multiple values like (T, error).
//
// The zero value of Result represents a successful execution wrapping the zero value
// of type T and a nil error.
type Result[T any] struct {
	val T
	err error
}

// Success instantiates a successful [Result] wrapping the computed value.
func Success[T any](v T) Result[T] {
	return Result[T]{val: v}
}

// Failure instantiates a failed [Result] wrapping the associated execution error.
//
// If the provided error is nil, the result is treated as a success wrapping
// the zero value of type T.
func Failure[T any](err error) Result[T] {
	return Result[T]{err: err}
}

// IsSuccess returns true if the result represents a successful operation (err is nil).
func (r Result[T]) IsSuccess() bool {
	return r.err == nil
}

// Unwrap returns the computed value and any associated execution error.
func (r Result[T]) Unwrap() (T, error) {
	return r.val, r.err
}

// Recover returns the wrapped value on success, or uses f to compute a fallback
// value on failure.
//
// If the recovery function f is nil, Recover returns the zero value of type T.
func (r Result[T]) Recover(f func(error) T) T {
	if r.err != nil {
		if f == nil {
			var zero T
			return zero
		}

		return f(r.err)
	}

	return r.val
}

// RecoverWith returns the [Result] itself if successful, or uses f to compute
// a fallback [Result] on failure.
//
// If the recovery function f is nil, RecoverWith returns the Result itself on failure.
func (r Result[T]) RecoverWith(f func(error) Result[T]) Result[T] {
	if r.err != nil {
		if f == nil {
			return r
		}

		return f(r.err)
	}

	return r
}

// MapResult transforms the value inside r using f if r represents a success,
// returning a new [Result] wrapping the result.
//
// If the transformer function f is nil, MapResult returns a failure wrapping
// a nil-mapper error. Due to Go's generic limitations where struct methods
// cannot introduce new type parameters, this is implemented as a package-level function.
func MapResult[T, U any](r Result[T], f func(T) U) Result[U] {
	if r.err != nil {
		return Failure[U](r.err)
	}

	if f == nil {
		return Failure[U](errors.New("generic: map function is nil"))
	}

	return Success(f(r.val))
}

// FlatMapResult transforms the value inside r using f, returning another [Result]
// if r represents a success.
//
// If the transformer function f is nil, FlatMapResult returns a failure wrapping
// a nil-mapper error.
func FlatMapResult[T, U any](r Result[T], f func(T) Result[U]) Result[U] {
	if r.err != nil {
		return Failure[U](r.err)
	}

	if f == nil {
		return Failure[U](errors.New("generic: flatmap function is nil"))
	}

	return f(r.val)
}

// TypedResult represents the outcome of an operation that either succeeds with a value
// of type T or fails with a specific error type E.
//
// It is the functional equivalent of Swift's Typed Throws. To prevent the
// common Go interface trap (where a typed nil pointer compared to nil interface
// returns false), TypedResult uses an explicit internal boolean flag to track failures,
// guaranteeing absolute correctness under concurrent reflect execution.
//
// The zero value of TypedResult represents a successful execution wrapping
// the zero value of type T and a zero error E.
type TypedResult[T any, E error] struct {
	val    T
	err    E
	hasErr bool
}

// SuccessTyped instantiates a successful [TypedResult] wrapping the computed value.
func SuccessTyped[T any, E error](v T) TypedResult[T, E] {
	return TypedResult[T, E]{val: v, hasErr: false}
}

// FailureTyped instantiates a failed [TypedResult] wrapping the associated execution error.
func FailureTyped[T any, E error](err E) TypedResult[T, E] {
	// If the error type itself is a nil interface value, prevent false-positive failures
	// by dynamically checking its interface reflection status.
	if isInterfaceNil(err) {
		return TypedResult[T, E]{hasErr: false}
	}

	return TypedResult[T, E]{err: err, hasErr: true}
}

// IsSuccess returns true if the result represents a successful execution (no error flag is set).
func (r TypedResult[T, E]) IsSuccess() bool {
	return !r.hasErr
}

// Unwrap returns the computed value and any associated execution error.
func (r TypedResult[T, E]) Unwrap() (T, E) {
	return r.val, r.err
}

// Recover returns the wrapped value on success, or uses f to compute a fallback
// value on failure.
//
// If the recovery function f is nil, Recover returns the zero value of type T.
func (r TypedResult[T, E]) Recover(f func(E) T) T {
	if !r.IsSuccess() {
		if f == nil {
			var zero T
			return zero
		}

		return f(r.err)
	}

	return r.val
}

// RecoverWith returns the [TypedResult] itself if successful, or uses f to compute
// a fallback [TypedResult] on failure.
//
// If the recovery function f is nil, RecoverWith returns the TypedResult itself on failure.
func (r TypedResult[T, E]) RecoverWith(f func(E) TypedResult[T, E]) TypedResult[T, E] {
	if !r.IsSuccess() {
		if f == nil {
			return r
		}

		return f(r.err)
	}

	return r
}

// MapTypedResult transforms the value inside r using f if r represents a success,
// returning a new [TypedResult] wrapping the result.
//
// If the transformer function f is nil, MapTypedResult returns a failure wrapping
// the zero value of error E.
func MapTypedResult[T, U any, E error](r TypedResult[T, E], f func(T) U) TypedResult[U, E] {
	if !r.IsSuccess() {
		return FailureTyped[U, E](r.err)
	}

	if f == nil {
		var zeroErr E
		return FailureTyped[U, E](zeroErr)
	}

	return SuccessTyped[U, E](f(r.val))
}

// FlatMapTypedResult transforms the value inside r using f, returning another [TypedResult]
// if r represents a success.
//
// If the transformer function f is nil, FlatMapTypedResult returns a failure wrapping
// the zero value of error E.
func FlatMapTypedResult[T, U any, E error](r TypedResult[T, E], f func(T) TypedResult[U, E]) TypedResult[U, E] {
	if !r.IsSuccess() {
		return FailureTyped[U, E](r.err)
	}

	if f == nil {
		var zeroErr E
		return FailureTyped[U, E](zeroErr)
	}

	return f(r.val)
}

// isInterfaceNil performs a dynamic reflection check to determine if the given error E
// wraps a nil concrete pointer under its interface.
func isInterfaceNil(err error) bool {
	val := reflect.ValueOf(err)
	if !val.IsValid() {
		return true
	}

	switch val.Kind() {
	case reflect.Chan,
		reflect.Func,
		reflect.Map,
		reflect.Pointer,
		reflect.UnsafePointer,
		reflect.Interface,
		reflect.Slice:
		return val.IsNil()
	default:
		return false
	}
}
