// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package require

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

type dummyStruct struct {
	Public string
	unexp  int
}

type customErr struct{ msg string }

func (c *customErr) Error() string { return c.msg }

func TestRequire_AllAssertions(t *testing.T) {
	t.Parallel()
	mock := &mockTB{}

	// 1. Equal & Equalf & EqualValues & EqualValuesf & EqualExportedValues & EqualExportedValuesf & NotEqual & NotEqualf
	mock.reset()
	Equal(mock, 1, 1)
	Equalf(mock, 1, 1, "msg")
	EqualValues(mock, int(1), int32(1))
	EqualValuesf(mock, int(1), int32(1), "msg")
	s1 := dummyStruct{Public: "a", unexp: 1}
	s2 := dummyStruct{Public: "a", unexp: 2}
	EqualExportedValues(mock, s1, s2)
	EqualExportedValuesf(mock, s1, s2, "msg")
	NotEqual(mock, 1, 2)
	NotEqualf(mock, 1, 2, "msg")
	if mock.failed {
		t.Fatalf("unexpected failure in equal assertions")
	}

	// 2. True, Truef, False, Falsef, Nil, Nilf, NotNil, NotNilf
	mock.reset()
	True(mock, true)
	Truef(mock, true, "msg")
	False(mock, false)
	Falsef(mock, false, "msg")
	Nil(mock, nil)
	Nilf(mock, nil, "msg")
	val := 42
	NotNil(mock, &val)
	NotNilf(mock, &val, "msg")
	if mock.failed {
		t.Fatalf("unexpected failure in bool/nil assertions")
	}

	// 3. Error assertions
	mock.reset()
	NoError(mock, nil)
	NoErrorf(mock, nil, "msg")
	err := errors.New("sample error")
	Error(mock, err)
	Errorf(mock, err, "msg")
	EqualError(mock, err, "sample error")
	EqualErrorf(mock, err, "sample error", "msg")
	targetErr := errors.New("target")
	wrappedErr := fmt.Errorf("wrap: %w", targetErr)
	ErrorIs(mock, wrappedErr, targetErr)
	ErrorIsf(mock, wrappedErr, targetErr, "msg")
	NotErrorIs(mock, wrappedErr, errors.New("other"))
	NotErrorIsf(mock, wrappedErr, errors.New("other"), "msg")
	cErr := &customErr{msg: "custom"}
	wCustom := fmt.Errorf("wrap: %w", cErr)
	var asTarget *customErr
	ErrorAs(mock, wCustom, &asTarget)
	ErrorAsf(mock, wCustom, &asTarget, "msg")
	ErrorContains(mock, wrappedErr, "target")
	ErrorContainsf(mock, wrappedErr, "target", "msg")
	if mock.failed {
		t.Fatalf("unexpected failure in error assertions")
	}

	// 4. Collection assertions
	mock.reset()
	Contains(mock, "hello world", "world")
	Containsf(mock, "hello world", "world", "msg")
	NotContains(mock, "hello", "xyz")
	NotContainsf(mock, "hello", "xyz", "msg")
	Len(mock, "abcd", 4)
	Lenf(mock, "abcd", 4, "msg")
	Empty(mock, "")
	Emptyf(mock, "", "msg")
	NotEmpty(mock, "hello")
	NotEmptyf(mock, "hello", "msg")
	Zero(mock, 0)
	Zerof(mock, 0, "msg")
	NotZero(mock, 42)
	NotZerof(mock, 42, "msg")
	if mock.failed {
		t.Fatalf("unexpected failure in collection assertions")
	}

	// 5. Comparison & Math
	mock.reset()
	Greater(mock, 10, 5)
	Greaterf(mock, 10, 5, "msg")
	GreaterOrEqual(mock, 10, 10)
	GreaterOrEqualf(mock, 10, 10, "msg")
	Less(mock, 5, 10)
	Lessf(mock, 5, 10, "msg")
	LessOrEqual(mock, 5, 5)
	LessOrEqualf(mock, 5, 5, "msg")
	InDelta(mock, 1.0, 1.05, 0.1)
	InDeltaf(mock, 1.0, 1.05, 0.1, "msg")
	IsType(mock, int(0), int(42))
	IsTypef(mock, int(0), int(42), "msg")
	JSONEq(mock, `{"a":1}`, `{"a":1}`)
	JSONEqf(mock, `{"a":1}`, `{"a":1}`, "msg")
	Regexp(mock, `^\d+$`, "123")
	Regexpf(mock, regexp.MustCompile(`^\d+$`), "123", "msg")
	p1 := &val
	p2 := &val
	p3 := new(int)
	Same(mock, p1, p2)
	Samef(mock, p1, p2, "msg")
	NotSame(mock, p1, p3)
	NotSamef(mock, p1, p3, "msg")
	now := time.Now()
	WithinDuration(mock, now, now.Add(10*time.Millisecond), 50*time.Millisecond)
	WithinDurationf(mock, now, now.Add(10*time.Millisecond), 50*time.Millisecond, "msg")
	if mock.failed {
		t.Fatalf("unexpected failure in comparison assertions")
	}

	// 6. Files & System
	tmpFile, err := os.CreateTemp("", "require_test_*")
	if err != nil {
		t.Fatalf("temp file creation failed: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	mock.reset()
	FileExists(mock, tmpFile.Name())
	FileExistsf(mock, tmpFile.Name(), "msg")
	NoFileExists(mock, tmpFile.Name()+"_nonexistent")
	NoFileExistsf(mock, tmpFile.Name()+"_nonexistent", "msg")
	NoDirExists(mock, tmpFile.Name()+"_nonexistent_dir")
	NoDirExistsf(mock, tmpFile.Name()+"_nonexistent_dir", "msg")
	if mock.failed {
		t.Fatalf("unexpected failure in file assertions")
	}

	// 7. Panics, Concurrency & Elements
	mock.reset()
	Panics(mock, func() { panic("boom") })
	Panicsf(mock, func() { panic("boom") }, "msg")
	PanicsWithError(mock, "boom", func() { panic("boom") })
	PanicsWithErrorf(mock, "boom", func() { panic("boom") }, "msg")
	NotPanics(mock, func() {})
	NotPanicsf(mock, func() {}, "msg")
	Eventually(mock, func() bool { return true }, 50*time.Millisecond, 10*time.Millisecond)
	Eventuallyf(mock, func() bool { return true }, 50*time.Millisecond, 10*time.Millisecond, "msg")
	Never(mock, func() bool { return false }, 30*time.Millisecond, 10*time.Millisecond)
	Neverf(mock, func() bool { return false }, 30*time.Millisecond, 10*time.Millisecond, "msg")
	ElementsMatch(mock, []int{1, 2, 3}, []int{3, 2, 1})
	ElementsMatchf(mock, []int{1, 2}, []int{2, 1}, "msg")
	if mock.failed {
		t.Fatalf("unexpected failure in panic/elements assertions")
	}
}

func TestRequire_Failures(t *testing.T) {
	t.Parallel()
	mock := &mockTB{}

	mock.reset()
	Equal(mock, 1, 2)
	Equalf(mock, 1, 2, "msg")
	EqualValues(mock, 1, "2")
	EqualValuesf(mock, 1, "2", "msg")
	EqualExportedValues(mock, dummyStruct{Public: "a"}, dummyStruct{Public: "b"})
	EqualExportedValuesf(mock, dummyStruct{Public: "a"}, dummyStruct{Public: "b"}, "msg")
	NotEqual(mock, 1, 1)
	NotEqualf(mock, 1, 1, "msg")
	True(mock, false)
	Truef(mock, false, "msg")
	False(mock, true)
	Falsef(mock, true, "msg")
	Nil(mock, 1)
	Nilf(mock, 1, "msg")
	NotNil(mock, nil)
	NotNilf(mock, nil, "msg")
	NoError(mock, errors.New("err"))
	NoErrorf(mock, errors.New("err"), "msg")
	Error(mock, nil)
	Errorf(mock, nil, "msg")
	EqualError(mock, nil, "err")
	EqualErrorf(mock, nil, "err", "msg")
	ErrorIs(mock, errors.New("a"), errors.New("b"))
	ErrorIsf(mock, errors.New("a"), errors.New("b"), "msg")
	NotErrorIs(mock, AnError, AnError)
	NotErrorIsf(mock, AnError, AnError, "msg")
	ErrorAs(mock, errors.New("a"), new(*customErr))
	ErrorAsf(mock, errors.New("a"), new(*customErr), "msg")
	ErrorContains(mock, nil, "sub")
	ErrorContainsf(mock, nil, "sub", "msg")
	Contains(mock, "a", "b")
	Containsf(mock, "a", "b", "msg")
	NotContains(mock, "ab", "b")
	NotContainsf(mock, "ab", "b", "msg")
	Len(mock, "a", 2)
	Lenf(mock, "a", 2, "msg")
	Empty(mock, "a")
	Emptyf(mock, "a", "msg")
	NotEmpty(mock, "")
	NotEmptyf(mock, "", "msg")
	Zero(mock, 1)
	Zerof(mock, 1, "msg")
	NotZero(mock, 0)
	NotZerof(mock, 0, "msg")
	Greater(mock, 1, 2)
	Greaterf(mock, 1, 2, "msg")
	GreaterOrEqual(mock, 1, 2)
	GreaterOrEqualf(mock, 1, 2, "msg")
	Less(mock, 2, 1)
	Lessf(mock, 2, 1, "msg")
	LessOrEqual(mock, 2, 1)
	LessOrEqualf(mock, 2, 1, "msg")
	InDelta(mock, 1.0, 5.0, 0.1)
	InDeltaf(mock, 1.0, 5.0, 0.1, "msg")
	IsType(mock, 1, "s")
	IsTypef(mock, 1, "s", "msg")
	JSONEq(mock, "{}", "[]")
	JSONEqf(mock, "{}", "[]", "msg")
	Regexp(mock, `\d+`, "abc")
	Regexpf(mock, `\d+`, "abc", "msg")
	Same(mock, new(int), new(int))
	Samef(mock, new(int), new(int), "msg")
	n := 1
	NotSame(mock, &n, &n)
	NotSamef(mock, &n, &n, "msg")
	WithinDuration(mock, time.Now(), time.Now().Add(time.Hour), time.Second)
	WithinDurationf(mock, time.Now(), time.Now().Add(time.Hour), time.Second, "msg")
	FileExists(mock, "nonexistent_file_path_12345")
	FileExistsf(mock, "nonexistent_file_path_12345", "msg")
	NoFileExists(mock, os.Args[0])
	NoFileExistsf(mock, os.Args[0], "msg")
	NoDirExists(mock, os.TempDir())
	NoDirExistsf(mock, os.TempDir(), "msg")
	Panics(mock, func() {})
	Panicsf(mock, func() {}, "msg")
	PanicsWithError(mock, "err", func() {})
	PanicsWithErrorf(mock, "err", func() {}, "msg")
	NotPanics(mock, func() { panic("oops") })
	NotPanicsf(mock, func() { panic("oops") }, "msg")
	Eventually(mock, func() bool { return false }, 10*time.Millisecond, 5*time.Millisecond)
	Eventuallyf(mock, func() bool { return false }, 10*time.Millisecond, 5*time.Millisecond, "msg")
	Never(mock, func() bool { return true }, 10*time.Millisecond, 5*time.Millisecond)
	Neverf(mock, func() bool { return true }, 10*time.Millisecond, 5*time.Millisecond, "msg")
	ElementsMatch(mock, []int{1}, []int{2})
	ElementsMatchf(mock, []int{1}, []int{2}, "msg")

	if !mock.failed {
		t.Fatalf("expected failures for all failing assertions")
	}
}
