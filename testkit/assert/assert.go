// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package assert provides fast, zero-dependency testing assertions.
package assert

import (
	"errors"
	"fmt"
	"reflect"
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

	return fail(t, fmt.Sprintf("Not equal: \nexpected: %#v\nactual  : %#v", expected, actual), msgAndArgs...)
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
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
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

// ErrorIs asserts that target is in err's chain.
func ErrorIs(t testing.TB, err, target error, msgAndArgs ...any) bool {
	t.Helper()

	if errors.Is(err, target) {
		return true
	}

	return fail(t, fmt.Sprintf("Target error should be in err chain:\nexpected: %q\nin chain: %+v", target, err), msgAndArgs...)
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
