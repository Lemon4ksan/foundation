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

// Equal asserts that two objects are equal, failing immediately if not.
func Equal(t testing.TB, expected, actual any, msgAndArgs ...any) {
	t.Helper()
	if !assert.Equal(t, expected, actual, msgAndArgs...) {
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

// True asserts that value is true, failing immediately if false.
func True(t testing.TB, value bool, msgAndArgs ...any) {
	t.Helper()
	if !assert.True(t, value, msgAndArgs...) {
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

// Nil asserts that object is nil, failing immediately if not.
func Nil(t testing.TB, object any, msgAndArgs ...any) {
	t.Helper()
	if !assert.Nil(t, object, msgAndArgs...) {
		t.FailNow()
	}
}

// NotNil asserts that object is not nil, failing immediately if nil.
func NotNil(t testing.TB, object any, msgAndArgs ...any) {
	t.Helper()
	if !assert.NotNil(t, object, msgAndArgs...) {
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

// Error asserts that err is non-nil, failing immediately if nil.
func Error(t testing.TB, err error, msgAndArgs ...any) {
	t.Helper()
	if !assert.Error(t, err, msgAndArgs...) {
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

// ErrorContains asserts that err contains substring, failing immediately if not.
func ErrorContains(t testing.TB, err error, contains string, msgAndArgs ...any) {
	t.Helper()
	if !assert.ErrorContains(t, err, contains, msgAndArgs...) {
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

// NotContains asserts container does not contain element, failing immediately if it does.
func NotContains(t testing.TB, container, element any, msgAndArgs ...any) {
	t.Helper()
	if !assert.NotContains(t, container, element, msgAndArgs...) {
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

// Empty asserts object is empty, failing immediately if not.
func Empty(t testing.TB, object any, msgAndArgs ...any) {
	t.Helper()
	if !assert.Empty(t, object, msgAndArgs...) {
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

// Zero asserts object is its type zero value, failing immediately if not.
func Zero(t testing.TB, object any, msgAndArgs ...any) {
	t.Helper()
	if !assert.Zero(t, object, msgAndArgs...) {
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

// Greater asserts e1 > e2, failing immediately if not.
func Greater[T assert.Ordered](t testing.TB, e1, e2 T, msgAndArgs ...any) {
	t.Helper()
	if !assert.Greater(t, e1, e2, msgAndArgs...) {
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

// Less asserts e1 < e2, failing immediately if not.
func Less[T assert.Ordered](t testing.TB, e1, e2 T, msgAndArgs ...any) {
	t.Helper()
	if !assert.Less(t, e1, e2, msgAndArgs...) {
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

// Panics asserts that f panics, failing immediately if it does not.
func Panics(t testing.TB, f func(), msgAndArgs ...any) {
	t.Helper()
	if !assert.Panics(t, f, msgAndArgs...) {
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

// Eventually asserts condition becomes true within waitFor duration, failing immediately if not.
func Eventually(t testing.TB, condition func() bool, waitFor, tick time.Duration, msgAndArgs ...any) {
	t.Helper()
	if !assert.Eventually(t, condition, waitFor, tick, msgAndArgs...) {
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
