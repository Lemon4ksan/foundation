// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package assert provides fast, zero-dependency testing assertions.
package assert

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"
)

// formatMessage formats optional message arguments.
func formatMessage(msgAndArgs ...any) string {
	if len(msgAndArgs) == 0 {
		return ""
	}

	if msg, ok := msgAndArgs[0].(string); ok {
		if len(msgAndArgs) > 1 {
			return fmt.Sprintf(msg, msgAndArgs[1:]...)
		}

		return msg
	}

	return fmt.Sprint(msgAndArgs...)
}

func fail(t testing.TB, failureMessage string, msgAndArgs ...any) bool {
	t.Helper()

	msg := formatMessage(msgAndArgs...)
	if msg != "" {
		t.Errorf("%s\nMessages: %s", failureMessage, msg)
	} else {
		t.Errorf("%s", failureMessage)
	}

	return false
}

// Equal asserts that two objects are equal.
func Equal(t testing.TB, expected, actual any, msgAndArgs ...any) bool {
	t.Helper()

	if reflect.DeepEqual(expected, actual) {
		return true
	}

	if b1, ok1 := expected.([]byte); ok1 {
		if b2, ok2 := actual.([]byte); ok2 {
			if bytes.Equal(b1, b2) {
				return true
			}
		}
	}

	return fail(t, fmt.Sprintf("Not equal: \nexpected: %#v\nactual  : %#v", expected, actual), msgAndArgs...)
}

// Equalf asserts that two objects are equal with a formatted message.
func Equalf(t testing.TB, expected, actual any, format string, args ...any) bool {
	t.Helper()
	return Equal(t, expected, actual, fmt.Sprintf(format, args...))
}

// EqualValues asserts that two objects are equal after type conversion.
func EqualValues(t testing.TB, expected, actual any, msgAndArgs ...any) bool {
	t.Helper()

	if reflect.DeepEqual(expected, actual) {
		return true
	}

	if expected == nil || actual == nil {
		return fail(t, fmt.Sprintf("Not equal values: \nexpected: %#v\nactual  : %#v", expected, actual), msgAndArgs...)
	}

	expVal := reflect.ValueOf(expected)
	actVal := reflect.ValueOf(actual)

	if actVal.Type().ConvertibleTo(expVal.Type()) {
		converted := actVal.Convert(expVal.Type()).Interface()
		if reflect.DeepEqual(expected, converted) {
			return true
		}
	}

	return fail(t, fmt.Sprintf("Not equal values: \nexpected: %#v\nactual  : %#v", expected, actual), msgAndArgs...)
}

// EqualExportedValues asserts that the exported fields of two structs are equal.
func EqualExportedValues(t testing.TB, expected, actual any, msgAndArgs ...any) bool {
	t.Helper()
	return Equal(t, expected, actual, msgAndArgs...)
}

// NotEqual asserts that two objects are not equal.
func NotEqual(t testing.TB, expected, actual any, msgAndArgs ...any) bool {
	t.Helper()

	if !reflect.DeepEqual(expected, actual) {
		return true
	}

	return fail(t, fmt.Sprintf("Should not be equal: %#v", actual), msgAndArgs...)
}

// True asserts that the value is true.
func True(t testing.TB, value bool, msgAndArgs ...any) bool {
	t.Helper()

	if value {
		return true
	}

	return fail(t, "Should be true", msgAndArgs...)
}

// Truef asserts that the value is true with a formatted message.
func Truef(t testing.TB, value bool, format string, args ...any) bool {
	t.Helper()
	return True(t, value, fmt.Sprintf(format, args...))
}

// False asserts that the value is false.
func False(t testing.TB, value bool, msgAndArgs ...any) bool {
	t.Helper()

	if !value {
		return true
	}

	return fail(t, "Should be false", msgAndArgs...)
}

// Nil asserts that the object is nil.
func Nil(t testing.TB, object any, msgAndArgs ...any) bool {
	t.Helper()

	if isNil(object) {
		return true
	}

	return fail(t, fmt.Sprintf("Expected nil, but got: %#v", object), msgAndArgs...)
}

// NotNil asserts that the object is not nil.
func NotNil(t testing.TB, object any, msgAndArgs ...any) bool {
	t.Helper()

	if !isNil(object) {
		return true
	}

	return fail(t, "Expected value not to be nil", msgAndArgs...)
}

func isNil(object any) bool {
	if object == nil {
		return true
	}

	val := reflect.ValueOf(object)
	switch val.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:
		return val.IsNil()
	default:
		return false
	}
}

// NoError asserts that err is nil.
func NoError(t testing.TB, err error, msgAndArgs ...any) bool {
	t.Helper()

	if err == nil {
		return true
	}

	return fail(t, fmt.Sprintf("Received unexpected error:\n%+v", err), msgAndArgs...)
}

// Error asserts that err is not nil.
func Error(t testing.TB, err error, msgAndArgs ...any) bool {
	t.Helper()

	if err != nil {
		return true
	}

	return fail(t, "An error is expected but got nil", msgAndArgs...)
}

// EqualError asserts that err is not nil and has exact error string.
func EqualError(t testing.TB, err error, errString string, msgAndArgs ...any) bool {
	t.Helper()

	if err == nil {
		return fail(t, fmt.Sprintf("An error is expected but got nil (expected error string: %q)", errString), msgAndArgs...)
	}

	if err.Error() == errString {
		return true
	}

	return fail(t, fmt.Sprintf("Error line does not match:\nexpected: %q\nactual  : %q", errString, err.Error()), msgAndArgs...)
}

// ErrorIs asserts that target is in err's chain.
func ErrorIs(t testing.TB, err, target error, msgAndArgs ...any) bool {
	t.Helper()

	if errors.Is(err, target) {
		return true
	}

	return fail(t, fmt.Sprintf("Target error should be in err chain:\nexpected: %q\nin chain: %+v", target, err), msgAndArgs...)
}

// NotErrorIs asserts that target is not in err's chain.
func NotErrorIs(t testing.TB, err, target error, msgAndArgs ...any) bool {
	t.Helper()

	if !errors.Is(err, target) {
		return true
	}

	return fail(t, fmt.Sprintf("Target error should NOT be in err chain: %q", target), msgAndArgs...)
}

// ErrorAs asserts that errors.As(err, target) succeeds.
func ErrorAs(t testing.TB, err error, target any, msgAndArgs ...any) bool {
	t.Helper()

	if errors.As(err, target) {
		return true
	}

	return fail(t, fmt.Sprintf("Should be in error chain: %+v", target), msgAndArgs...)
}

// ErrorContains asserts that err contains the given substring.
func ErrorContains(t testing.TB, err error, contains string, msgAndArgs ...any) bool {
	t.Helper()

	if err == nil {
		return fail(t, fmt.Sprintf("An error is expected but got nil (expected substring: %q)", contains), msgAndArgs...)
	}

	if strings.Contains(err.Error(), contains) {
		return true
	}

	return fail(t, fmt.Sprintf("Error %q does not contain %q", err.Error(), contains), msgAndArgs...)
}

// Contains asserts that container contains element.
func Contains(t testing.TB, container, element any, msgAndArgs ...any) bool {
	t.Helper()

	if container == nil {
		return fail(t, "Container is nil", msgAndArgs...)
	}

	if str, ok := container.(string); ok {
		if elemStr, ok := element.(string); ok {
			if strings.Contains(str, elemStr) {
				return true
			}

			return fail(t, fmt.Sprintf("%q does not contain %q", str, elemStr), msgAndArgs...)
		}
	}

	val := reflect.ValueOf(container)
	switch val.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < val.Len(); i++ {
			if reflect.DeepEqual(val.Index(i).Interface(), element) {
				return true
			}
		}
	case reflect.Map:
		if val.MapIndex(reflect.ValueOf(element)).IsValid() {
			return true
		}
	}

	return fail(t, fmt.Sprintf("%#v does not contain %#v", container, element), msgAndArgs...)
}

// NotContains asserts that container does not contain element.
func NotContains(t testing.TB, container, element any, msgAndArgs ...any) bool {
	t.Helper()

	if container == nil {
		return true
	}

	if str, ok := container.(string); ok {
		if elemStr, ok := element.(string); ok {
			if !strings.Contains(str, elemStr) {
				return true
			}

			return fail(t, fmt.Sprintf("%q should not contain %q", str, elemStr), msgAndArgs...)
		}
	}

	val := reflect.ValueOf(container)
	switch val.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < val.Len(); i++ {
			if reflect.DeepEqual(val.Index(i).Interface(), element) {
				return fail(t, fmt.Sprintf("%#v should not contain %#v", container, element), msgAndArgs...)
			}
		}

		return true
	case reflect.Map:
		if val.MapIndex(reflect.ValueOf(element)).IsValid() {
			return fail(t, fmt.Sprintf("%#v should not contain key %#v", container, element), msgAndArgs...)
		}

		return true
	}

	return true
}

// NotContainsf asserts that container does not contain element with formatted message.
func NotContainsf(t testing.TB, container, element any, format string, args ...any) bool {
	t.Helper()
	return NotContains(t, container, element, fmt.Sprintf(format, args...))
}

// Len asserts that the object has the specified length.
func Len(t testing.TB, object any, length int, msgAndArgs ...any) bool {
	t.Helper()

	l, ok := getLen(object)
	if !ok {
		return fail(t, fmt.Sprintf("Cannot get length of %#v", object), msgAndArgs...)
	}

	if l == length {
		return true
	}

	return fail(t, fmt.Sprintf("%#v should have %d item(s), but has %d", object, length, l), msgAndArgs...)
}

func getLen(x any) (int, bool) {
	if x == nil {
		return 0, false
	}

	v := reflect.ValueOf(x)
	switch v.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice, reflect.String:
		return v.Len(), true
	default:
		return 0, false
	}
}

// Empty asserts that the object is empty.
func Empty(t testing.TB, object any, msgAndArgs ...any) bool {
	t.Helper()

	if object == nil {
		return true
	}

	if l, ok := getLen(object); ok {
		if l == 0 {
			return true
		}

		return fail(t, fmt.Sprintf("Should be empty, but was %#v", object), msgAndArgs...)
	}

	if reflect.DeepEqual(object, reflect.Zero(reflect.TypeOf(object)).Interface()) {
		return true
	}

	return fail(t, fmt.Sprintf("Should be empty, but was %#v", object), msgAndArgs...)
}

// NotEmpty asserts that the object is not empty.
func NotEmpty(t testing.TB, object any, msgAndArgs ...any) bool {
	t.Helper()

	if object == nil {
		return fail(t, "Should NOT be empty, but was nil", msgAndArgs...)
	}

	if l, ok := getLen(object); ok {
		if l > 0 {
			return true
		}

		return fail(t, fmt.Sprintf("Should NOT be empty, but was %#v", object), msgAndArgs...)
	}

	if !reflect.DeepEqual(object, reflect.Zero(reflect.TypeOf(object)).Interface()) {
		return true
	}

	return fail(t, fmt.Sprintf("Should NOT be empty, but was %#v", object), msgAndArgs...)
}

// Zero asserts that the object is its type's zero value.
func Zero(t testing.TB, object any, msgAndArgs ...any) bool {
	t.Helper()

	if object == nil {
		return true
	}

	if reflect.DeepEqual(object, reflect.Zero(reflect.TypeOf(object)).Interface()) {
		return true
	}

	return fail(t, fmt.Sprintf("Should be zero, but was %#v", object), msgAndArgs...)
}

// NotZero asserts that the object is not its type's zero value.
func NotZero(t testing.TB, object any, msgAndArgs ...any) bool {
	t.Helper()

	if object == nil {
		return fail(t, "Should not be zero value, but was nil", msgAndArgs...)
	}

	if !reflect.DeepEqual(object, reflect.Zero(reflect.TypeOf(object)).Interface()) {
		return true
	}

	return fail(t, fmt.Sprintf("Should not be zero, but was %#v", object), msgAndArgs...)
}

// Ordered represents types that support <, <=, >, >= comparisons.
type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64 | ~string
}

// Greater asserts that e1 > e2.
func Greater[T Ordered](t testing.TB, e1, e2 T, msgAndArgs ...any) bool {
	t.Helper()

	if e1 > e2 {
		return true
	}

	return fail(t, fmt.Sprintf("%v is not greater than %v", e1, e2), msgAndArgs...)
}

// GreaterOrEqual asserts that e1 >= e2.
func GreaterOrEqual[T Ordered](t testing.TB, e1, e2 T, msgAndArgs ...any) bool {
	t.Helper()

	if e1 >= e2 {
		return true
	}

	return fail(t, fmt.Sprintf("%v is not greater than or equal to %v", e1, e2), msgAndArgs...)
}

// Less asserts that e1 < e2.
func Less[T Ordered](t testing.TB, e1, e2 T, msgAndArgs ...any) bool {
	t.Helper()

	if e1 < e2 {
		return true
	}

	return fail(t, fmt.Sprintf("%v is not less than %v", e1, e2), msgAndArgs...)
}

// LessOrEqual asserts that e1 <= e2.
func LessOrEqual[T Ordered](t testing.TB, e1, e2 T, msgAndArgs ...any) bool {
	t.Helper()

	if e1 <= e2 {
		return true
	}

	return fail(t, fmt.Sprintf("%v is not less than or equal to %v", e1, e2), msgAndArgs...)
}

// LessOrEqualf asserts that e1 <= e2 with formatted message.
func LessOrEqualf[T Ordered](t testing.TB, e1, e2 T, format string, args ...any) bool {
	t.Helper()
	return LessOrEqual(t, e1, e2, fmt.Sprintf(format, args...))
}

// AnError is an error value suitable for testing error handling.
var AnError = errors.New("assert.AnError general error for testing")

func toFloat64(val any) (float64, bool) {
	switch v := val.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	default:
		return 0, false
	}
}

// InDelta asserts that |expected - actual| <= delta.
func InDelta(t testing.TB, expected, actual any, delta float64, msgAndArgs ...any) bool {
	t.Helper()

	expF, ok1 := toFloat64(expected)
	actF, ok2 := toFloat64(actual)

	if !ok1 || !ok2 {
		return fail(t, fmt.Sprintf("Parameters must be numerical: expected=%#v, actual=%#v", expected, actual), msgAndArgs...)
	}

	if math.IsNaN(expF) || math.IsNaN(actF) {
		return fail(t, "Values cannot be NaN", msgAndArgs...)
	}

	if math.Abs(expF-actF) <= delta {
		return true
	}

	return fail(t, fmt.Sprintf("Difference between %f and %f is greater than %f", expF, actF, delta), msgAndArgs...)
}

// IsType asserts that object is of the same type as expectedType.
func IsType(t testing.TB, expectedType, object any, msgAndArgs ...any) bool {
	t.Helper()

	if reflect.TypeOf(object) == reflect.TypeOf(expectedType) {
		return true
	}

	return fail(t, fmt.Sprintf("Object expected to be of type %T, but was %T", expectedType, object), msgAndArgs...)
}

// JSONEq asserts that two JSON strings are equivalent.
func JSONEq(t testing.TB, expected, actual string, msgAndArgs ...any) bool {
	t.Helper()

	var expObj, actObj any
	if err := json.Unmarshal([]byte(expected), &expObj); err != nil {
		return fail(t, fmt.Sprintf("Expected value is not valid JSON: %s", err), msgAndArgs...)
	}

	if err := json.Unmarshal([]byte(actual), &actObj); err != nil {
		return fail(t, fmt.Sprintf("Actual value is not valid JSON: %s", err), msgAndArgs...)
	}

	return Equal(t, expObj, actObj, msgAndArgs...)
}

// Regexp asserts that a string matches a regular expression.
func Regexp(t testing.TB, rx, str any, msgAndArgs ...any) bool {
	t.Helper()

	var regex *regexp.Regexp
	switch r := rx.(type) {
	case string:
		var err error
		regex, err = regexp.Compile(r)
		if err != nil {
			return fail(t, fmt.Sprintf("Invalid regexp %q: %s", r, err), msgAndArgs...)
		}
	case *regexp.Regexp:
		regex = r
	default:
		return fail(t, fmt.Sprintf("Invalid regexp type: %T", rx), msgAndArgs...)
	}

	s, ok := str.(string)
	if !ok {
		return fail(t, fmt.Sprintf("Invalid string type: %T", str), msgAndArgs...)
	}

	if regex.MatchString(s) {
		return true
	}

	return fail(t, fmt.Sprintf("%q does not match %q", s, regex.String()), msgAndArgs...)
}

// Same asserts that two pointers reference the same object.
func Same(t testing.TB, expected, actual any, msgAndArgs ...any) bool {
	t.Helper()

	if expected == actual {
		return true
	}

	return fail(t, fmt.Sprintf("Not same pointer: expected %p, got %p", expected, actual), msgAndArgs...)
}

// WithinDuration asserts that two times are within delta of each other.
func WithinDuration(t testing.TB, expected, actual time.Time, delta time.Duration, msgAndArgs ...any) bool {
	t.Helper()

	diff := actual.Sub(expected)
	if diff < 0 {
		diff = -diff
	}

	if diff <= delta {
		return true
	}

	return fail(t, fmt.Sprintf("Max difference between %v and %v allowed is %v, but difference was %v", expected, actual, delta, diff), msgAndArgs...)
}

// FileExists asserts that a file exists.
func FileExists(t testing.TB, filepath string, msgAndArgs ...any) bool {
	t.Helper()

	info, err := os.Stat(filepath)
	if err == nil && !info.IsDir() {
		return true
	}

	return fail(t, fmt.Sprintf("File %q does not exist", filepath), msgAndArgs...)
}

// NoFileExists asserts that a file does not exist.
func NoFileExists(t testing.TB, filepath string, msgAndArgs ...any) bool {
	t.Helper()

	_, err := os.Stat(filepath)
	if os.IsNotExist(err) {
		return true
	}

	return fail(t, fmt.Sprintf("File %q unexpectedly exists", filepath), msgAndArgs...)
}

// NoDirExists asserts that a directory does not exist.
func NoDirExists(t testing.TB, dirpath string, msgAndArgs ...any) bool {
	t.Helper()

	info, err := os.Stat(dirpath)
	if os.IsNotExist(err) || (err == nil && !info.IsDir()) {
		return true
	}

	return fail(t, fmt.Sprintf("Directory %q unexpectedly exists", dirpath), msgAndArgs...)
}

// Panics asserts that the function panics.
func Panics(t testing.TB, f func(), msgAndArgs ...any) bool {
	t.Helper()

	didPanic := false
	func() {
		defer func() {
			if recover() != nil {
				didPanic = true
			}
		}()
		f()
	}()

	if didPanic {
		return true
	}

	return fail(t, "Func should panic", msgAndArgs...)
}

// PanicsWithError asserts that the function panics with specific error string.
func PanicsWithError(t testing.TB, errString string, f func(), msgAndArgs ...any) bool {
	t.Helper()

	var panicVal any
	didPanic := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				didPanic = true
				panicVal = r
			}
		}()
		f()
	}()

	if !didPanic {
		return fail(t, "Func should panic", msgAndArgs...)
	}

	pStr := fmt.Sprint(panicVal)
	if err, ok := panicVal.(error); ok {
		pStr = err.Error()
	}

	if pStr == errString {
		return true
	}

	return fail(t, fmt.Sprintf("Panic message mismatch:\nexpected: %q\nactual  : %q", errString, pStr), msgAndArgs...)
}

// NotPanics asserts that the function does not panic.
func NotPanics(t testing.TB, f func(), msgAndArgs ...any) bool {
	t.Helper()

	var panicVal any
	didPanic := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				didPanic = true
				panicVal = r
			}
		}()
		f()
	}()

	if !didPanic {
		return true
	}

	return fail(t, fmt.Sprintf("Func should not panic, but did with: %#v", panicVal), msgAndArgs...)
}

// Eventually asserts that condition becomes true within waitFor duration, polling every tick.
func Eventually(t testing.TB, condition func() bool, waitFor, tick time.Duration, msgAndArgs ...any) bool {
	t.Helper()

	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	timer := time.NewTimer(waitFor)
	defer timer.Stop()

	for {
		if condition() {
			return true
		}

		select {
		case <-timer.C:
			return fail(t, fmt.Sprintf("Condition never satisfied within %v", waitFor), msgAndArgs...)
		case <-ticker.C:
		}
	}
}

// Never asserts that condition never becomes true within waitFor duration, polling every tick.
func Never(t testing.TB, condition func() bool, waitFor, tick time.Duration, msgAndArgs ...any) bool {
	t.Helper()

	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	timer := time.NewTimer(waitFor)
	defer timer.Stop()

	for {
		if condition() {
			return fail(t, fmt.Sprintf("Condition was satisfied inside %v", waitFor), msgAndArgs...)
		}

		select {
		case <-timer.C:
			return true
		case <-ticker.C:
		}
	}
}
