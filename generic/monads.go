// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generic

import (
	"errors"
	"fmt"
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

// From creates an [Optional] from a value and a boolean indicator (comma-ok pattern).
//
// If ok is true, it returns [Some](v). Otherwise, it returns [None].
func From[T any](v T, ok bool) Optional[T] {
	if !ok {
		return None[T]()
	}

	return Some(v)
}

// FromPtr creates an [Optional] wrapping a pointer value.
//
// If ptr is nil, it returns [None]. Otherwise, it returns [Some](ptr).
func FromPtr[T any](ptr *T) Optional[*T] {
	if ptr == nil {
		return None[*T]()
	}

	return Some(ptr)
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

// MustValue returns the wrapped value if present, or panics if the optional is empty.
func (o Optional[T]) MustValue() T {
	if !o.valid {
		panic("generic: called MustValue on an empty Optional")
	}

	return o.val
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

// ToResult instantiates a [Result] from a value and error tuple.
//
// If err is not nil, it returns [Failure](err). Otherwise, it returns [Success](v).
func ToResult[T any](v T, err error) Result[T] {
	if err != nil {
		return Failure[T](err)
	}

	return Success(v)
}

// IsSuccess returns true if the result represents a successful operation (err is nil).
func (r Result[T]) IsSuccess() bool {
	return r.err == nil
}

// Unwrap returns the computed value and any associated execution error.
func (r Result[T]) Unwrap() (T, error) {
	return r.val, r.err
}

// MustValue returns the computed value if successful, or panics with the execution error.
func (r Result[T]) MustValue() T {
	if r.err != nil {
		panic(r.err)
	}

	return r.val
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

// FromTypedResult instantiates a [TypedResult] from a value and typed error tuple.
//
// If err is not nil (or non-zero error), it returns [FailureTyped](err). Otherwise, it returns [SuccessTyped](v).
func FromTypedResult[T any, E error](v T, err E) TypedResult[T, E] {
	if isInterfaceNil(err) {
		return SuccessTyped[T, E](v)
	}

	return FailureTyped[T, E](err)
}

// IsSuccess returns true if the result represents a successful execution (no error flag is set).
func (r TypedResult[T, E]) IsSuccess() bool {
	return !r.hasErr
}

// Unwrap returns the computed value and any associated execution error.
func (r TypedResult[T, E]) Unwrap() (T, E) {
	return r.val, r.err
}

// MustValue returns the computed value if successful, or panics with the execution error.
func (r TypedResult[T, E]) MustValue() T {
	if !r.IsSuccess() {
		panic(r.err)
	}

	return r.val
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

// Either represents a value of one of two possible types (a disjoint union).
//
// By convention, [Left] represents an error, fallback, or alternative outcome,
// while [Right] represents a successful computation or preferred value.
type Either[L, R any] struct {
	left    L
	right   R
	isRight bool
}

// Left instantiates a left-biased [Either] wrapping the provided value.
func Left[L, R any](val L) Either[L, R] {
	return Either[L, R]{left: val, isRight: false}
}

// Right instantiates a right-biased [Either] wrapping the provided value.
func Right[L, R any](val R) Either[L, R] {
	return Either[L, R]{right: val, isRight: true}
}

// FromError creates an [Either] from a value and an error.
// If err is not nil, it returns [Left](err). Otherwise, it returns [Right](v).
func FromError[R any](v R, err error) Either[error, R] {
	if err != nil {
		return Left[error, R](err)
	}

	return Right[error, R](v)
}

// IsLeft returns true if the either represents a Left value.
func (e Either[L, R]) IsLeft() bool {
	return !e.isRight
}

// IsRight returns true if the either represents a Right value.
func (e Either[L, R]) IsRight() bool {
	return e.isRight
}

// Left returns the Left value if present, or the zero value of type L if Right.
func (e Either[L, R]) Left() L {
	return e.left
}

// Right returns the Right value if present, or the zero value of type R if Left.
func (e Either[L, R]) Right() R {
	return e.right
}

// LeftOptional returns the Left value wrapped in an [Optional].
func (e Either[L, R]) LeftOptional() Optional[L] {
	return From(e.left, !e.isRight)
}

// RightOptional returns the Right value wrapped in an [Optional].
func (e Either[L, R]) RightOptional() Optional[R] {
	return From(e.right, e.isRight)
}

// ValueOr returns the Right value if present, otherwise returning the fallback value.
func (e Either[L, R]) ValueOr(fallback R) R {
	if !e.isRight {
		return fallback
	}

	return e.right
}

// Fold invokes onLeft if the value is Left, or onRight if the value is Right.
func (e Either[L, R]) Fold(onLeft func(L), onRight func(R)) {
	if e.isRight {
		if onRight != nil {
			onRight(e.right)
		}
	} else {
		if onLeft != nil {
			onLeft(e.left)
		}
	}
}

// Swap inverts the [Either], transforming Left into Right and Right into Left.
func (e Either[L, R]) Swap() Either[R, L] {
	if e.isRight {
		return Left[R, L](e.right)
	}

	return Right[R, L](e.left)
}

// AsResult converts the [Either] into a [Result].
//
// If e is Right, it returns a successful [Result] wrapping the Right value.
// If e is Left and type-asserts to error, it returns a failed [Result] wrapping that error.
// Otherwise, it formats the Left value into an error using [fmt.Errorf].
func (e Either[L, R]) AsResult() Result[R] {
	if e.isRight {
		return Success(e.right)
	}

	if err, ok := any(e.left).(error); ok && err != nil {
		return Failure[R](err)
	}

	return Failure[R](fmt.Errorf("%v", e.left))
}

// MapEither transforms the Right value using fn, returning a new [Either] while preserving Left.
// Due to Go generic limitations where methods cannot introduce new type parameters,
// this is implemented as a package-level function.
func MapEither[L, R, NewR any](e Either[L, R], fn func(R) NewR) Either[L, NewR] {
	if e.isRight {
		if fn == nil {
			var zero NewR
			return Right[L, NewR](zero)
		}

		return Right[L, NewR](fn(e.right))
	}

	return Left[L, NewR](e.left)
}

// MapLeft transforms the Left value using fn, returning a new [Either] while preserving Right.
func MapLeft[L, R, NewL any](e Either[L, R], fn func(L) NewL) Either[NewL, R] {
	if !e.isRight {
		if fn == nil {
			var zero NewL
			return Left[NewL, R](zero)
		}

		return Left[NewL, R](fn(e.left))
	}

	return Right[NewL, R](e.right)
}

// FlatMapEither transforms the Right value using fn, returning a new [Either] if e is Right.
func FlatMapEither[L, R, NewR any](e Either[L, R], fn func(R) Either[L, NewR]) Either[L, NewR] {
	if e.isRight {
		if fn == nil {
			var zero NewR
			return Right[L, NewR](zero)
		}

		return fn(e.right)
	}

	return Left[L, NewR](e.left)
}

// FoldMap evaluates onLeft if e is Left, or onRight if e is Right, returning the mapped value.
func FoldMap[L, R, T any](e Either[L, R], onLeft func(L) T, onRight func(R) T) T {
	if e.isRight {
		if onRight != nil {
			return onRight(e.right)
		}

		var zero T
		return zero
	}

	if onLeft != nil {
		return onLeft(e.left)
	}

	var zero T
	return zero
}
