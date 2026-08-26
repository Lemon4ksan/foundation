// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package require provides immediate-failure testing assertions.
package require

import (
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/testkit/assert"
)

// AnError is an error value suitable for testing error handling.
var AnError = assert.AnError

// Equal asserts that two objects are equal, failing immediately if not.
func Equal(t testing.TB, expected, actual any, msgAndArgs ...any) {
	t.Helper()
	if !assert.Equal(t, expected, actual, msgAndArgs...) {
		t.FailNow()
	}
}

// Equalf asserts that two objects are equal with format, failing immediately if not.
func Equalf(t testing.TB, expected, actual any, format string, args ...any) {
	t.Helper()
	if !assert.Equalf(t, expected, actual, format, args...) {
		t.FailNow()
	}
}

// EqualValues asserts that two objects are equal after type conversion, failing immediately if not.
func EqualValues(t testing.TB, expected, actual any, msgAndArgs ...any) {
	t.Helper()
	if !assert.EqualValues(t, expected, actual, msgAndArgs...) {
		t.FailNow()
	}
}

// EqualValuesf asserts that two objects are equal after type conversion with format, failing immediately if not.
func EqualValuesf(t testing.TB, expected, actual any, format string, args ...any) {
	t.Helper()
	if !assert.EqualValuesf(t, expected, actual, format, args...) {
		t.FailNow()
	}
}

// EqualExportedValues asserts that the exported fields of two structs are equal.
func EqualExportedValues(t testing.TB, expected, actual any, msgAndArgs ...any) {
	t.Helper()
	if !assert.EqualExportedValues(t, expected, actual, msgAndArgs...) {
		t.FailNow()
	}
}

// EqualExportedValuesf asserts that the exported fields of two structs are equal with format.
func EqualExportedValuesf(t testing.TB, expected, actual any, format string, args ...any) {
	t.Helper()
	if !assert.EqualExportedValuesf(t, expected, actual, format, args...) {
		t.FailNow()
	}
}

// NotEqual asserts that two objects are not equal, failing immediately if they are.
func NotEqual(t testing.TB, expected, actual any, msgAndArgs ...any) {
	t.Helper()
	if !assert.NotEqual(t, expected, actual, msgAndArgs...) {
		t.FailNow()
	}
}

// NotEqualf asserts that two objects are not equal with format, failing immediately if they are.
func NotEqualf(t testing.TB, expected, actual any, format string, args ...any) {
	t.Helper()
	if !assert.NotEqualf(t, expected, actual, format, args...) {
		t.FailNow()
	}
}

// True asserts that value is true, failing immediately if false.
func True(t testing.TB, value bool, msgAndArgs ...any) {
	t.Helper()
	if !assert.True(t, value, msgAndArgs...) {
		t.FailNow()
	}
}

// Truef asserts that value is true with format, failing immediately if false.
func Truef(t testing.TB, value bool, format string, args ...any) {
	t.Helper()
	if !assert.Truef(t, value, format, args...) {
		t.FailNow()
	}
}

// False asserts that value is false, failing immediately if true.
func False(t testing.TB, value bool, msgAndArgs ...any) {
	t.Helper()
	if !assert.False(t, value, msgAndArgs...) {
		t.FailNow()
	}
}

// Falsef asserts that value is false with format, failing immediately if true.
func Falsef(t testing.TB, value bool, format string, args ...any) {
	t.Helper()
	if !assert.Falsef(t, value, format, args...) {
		t.FailNow()
	}
}

// Nil asserts that object is nil, failing immediately if not.
func Nil(t testing.TB, object any, msgAndArgs ...any) bool {
	t.Helper()
	if !assert.Nil(t, object, msgAndArgs...) {
		t.FailNow()
	}
	return true
}

// Nilf asserts that object is nil with format, failing immediately if not.
func Nilf(t testing.TB, object any, format string, args ...any) {
	t.Helper()
	if !assert.Nilf(t, object, format, args...) {
		t.FailNow()
	}
}

// NotNil asserts that object is not nil, failing immediately if nil.
func NotNil(t testing.TB, object any, msgAndArgs ...any) bool {
	t.Helper()
	if !assert.NotNil(t, object, msgAndArgs...) {
		t.FailNow()
	}
	return true
}

// NotNilf asserts that object is not nil with format, failing immediately if nil.
func NotNilf(t testing.TB, object any, format string, args ...any) {
	t.Helper()
	if !assert.NotNilf(t, object, format, args...) {
		t.FailNow()
	}
}

// NoError asserts that err is nil, failing immediately if non-nil.
func NoError(t testing.TB, err error, msgAndArgs ...any) {
	t.Helper()
	if !assert.NoError(t, err, msgAndArgs...) {
		t.FailNow()
	}
}

// NoErrorf asserts that err is nil with format, failing immediately if non-nil.
func NoErrorf(t testing.TB, err error, format string, args ...any) {
	t.Helper()
	if !assert.NoErrorf(t, err, format, args...) {
		t.FailNow()
	}
}

// Error asserts that err is non-nil, failing immediately if nil.
func Error(t testing.TB, err error, msgAndArgs ...any) {
	t.Helper()
	if !assert.Error(t, err, msgAndArgs...) {
		t.FailNow()
	}
}

// Errorf asserts that err is non-nil with format, failing immediately if nil.
func Errorf(t testing.TB, err error, format string, args ...any) {
	t.Helper()
	if !assert.Errorf(t, err, format, args...) {
		t.FailNow()
	}
}

// EqualError asserts that err is not nil and matches errString, failing immediately if not.
func EqualError(t testing.TB, err error, errString string, msgAndArgs ...any) {
	t.Helper()
	if !assert.EqualError(t, err, errString, msgAndArgs...) {
		t.FailNow()
	}
}

// EqualErrorf asserts that err is not nil and matches errString with format.
func EqualErrorf(t testing.TB, err error, errString string, format string, args ...any) {
	t.Helper()
	if !assert.EqualErrorf(t, err, errString, format, args...) {
		t.FailNow()
	}
}

// ErrorIs asserts that target is in err chain, failing immediately if not.
func ErrorIs(t testing.TB, err, target error, msgAndArgs ...any) {
	t.Helper()
	if !assert.ErrorIs(t, err, target, msgAndArgs...) {
		t.FailNow()
	}
}

// ErrorIsf asserts that target is in err chain with format.
func ErrorIsf(t testing.TB, err, target error, format string, args ...any) {
	t.Helper()
	if !assert.ErrorIsf(t, err, target, format, args...) {
		t.FailNow()
	}
}

// NotErrorIs asserts that target is not in err chain, failing immediately if it is.
func NotErrorIs(t testing.TB, err, target error, msgAndArgs ...any) {
	t.Helper()
	if !assert.NotErrorIs(t, err, target, msgAndArgs...) {
		t.FailNow()
	}
}

// NotErrorIsf asserts that target is not in err chain with format.
func NotErrorIsf(t testing.TB, err, target error, format string, args ...any) {
	t.Helper()
	if !assert.NotErrorIsf(t, err, target, format, args...) {
		t.FailNow()
	}
}

// ErrorAs asserts that errors.As(err, target) succeeds, failing immediately if not.
func ErrorAs(t testing.TB, err error, target any, msgAndArgs ...any) {
	t.Helper()
	if !assert.ErrorAs(t, err, target, msgAndArgs...) {
		t.FailNow()
	}
}

// ErrorAsf asserts that errors.As(err, target) succeeds with format.
func ErrorAsf(t testing.TB, err error, target any, format string, args ...any) {
	t.Helper()
	if !assert.ErrorAsf(t, err, target, format, args...) {
		t.FailNow()
	}
}

// ErrorContains asserts that err contains substring, failing immediately if not.
func ErrorContains(t testing.TB, err error, contains string, msgAndArgs ...any) {
	t.Helper()
	if !assert.ErrorContains(t, err, contains, msgAndArgs...) {
		t.FailNow()
	}
}

// ErrorContainsf asserts that err contains substring with format.
func ErrorContainsf(t testing.TB, err error, contains string, format string, args ...any) {
	t.Helper()
	if !assert.ErrorContainsf(t, err, contains, format, args...) {
		t.FailNow()
	}
}

// Contains asserts container contains element, failing immediately if not.
func Contains(t testing.TB, container, element any, msgAndArgs ...any) {
	t.Helper()
	if !assert.Contains(t, container, element, msgAndArgs...) {
		t.FailNow()
	}
}

// Containsf asserts container contains element with format.
func Containsf(t testing.TB, container, element any, format string, args ...any) {
	t.Helper()
	if !assert.Containsf(t, container, element, format, args...) {
		t.FailNow()
	}
}

// NotContains asserts container does not contain element, failing immediately if it does.
func NotContains(t testing.TB, container, element any, msgAndArgs ...any) {
	t.Helper()
	if !assert.NotContains(t, container, element, msgAndArgs...) {
		t.FailNow()
	}
}

// NotContainsf asserts container does not contain element with format.
func NotContainsf(t testing.TB, container, element any, format string, args ...any) {
	t.Helper()
	if !assert.NotContainsf(t, container, element, format, args...) {
		t.FailNow()
	}
}

// Len asserts object has length, failing immediately if not.
func Len(t testing.TB, object any, length int, msgAndArgs ...any) {
	t.Helper()
	if !assert.Len(t, object, length, msgAndArgs...) {
		t.FailNow()
	}
}

// Lenf asserts object has length with format.
func Lenf(t testing.TB, object any, length int, format string, args ...any) {
	t.Helper()
	if !assert.Lenf(t, object, length, format, args...) {
		t.FailNow()
	}
}

// Empty asserts object is empty, failing immediately if not.
func Empty(t testing.TB, object any, msgAndArgs ...any) {
	t.Helper()
	if !assert.Empty(t, object, msgAndArgs...) {
		t.FailNow()
	}
}

// Emptyf asserts object is empty with format.
func Emptyf(t testing.TB, object any, format string, args ...any) {
	t.Helper()
	if !assert.Emptyf(t, object, format, args...) {
		t.FailNow()
	}
}

// NotEmpty asserts object is not empty, failing immediately if empty.
func NotEmpty(t testing.TB, object any, msgAndArgs ...any) {
	t.Helper()
	if !assert.NotEmpty(t, object, msgAndArgs...) {
		t.FailNow()
	}
}

// NotEmptyf asserts object is not empty with format.
func NotEmptyf(t testing.TB, object any, format string, args ...any) {
	t.Helper()
	if !assert.NotEmptyf(t, object, format, args...) {
		t.FailNow()
	}
}

// Zero asserts object is its type zero value, failing immediately if not.
func Zero(t testing.TB, object any, msgAndArgs ...any) {
	t.Helper()
	if !assert.Zero(t, object, msgAndArgs...) {
		t.FailNow()
	}
}

// Zerof asserts object is its type zero value with format.
func Zerof(t testing.TB, object any, format string, args ...any) {
	t.Helper()
	if !assert.Zerof(t, object, format, args...) {
		t.FailNow()
	}
}

// NotZero asserts object is not zero value, failing immediately if zero.
func NotZero(t testing.TB, object any, msgAndArgs ...any) {
	t.Helper()
	if !assert.NotZero(t, object, msgAndArgs...) {
		t.FailNow()
	}
}

// NotZerof asserts object is not zero value with format.
func NotZerof(t testing.TB, object any, format string, args ...any) {
	t.Helper()
	if !assert.NotZerof(t, object, format, args...) {
		t.FailNow()
	}
}

// Greater asserts e1 > e2, failing immediately if not.
func Greater[T assert.Ordered](t testing.TB, e1, e2 T, msgAndArgs ...any) {
	t.Helper()
	if !assert.Greater(t, e1, e2, msgAndArgs...) {
		t.FailNow()
	}
}

// Greaterf asserts e1 > e2 with format.
func Greaterf[T assert.Ordered](t testing.TB, e1, e2 T, format string, args ...any) {
	t.Helper()
	if !assert.Greaterf(t, e1, e2, format, args...) {
		t.FailNow()
	}
}

// GreaterOrEqual asserts e1 >= e2, failing immediately if not.
func GreaterOrEqual[T assert.Ordered](t testing.TB, e1, e2 T, msgAndArgs ...any) {
	t.Helper()
	if !assert.GreaterOrEqual(t, e1, e2, msgAndArgs...) {
		t.FailNow()
	}
}

// GreaterOrEqualf asserts e1 >= e2 with format.
func GreaterOrEqualf[T assert.Ordered](t testing.TB, e1, e2 T, format string, args ...any) {
	t.Helper()
	if !assert.GreaterOrEqualf(t, e1, e2, format, args...) {
		t.FailNow()
	}
}

// Less asserts e1 < e2, failing immediately if not.
func Less[T assert.Ordered](t testing.TB, e1, e2 T, msgAndArgs ...any) {
	t.Helper()
	if !assert.Less(t, e1, e2, msgAndArgs...) {
		t.FailNow()
	}
}

// Lessf asserts e1 < e2 with format.
func Lessf[T assert.Ordered](t testing.TB, e1, e2 T, format string, args ...any) {
	t.Helper()
	if !assert.Lessf(t, e1, e2, format, args...) {
		t.FailNow()
	}
}

// LessOrEqual asserts e1 <= e2, failing immediately if not.
func LessOrEqual[T assert.Ordered](t testing.TB, e1, e2 T, msgAndArgs ...any) {
	t.Helper()
	if !assert.LessOrEqual(t, e1, e2, msgAndArgs...) {
		t.FailNow()
	}
}

// LessOrEqualf asserts e1 <= e2 with format, failing immediately if not.
func LessOrEqualf[T assert.Ordered](t testing.TB, e1, e2 T, format string, args ...any) {
	t.Helper()
	if !assert.LessOrEqualf(t, e1, e2, format, args...) {
		t.FailNow()
	}
}

// InDelta asserts |expected - actual| <= delta, failing immediately if not.
func InDelta(t testing.TB, expected, actual any, delta float64, msgAndArgs ...any) {
	t.Helper()
	if !assert.InDelta(t, expected, actual, delta, msgAndArgs...) {
		t.FailNow()
	}
}

// InDeltaf asserts |expected - actual| <= delta with format.
func InDeltaf(t testing.TB, expected, actual any, delta float64, format string, args ...any) {
	t.Helper()
	if !assert.InDeltaf(t, expected, actual, delta, format, args...) {
		t.FailNow()
	}
}

// IsType asserts object is of same type as expectedType, failing immediately if not.
func IsType(t testing.TB, expectedType, object any, msgAndArgs ...any) {
	t.Helper()
	if !assert.IsType(t, expectedType, object, msgAndArgs...) {
		t.FailNow()
	}
}

// IsTypef asserts object is of same type as expectedType with format.
func IsTypef(t testing.TB, expectedType, object any, format string, args ...any) {
	t.Helper()
	if !assert.IsTypef(t, expectedType, object, format, args...) {
		t.FailNow()
	}
}

// JSONEq asserts two JSON strings are equivalent, failing immediately if not.
func JSONEq(t testing.TB, expected, actual string, msgAndArgs ...any) {
	t.Helper()
	if !assert.JSONEq(t, expected, actual, msgAndArgs...) {
		t.FailNow()
	}
}

// JSONEqf asserts two JSON strings are equivalent with format.
func JSONEqf(t testing.TB, expected, actual string, format string, args ...any) {
	t.Helper()
	if !assert.JSONEqf(t, expected, actual, format, args...) {
		t.FailNow()
	}
}

// Regexp asserts string matches regular expression, failing immediately if not.
func Regexp(t testing.TB, rx, str any, msgAndArgs ...any) {
	t.Helper()
	if !assert.Regexp(t, rx, str, msgAndArgs...) {
		t.FailNow()
	}
}

// Regexpf asserts string matches regular expression with format.
func Regexpf(t testing.TB, rx, str any, format string, args ...any) {
	t.Helper()
	if !assert.Regexpf(t, rx, str, format, args...) {
		t.FailNow()
	}
}

// Same asserts two pointers reference the same object, failing immediately if not.
func Same(t testing.TB, expected, actual any, msgAndArgs ...any) {
	t.Helper()
	if !assert.Same(t, expected, actual, msgAndArgs...) {
		t.FailNow()
	}
}

// Samef asserts two pointers reference the same object with format.
func Samef(t testing.TB, expected, actual any, format string, args ...any) {
	t.Helper()
	if !assert.Samef(t, expected, actual, format, args...) {
		t.FailNow()
	}
}

// NotSame asserts two pointers do not reference the same object, failing immediately if they do.
func NotSame(t testing.TB, expected, actual any, msgAndArgs ...any) {
	t.Helper()
	if !assert.NotSame(t, expected, actual, msgAndArgs...) {
		t.FailNow()
	}
}

// NotSamef asserts two pointers do not reference the same object with format, failing immediately if they do.
func NotSamef(t testing.TB, expected, actual any, format string, args ...any) {
	t.Helper()
	if !assert.NotSamef(t, expected, actual, format, args...) {
		t.FailNow()
	}
}


// WithinDuration asserts two times are within delta of each other, failing immediately if not.
func WithinDuration(t testing.TB, expected, actual time.Time, delta time.Duration, msgAndArgs ...any) {
	t.Helper()
	if !assert.WithinDuration(t, expected, actual, delta, msgAndArgs...) {
		t.FailNow()
	}
}

// WithinDurationf asserts two times are within delta of each other with format.
func WithinDurationf(t testing.TB, expected, actual time.Time, delta time.Duration, format string, args ...any) {
	t.Helper()
	if !assert.WithinDurationf(t, expected, actual, delta, format, args...) {
		t.FailNow()
	}
}

// FileExists asserts that file exists, failing immediately if not.
func FileExists(t testing.TB, filepath string, msgAndArgs ...any) {
	t.Helper()
	if !assert.FileExists(t, filepath, msgAndArgs...) {
		t.FailNow()
	}
}

// FileExistsf asserts that file exists with format.
func FileExistsf(t testing.TB, filepath string, format string, args ...any) {
	t.Helper()
	if !assert.FileExistsf(t, filepath, format, args...) {
		t.FailNow()
	}
}

// NoFileExists asserts that file does not exist, failing immediately if it does.
func NoFileExists(t testing.TB, filepath string, msgAndArgs ...any) {
	t.Helper()
	if !assert.NoFileExists(t, filepath, msgAndArgs...) {
		t.FailNow()
	}
}

// NoFileExistsf asserts that file does not exist with format.
func NoFileExistsf(t testing.TB, filepath string, format string, args ...any) {
	t.Helper()
	if !assert.NoFileExistsf(t, filepath, format, args...) {
		t.FailNow()
	}
}

// NoDirExists asserts that directory does not exist, failing immediately if it does.
func NoDirExists(t testing.TB, dirpath string, msgAndArgs ...any) {
	t.Helper()
	if !assert.NoDirExists(t, dirpath, msgAndArgs...) {
		t.FailNow()
	}
}

// NoDirExistsf asserts that directory does not exist with format.
func NoDirExistsf(t testing.TB, dirpath string, format string, args ...any) {
	t.Helper()
	if !assert.NoDirExistsf(t, dirpath, format, args...) {
		t.FailNow()
	}
}

// Panics asserts that f panics, failing immediately if it does not.
func Panics(t testing.TB, f func(), msgAndArgs ...any) {
	t.Helper()
	if !assert.Panics(t, f, msgAndArgs...) {
		t.FailNow()
	}
}

// Panicsf asserts that f panics with format.
func Panicsf(t testing.TB, f func(), format string, args ...any) {
	t.Helper()
	if !assert.Panicsf(t, f, format, args...) {
		t.FailNow()
	}
}

// PanicsWithError asserts that f panics with specific error string, failing immediately if not.
func PanicsWithError(t testing.TB, errString string, f func(), msgAndArgs ...any) {
	t.Helper()
	if !assert.PanicsWithError(t, errString, f, msgAndArgs...) {
		t.FailNow()
	}
}

// PanicsWithErrorf asserts that f panics with specific error string with format.
func PanicsWithErrorf(t testing.TB, errString string, f func(), format string, args ...any) {
	t.Helper()
	if !assert.PanicsWithErrorf(t, errString, f, format, args...) {
		t.FailNow()
	}
}

// NotPanics asserts that f does not panic, failing immediately if it does.
func NotPanics(t testing.TB, f func(), msgAndArgs ...any) {
	t.Helper()
	if !assert.NotPanics(t, f, msgAndArgs...) {
		t.FailNow()
	}
}

// NotPanicsf asserts that f does not panic with format.
func NotPanicsf(t testing.TB, f func(), format string, args ...any) {
	t.Helper()
	if !assert.NotPanicsf(t, f, format, args...) {
		t.FailNow()
	}
}

// Eventually asserts condition becomes true within waitFor duration, failing immediately if not.
func Eventually(t testing.TB, condition func() bool, waitFor, tick time.Duration, msgAndArgs ...any) {
	t.Helper()
	if !assert.Eventually(t, condition, waitFor, tick, msgAndArgs...) {
		t.FailNow()
	}
}

// Eventuallyf asserts condition becomes true within waitFor duration with format.
func Eventuallyf(t testing.TB, condition func() bool, waitFor, tick time.Duration, format string, args ...any) {
	t.Helper()
	if !assert.Eventuallyf(t, condition, waitFor, tick, format, args...) {
		t.FailNow()
	}
}

// Never asserts condition never becomes true within waitFor duration, failing immediately if it does.
func Never(t testing.TB, condition func() bool, waitFor, tick time.Duration, msgAndArgs ...any) {
	t.Helper()
	if !assert.Never(t, condition, waitFor, tick, msgAndArgs...) {
		t.FailNow()
	}
}

// Neverf asserts condition never becomes true within waitFor duration with format.
func Neverf(t testing.TB, condition func() bool, waitFor, tick time.Duration, format string, args ...any) {
	t.Helper()
	if !assert.Neverf(t, condition, waitFor, tick, format, args...) {
		t.FailNow()
	}
}

// ElementsMatch asserts that listA contains the same elements as listB, failing immediately if not.
func ElementsMatch(t testing.TB, listA, listB any, msgAndArgs ...any) {
	t.Helper()
	if !assert.ElementsMatch(t, listA, listB, msgAndArgs...) {
		t.FailNow()
	}
}

// ElementsMatchf asserts that listA contains the same elements as listB with format, failing immediately if not.
func ElementsMatchf(t testing.TB, listA, listB any, format string, args ...any) {
	t.Helper()
	if !assert.ElementsMatchf(t, listA, listB, format, args...) {
		t.FailNow()
	}
}

