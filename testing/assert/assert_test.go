// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package assert

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"
)

type mockTB struct {
	testing.TB
	failed   bool
	messages []string
}

func (m *mockTB) Helper() {}

func (m *mockTB) Errorf(format string, args ...any) {
	m.failed = true
	m.messages = append(m.messages, fmt.Sprintf(format, args...))
}

func (m *mockTB) FailNow() {
	m.failed = true
}

func (m *mockTB) reset() {
	m.failed = false
	m.messages = nil
}

type dummyExported struct {
	Public string
	unexp  int
}

type customErr struct{ msg string }

func (c *customErr) Error() string { return c.msg }

type otherErr struct{ msg string }

func (o *otherErr) Error() string { return o.msg }

func TestAssert_Equal_And_EqualValues(t *testing.T) {
	t.Parallel()
	mock := &mockTB{}

	// 1. Equal & Equalf
	mock.reset()
	if !Equal(mock, "hello", "hello") || mock.failed {
		t.Fatalf("Equal failed on identical strings")
	}
	if !Equalf(mock, "hello", "hello", "msg %d", 1) || mock.failed {
		t.Fatalf("Equalf failed on identical strings")
	}
	if !Equal(mock, []byte("abc"), []byte("abc")) || mock.failed {
		t.Fatalf("Equal failed on identical byte slices")
	}

	// Exported structs
	s1 := dummyExported{Public: "test", unexp: 1}
	s2 := dummyExported{Public: "test", unexp: 2}
	if !Equal(mock, s1, s2) || mock.failed {
		t.Fatalf("Equal should match structs on exported fields")
	}
	if !Equal(mock, &s1, &s2) || mock.failed {
		t.Fatalf("Equal should match struct pointers on exported fields")
	}

	mock.reset()
	var nilStruct1, nilStruct2 *dummyExported
	if !Equal(mock, nilStruct1, nilStruct2) || mock.failed {
		t.Fatalf("Equal should match nil pointers")
	}

	// Non-equal
	mock.reset()
	if Equal(mock, "hello", "world", "custom fail msg") || !mock.failed {
		t.Fatalf("Equal should fail on different strings")
	}

	// 2. EqualValues & EqualValuesf
	mock.reset()
	type MyInt int
	if !EqualValues(mock, int(42), MyInt(42)) || mock.failed {
		t.Fatalf("EqualValues failed on convertible types")
	}
	if !EqualValuesf(mock, int(42), MyInt(42), "convertible %s", "ok") || mock.failed {
		t.Fatalf("EqualValuesf failed")
	}
	if !EqualValues(mock, nil, nil) || mock.failed {
		t.Fatalf("EqualValues failed on nil")
	}
	mock.reset()
	if EqualValues(mock, nil, 42) || !mock.failed {
		t.Fatalf("EqualValues should fail nil vs non-nil")
	}
	mock.reset()
	if EqualValues(mock, "string", 42) || !mock.failed {
		t.Fatalf("EqualValues should fail non-convertible types")
	}

	// 3. EqualExportedValues & EqualExportedValuesf
	mock.reset()
	if !EqualExportedValues(mock, s1, s2) || mock.failed {
		t.Fatalf("EqualExportedValues failed")
	}
	if !EqualExportedValuesf(mock, s1, s2, "fmt") || mock.failed {
		t.Fatalf("EqualExportedValuesf failed")
	}

	// 4. NotEqual & NotEqualf
	mock.reset()
	if !NotEqual(mock, 10, 20) || mock.failed {
		t.Fatalf("NotEqual failed")
	}
	if !NotEqualf(mock, 10, 20, "not equal %s", "check") || mock.failed {
		t.Fatalf("NotEqualf failed")
	}
	mock.reset()
	if NotEqual(mock, 10, 10) || !mock.failed {
		t.Fatalf("NotEqual should fail on identical items")
	}
}

func TestAssert_Booleans_And_Nils(t *testing.T) {
	t.Parallel()
	mock := &mockTB{}

	// True & Truef
	mock.reset()
	if !True(mock, true) || mock.failed {
		t.Fatalf("True failed")
	}
	if !Truef(mock, true, "msg") || mock.failed {
		t.Fatalf("Truef failed")
	}
	mock.reset()
	if True(mock, false) || !mock.failed {
		t.Fatalf("True should fail on false")
	}

	// False & Falsef
	mock.reset()
	if !False(mock, false) || mock.failed {
		t.Fatalf("False failed")
	}
	if !Falsef(mock, false, "msg") || mock.failed {
		t.Fatalf("Falsef failed")
	}
	mock.reset()
	if False(mock, true) || !mock.failed {
		t.Fatalf("False should fail on true")
	}

	// Nil & Nilf
	mock.reset()
	var nilPtr *int
	var nilChan chan int
	var nilFunc func()
	var nilMap map[string]int
	var nilSlice []int
	if !Nil(mock, nil) || !Nil(mock, nilPtr) || !Nil(mock, nilChan) ||
		!Nil(mock, nilFunc) || !Nil(mock, nilMap) || !Nil(mock, nilSlice) || mock.failed {
		t.Fatalf("Nil failed on typed nils")
	}
	if !Nilf(mock, nil, "msg") || mock.failed {
		t.Fatalf("Nilf failed")
	}
	mock.reset()
	val := 10
	if Nil(mock, &val) || !mock.failed {
		t.Fatalf("Nil should fail on non-nil ptr")
	}
	mock.reset()
	if Nil(mock, 10) || !mock.failed {
		t.Fatalf("Nil should fail on non-nil value")
	}

	// NotNil & NotNilf
	mock.reset()
	if !NotNil(mock, &val) || mock.failed {
		t.Fatalf("NotNil failed on pointer")
	}
	if !NotNilf(mock, &val, "msg") || mock.failed {
		t.Fatalf("NotNilf failed")
	}
	mock.reset()
	if NotNil(mock, nil) || !mock.failed {
		t.Fatalf("NotNil should fail on nil")
	}
}

func TestAssert_Errors(t *testing.T) {
	t.Parallel()
	mock := &mockTB{}

	// NoError & NoErrorf
	mock.reset()
	if !NoError(mock, nil) || mock.failed {
		t.Fatalf("NoError failed on nil")
	}
	if !NoErrorf(mock, nil, "msg") || mock.failed {
		t.Fatalf("NoErrorf failed")
	}
	mock.reset()
	err := errors.New("sample error")
	if NoError(mock, err) || !mock.failed {
		t.Fatalf("NoError should fail on error")
	}

	// Error & Errorf
	mock.reset()
	if !Error(mock, err) || mock.failed {
		t.Fatalf("Error failed on error")
	}
	if !Errorf(mock, err, "msg") || mock.failed {
		t.Fatalf("Errorf failed")
	}
	mock.reset()
	if Error(mock, nil) || !mock.failed {
		t.Fatalf("Error should fail on nil")
	}

	// EqualError & EqualErrorf
	mock.reset()
	if !EqualError(mock, err, "sample error") || mock.failed {
		t.Fatalf("EqualError failed")
	}
	if !EqualErrorf(mock, err, "sample error", "msg") || mock.failed {
		t.Fatalf("EqualErrorf failed")
	}
	mock.reset()
	if EqualError(mock, nil, "sample error") || !mock.failed {
		t.Fatalf("EqualError should fail on nil")
	}
	mock.reset()
	if EqualError(mock, err, "wrong message") || !mock.failed {
		t.Fatalf("EqualError should fail on mismatch")
	}

	// ErrorIs & ErrorIsf & NotErrorIs & NotErrorIsf
	targetErr := errors.New("target error")
	wrappedErr := fmt.Errorf("wrapped: %w", targetErr)
	mock.reset()
	if !ErrorIs(mock, wrappedErr, targetErr) || mock.failed {
		t.Fatalf("ErrorIs failed")
	}
	if !ErrorIsf(mock, wrappedErr, targetErr, "msg") || mock.failed {
		t.Fatalf("ErrorIsf failed")
	}
	mock.reset()
	if ErrorIs(mock, wrappedErr, errors.New("other")) || !mock.failed {
		t.Fatalf("ErrorIs should fail on different error")
	}

	mock.reset()
	if !NotErrorIs(mock, wrappedErr, errors.New("other")) || mock.failed {
		t.Fatalf("NotErrorIs failed")
	}
	if !NotErrorIsf(mock, wrappedErr, errors.New("other"), "msg") || mock.failed {
		t.Fatalf("NotErrorIsf failed")
	}
	mock.reset()
	if NotErrorIs(mock, wrappedErr, targetErr) || !mock.failed {
		t.Fatalf("NotErrorIs should fail on matched error")
	}

	// ErrorAs & ErrorAsf
	cErr := &customErr{msg: "custom"}
	wCustom := fmt.Errorf("wrap: %w", cErr)
	var asTarget *customErr
	mock.reset()
	if !ErrorAs(mock, wCustom, &asTarget) || mock.failed {
		t.Fatalf("ErrorAs failed")
	}
	if !ErrorAsf(mock, wCustom, &asTarget, "msg") || mock.failed {
		t.Fatalf("ErrorAsf failed")
	}
	mock.reset()
	var oTarget *otherErr
	if ErrorAs(mock, wCustom, &oTarget) || !mock.failed {
		t.Fatalf("ErrorAs should fail on mismatch")
	}

	// ErrorContains & ErrorContainsf
	mock.reset()
	if !ErrorContains(mock, wrappedErr, "target") || mock.failed {
		t.Fatalf("ErrorContains failed")
	}
	if !ErrorContainsf(mock, wrappedErr, "target", "msg") || mock.failed {
		t.Fatalf("ErrorContainsf failed")
	}
	mock.reset()
	if ErrorContains(mock, nil, "target") || !mock.failed {
		t.Fatalf("ErrorContains should fail on nil")
	}
	mock.reset()
	if ErrorContains(mock, wrappedErr, "nonexistent") || !mock.failed {
		t.Fatalf("ErrorContains should fail on nonexistent substring")
	}
}

func TestAssert_Collections_And_Comparisons(t *testing.T) {
	t.Parallel()
	mock := &mockTB{}

	// Contains & Containsf
	mock.reset()
	if !Contains(mock, "hello world", "world") || mock.failed {
		t.Fatalf("Contains string failed")
	}
	if !Containsf(mock, "hello world", "world", "msg") || mock.failed {
		t.Fatalf("Containsf failed")
	}
	if !Contains(mock, []int{10, 20, 30}, 20) || mock.failed {
		t.Fatalf("Contains slice failed")
	}
	if !Contains(mock, map[string]int{"a": 1}, "a") || mock.failed {
		t.Fatalf("Contains map failed")
	}
	mock.reset()
	if Contains(mock, nil, "a") || !mock.failed {
		t.Fatalf("Contains nil container should fail")
	}
	mock.reset()
	if Contains(mock, "hello", "xyz") || !mock.failed {
		t.Fatalf("Contains string mismatch should fail")
	}

	// NotContains & NotContainsf
	mock.reset()
	if !NotContains(mock, nil, "a") || mock.failed {
		t.Fatalf("NotContains nil container should succeed")
	}
	if !NotContains(mock, "hello", "xyz") || mock.failed {
		t.Fatalf("NotContains string failed")
	}
	if !NotContainsf(mock, "hello", "xyz", "msg") || mock.failed {
		t.Fatalf("NotContainsf failed")
	}
	if !NotContains(mock, []int{1, 2, 3}, 99) || mock.failed {
		t.Fatalf("NotContains slice failed")
	}
	if !NotContains(mock, map[string]int{"a": 1}, "b") || mock.failed {
		t.Fatalf("NotContains map failed")
	}
	mock.reset()
	if NotContains(mock, "hello", "ell") || !mock.failed {
		t.Fatalf("NotContains string should fail")
	}
	mock.reset()
	if NotContains(mock, []int{1, 2, 3}, 2) || !mock.failed {
		t.Fatalf("NotContains slice should fail")
	}
	mock.reset()
	if NotContains(mock, map[string]int{"a": 1}, "a") || !mock.failed {
		t.Fatalf("NotContains map should fail")
	}

	// Len & Lenf
	mock.reset()
	if !Len(mock, "test", 4) || !Len(mock, []int{1, 2}, 2) || !Len(mock, map[string]int{"a": 1}, 1) || mock.failed {
		t.Fatalf("Len failed")
	}
	if !Lenf(mock, "test", 4, "msg") || mock.failed {
		t.Fatalf("Lenf failed")
	}
	mock.reset()
	if Len(mock, 123, 3) || !mock.failed {
		t.Fatalf("Len on non-len type should fail")
	}
	mock.reset()
	if Len(mock, "test", 5) || !mock.failed {
		t.Fatalf("Len mismatch should fail")
	}

	// Empty & Emptyf & NotEmpty & NotEmptyf
	mock.reset()
	if !Empty(mock, nil) || !Empty(mock, "") || !Empty(mock, []int{}) || !Empty(mock, 0) || mock.failed {
		t.Fatalf("Empty failed")
	}
	if !Emptyf(mock, "", "msg") || mock.failed {
		t.Fatalf("Emptyf failed")
	}
	mock.reset()
	if Empty(mock, "not empty") || !mock.failed {
		t.Fatalf("Empty should fail on non-empty string")
	}
	mock.reset()
	if Empty(mock, 42) || !mock.failed {
		t.Fatalf("Empty should fail on non-zero int")
	}

	mock.reset()
	if !NotEmpty(mock, "hello") || !NotEmpty(mock, []int{1}) || !NotEmpty(mock, 42) || mock.failed {
		t.Fatalf("NotEmpty failed")
	}
	if !NotEmptyf(mock, "hello", "msg") || mock.failed {
		t.Fatalf("NotEmptyf failed")
	}
	mock.reset()
	if NotEmpty(mock, nil) || !mock.failed {
		t.Fatalf("NotEmpty should fail on nil")
	}
	mock.reset()
	if NotEmpty(mock, "") || !mock.failed {
		t.Fatalf("NotEmpty should fail on empty string")
	}
	mock.reset()
	if NotEmpty(mock, 0) || !mock.failed {
		t.Fatalf("NotEmpty should fail on zero int")
	}

	// Zero & Zerof & NotZero & NotZerof
	mock.reset()
	if !Zero(mock, nil) || !Zero(mock, 0) || !Zero(mock, "") || mock.failed {
		t.Fatalf("Zero failed")
	}
	if !Zerof(mock, 0, "msg") || mock.failed {
		t.Fatalf("Zerof failed")
	}
	mock.reset()
	if Zero(mock, 100) || !mock.failed {
		t.Fatalf("Zero should fail on 100")
	}

	mock.reset()
	if !NotZero(mock, 100) || !NotZero(mock, "text") || mock.failed {
		t.Fatalf("NotZero failed")
	}
	if !NotZerof(mock, 100, "msg") || mock.failed {
		t.Fatalf("NotZerof failed")
	}
	mock.reset()
	if NotZero(mock, nil) || !mock.failed {
		t.Fatalf("NotZero should fail on nil")
	}
	mock.reset()
	if NotZero(mock, 0) || !mock.failed {
		t.Fatalf("NotZero should fail on 0")
	}

	// Greater, GreaterOrEqual, Less, LessOrEqual
	mock.reset()
	if !Greater(mock, 10, 5) || !Greaterf(mock, 10, 5, "msg") || mock.failed {
		t.Fatalf("Greater failed")
	}
	mock.reset()
	if Greater(mock, 5, 10) || !mock.failed {
		t.Fatalf("Greater should fail")
	}

	mock.reset()
	if !GreaterOrEqual(mock, 10, 10) || !GreaterOrEqual(mock, 10, 5) ||
		!GreaterOrEqualf(mock, 10, 10, "msg") || mock.failed {
		t.Fatalf("GreaterOrEqual failed")
	}
	mock.reset()
	if GreaterOrEqual(mock, 5, 10) || !mock.failed {
		t.Fatalf("GreaterOrEqual should fail")
	}

	mock.reset()
	if !Less(mock, 5, 10) || !Lessf(mock, 5, 10, "msg") || mock.failed {
		t.Fatalf("Less failed")
	}
	mock.reset()
	if Less(mock, 10, 5) || !mock.failed {
		t.Fatalf("Less should fail")
	}

	mock.reset()
	if !LessOrEqual(mock, 10, 10) || !LessOrEqual(mock, 5, 10) ||
		!LessOrEqualf(mock, 10, 10, "msg") || mock.failed {
		t.Fatalf("LessOrEqual failed")
	}
	mock.reset()
	if LessOrEqual(mock, 10, 5) || !mock.failed {
		t.Fatalf("LessOrEqual should fail")
	}

	// InDelta & InDeltaf
	mock.reset()
	if !InDelta(mock, 10.0, 10.05, 0.1) || !InDelta(mock, 10, 11, 2) ||
		!InDeltaf(mock, 10.0, 10.05, 0.1, "msg") || mock.failed {
		t.Fatalf("InDelta failed")
	}
	mock.reset()
	if InDelta(mock, "abc", 10.0, 0.1) || !mock.failed {
		t.Fatalf("InDelta should fail on non-number")
	}
	mock.reset()
	if InDelta(mock, 10.0, 20.0, 1.0) || !mock.failed {
		t.Fatalf("InDelta should fail when outside delta")
	}

	// IsType & IsTypef
	mock.reset()
	if !IsType(mock, int(0), int(42)) || !IsTypef(mock, int(0), int(42), "msg") || mock.failed {
		t.Fatalf("IsType failed")
	}
	mock.reset()
	if IsType(mock, int(0), "str") || !mock.failed {
		t.Fatalf("IsType should fail on type mismatch")
	}

	// JSONEq & JSONEqf
	mock.reset()
	j1 := `{"a": 1, "b": "two"}`
	j2 := `{"b": "two", "a": 1}`
	if !JSONEq(mock, j1, j2) || !JSONEqf(mock, j1, j2, "msg") || mock.failed {
		t.Fatalf("JSONEq failed")
	}
	mock.reset()
	if JSONEq(mock, "invalid json", j2) || !mock.failed {
		t.Fatalf("JSONEq should fail on invalid expected JSON")
	}
	mock.reset()
	if JSONEq(mock, j1, "invalid json") || !mock.failed {
		t.Fatalf("JSONEq should fail on invalid actual JSON")
	}

	// Regexp & Regexpf
	mock.reset()
	if !Regexp(mock, `^[a-z]+$`, "hello") || !Regexp(mock, regexp.MustCompile(`\d+`), "123") ||
		!Regexpf(mock, `\d+`, "123", "msg") || mock.failed {
		t.Fatalf("Regexp failed")
	}
	mock.reset()
	if Regexp(mock, `[invalid(regex`, "hello") || !mock.failed {
		t.Fatalf("Regexp should fail on invalid regex string")
	}
	mock.reset()
	if Regexp(mock, 123, "hello") || !mock.failed {
		t.Fatalf("Regexp should fail on invalid regex type")
	}
	mock.reset()
	if Regexp(mock, `\d+`, 123) || !mock.failed {
		t.Fatalf("Regexp should fail on non-string target")
	}
	mock.reset()
	if Regexp(mock, `^\d+$`, "abc") || !mock.failed {
		t.Fatalf("Regexp should fail on non-matching string")
	}

	// Same & Samef & NotSame & NotSamef
	targetVal := 42
	p1 := &targetVal
	p2 := &targetVal
	p3 := new(int)
	mock.reset()
	if !Same(mock, p1, p2) || !Samef(mock, p1, p2, "msg") || mock.failed {
		t.Fatalf("Same failed")
	}
	mock.reset()
	if Same(mock, p1, p3) || !mock.failed {
		t.Fatalf("Same should fail on different pointers")
	}

	mock.reset()
	if !NotSame(mock, p1, p3) || !NotSamef(mock, p1, p3, "msg") || mock.failed {
		t.Fatalf("NotSame failed")
	}
	mock.reset()
	if NotSame(mock, p1, p2) || !mock.failed {
		t.Fatalf("NotSame should fail on same pointers")
	}

	// WithinDuration & WithinDurationf
	now := time.Now()
	mock.reset()
	if !WithinDuration(mock, now, now.Add(50*time.Millisecond), 100*time.Millisecond) ||
		!WithinDurationf(mock, now, now.Add(50*time.Millisecond), 100*time.Millisecond, "msg") || mock.failed {
		t.Fatalf("WithinDuration failed")
	}
	mock.reset()
	if WithinDuration(mock, now, now.Add(200*time.Millisecond), 100*time.Millisecond) || !mock.failed {
		t.Fatalf("WithinDuration should fail")
	}

	// FileExists & FileExistsf & NoFileExists & NoFileExistsf & NoDirExists & NoDirExistsf
	tmpFile, err := os.CreateTemp("", "assert_test_*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	mock.reset()
	if !FileExists(mock, tmpFile.Name()) || !FileExistsf(mock, tmpFile.Name(), "msg") || mock.failed {
		t.Fatalf("FileExists failed")
	}
	mock.reset()
	nonExistent := tmpFile.Name() + "_nonexistent"
	if FileExists(mock, nonExistent) || !mock.failed {
		t.Fatalf("FileExists should fail on nonexistent file")
	}

	mock.reset()
	if !NoFileExists(mock, nonExistent) || !NoFileExistsf(mock, nonExistent, "msg") || mock.failed {
		t.Fatalf("NoFileExists failed")
	}
	mock.reset()
	if NoFileExists(mock, tmpFile.Name()) || !mock.failed {
		t.Fatalf("NoFileExists should fail on existing file")
	}

	mock.reset()
	if !NoDirExists(mock, nonExistent) || !NoDirExistsf(mock, nonExistent, "msg") || mock.failed {
		t.Fatalf("NoDirExists failed")
	}
	mock.reset()
	if NoDirExists(mock, os.TempDir()) || !mock.failed {
		t.Fatalf("NoDirExists should fail on existing dir")
	}

	// Panics & Panicsf & PanicsWithError & PanicsWithErrorf & NotPanics & NotPanicsf
	mock.reset()
	if !Panics(mock, func() { panic("boom") }) || !Panicsf(mock, func() { panic("boom") }, "msg") || mock.failed {
		t.Fatalf("Panics failed")
	}
	mock.reset()
	if Panics(mock, func() {}) || !mock.failed {
		t.Fatalf("Panics should fail if no panic")
	}

	mock.reset()
	if !PanicsWithError(mock, "boom", func() { panic(errors.New("boom")) }) ||
		!PanicsWithErrorf(mock, "boom", func() { panic("boom") }, "msg") || mock.failed {
		t.Fatalf("PanicsWithError failed")
	}
	mock.reset()
	if PanicsWithError(mock, "boom", func() {}) || !mock.failed {
		t.Fatalf("PanicsWithError should fail if no panic")
	}
	mock.reset()
	if PanicsWithError(mock, "boom", func() { panic("other") }) || !mock.failed {
		t.Fatalf("PanicsWithError should fail on message mismatch")
	}

	mock.reset()
	if !NotPanics(mock, func() {}) || !NotPanicsf(mock, func() {}, "msg") || mock.failed {
		t.Fatalf("NotPanics failed")
	}
	mock.reset()
	if NotPanics(mock, func() { panic("oops") }) || !mock.failed {
		t.Fatalf("NotPanics should fail on panic")
	}

	// Eventually & Eventuallyf & Never & Neverf
	mock.reset()
	count := 0
	fnTrue := func() bool { return true }
	if !Eventually(mock, func() bool {
		count++
		return count >= 2
	}, 100*time.Millisecond, 10*time.Millisecond) ||
		!Eventuallyf(mock, fnTrue, 50*time.Millisecond, 10*time.Millisecond, "msg") || mock.failed {
		t.Fatalf("Eventually failed")
	}
	mock.reset()
	fnFalse := func() bool { return false }
	if Eventually(mock, fnFalse, 20*time.Millisecond, 5*time.Millisecond) || !mock.failed {
		t.Fatalf("Eventually should fail on timeout")
	}

	mock.reset()
	if !Never(mock, fnFalse, 30*time.Millisecond, 10*time.Millisecond) ||
		!Neverf(mock, fnFalse, 30*time.Millisecond, 10*time.Millisecond, "msg") || mock.failed {
		t.Fatalf("Never failed")
	}
	mock.reset()
	if Never(mock, func() bool { return true }, 30*time.Millisecond, 10*time.Millisecond) || !mock.failed {
		t.Fatalf("Never should fail if condition becomes true")
	}

	// ElementsMatch & ElementsMatchf
	mock.reset()
	if !ElementsMatch(mock, []int{1, 2, 3}, []int{3, 2, 1}) || !ElementsMatch(mock, nil, nil) ||
		!ElementsMatchf(mock, []string{"a", "b"}, []string{"b", "a"}, "msg") || mock.failed {
		t.Fatalf("ElementsMatch failed")
	}
	mock.reset()
	if ElementsMatch(mock, []int{1}, nil) || !mock.failed {
		t.Fatalf("ElementsMatch should fail slice vs nil")
	}
	mock.reset()
	if ElementsMatch(mock, 123, []int{1}) || !mock.failed {
		t.Fatalf("ElementsMatch should fail on non-slice")
	}
	mock.reset()
	if ElementsMatch(mock, []int{1, 2}, []int{1, 2, 3}) || !mock.failed {
		t.Fatalf("ElementsMatch should fail on length difference")
	}
	mock.reset()
	if ElementsMatch(mock, []int{1, 2, 3}, []int{1, 2, 4}) || !mock.failed {
		t.Fatalf("ElementsMatch should fail on missing elements")
	}
}
