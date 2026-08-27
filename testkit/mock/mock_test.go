// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mock

import (
	"fmt"
	"reflect"
	"testing"
)

type testService interface {
	DoWork(msg string, count int) (int, error)
	SetStatus(status *string)
	Ping()
}

type mockTestService struct {
	ctrl     *Controller
	recorder *mockTestServiceRecorder
}

type mockTestServiceRecorder struct {
	mock *mockTestService
}

func newMockTestService(ctrl *Controller) *mockTestService {
	mock := &mockTestService{ctrl: ctrl}
	mock.recorder = &mockTestServiceRecorder{mock: mock}

	return mock
}

func (m *mockTestService) EXPECT() *mockTestServiceRecorder {
	return m.recorder
}

func (m *mockTestService) DoWork(msg string, count int) (int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DoWork", msg, count)
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].(error)

	return ret0, ret1
}

func (mr *mockTestServiceRecorder) DoWork(msg, count any) *Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(
		mr.mock,
		"DoWork",
		reflect.TypeOf((*mockTestService)(nil).DoWork),
		msg,
		count,
	)
}

func (m *mockTestService) SetStatus(status *string) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetStatus", status)
}

func (mr *mockTestServiceRecorder) SetStatus(status any) *Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCall(mr.mock, "SetStatus", status)
}

func (m *mockTestService) Ping() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Ping")
}

func (mr *mockTestServiceRecorder) Ping() *Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCall(mr.mock, "Ping")
}

type nonTestReporter struct {
	errors []string
	fatals []string
}

func (c *nonTestReporter) Errorf(format string, args ...any) {
	c.errors = append(c.errors, fmt.Sprintf(format, args...))
}

func (c *nonTestReporter) Fatalf(format string, args ...any) {
	c.fatals = append(c.fatals, fmt.Sprintf(format, args...))
}

func (c *nonTestReporter) Helper() {}

func TestMockController_AllMatchers(t *testing.T) {
	t.Parallel()

	// 1. Any Matcher
	anyM := Any()
	if !anyM.Matches(123) || !anyM.Matches(nil) || anyM.String() != "is anything" {
		t.Fatalf("Any matcher failed")
	}

	// 2. Eq Matcher
	eqM := Eq("abc")
	if !eqM.Matches("abc") || eqM.Matches("xyz") || eqM.String() != "is equal to abc" {
		t.Fatalf("Eq matcher failed")
	}

	// 3. Nil Matcher
	nilM := Nil()
	var nilPtr *int
	var nilSlice []string
	var nilMap map[string]int
	var nilChan chan int
	var nilFunc func()
	if !nilM.Matches(nil) || !nilM.Matches(nilPtr) || !nilM.Matches(nilSlice) ||
		!nilM.Matches(nilMap) || !nilM.Matches(nilChan) || !nilM.Matches(nilFunc) {
		t.Fatalf("Nil matcher failed on nil values")
	}
	val := 10
	if nilM.Matches(&val) || nilM.Matches(10) || nilM.String() != "is nil" {
		t.Fatalf("Nil matcher failed on non-nil values")
	}

	// 4. Not Matcher
	notM := Not(eqM)
	if notM.Matches("abc") || !notM.Matches("xyz") {
		t.Fatalf("Not matcher failed with Matcher")
	}
	notVal := Not("abc")
	if notVal.Matches("abc") || !notVal.Matches("xyz") || notVal.String() != "not(is equal to abc)" {
		t.Fatalf("Not matcher failed with value")
	}

	// 5. AssignableToTypeOf Matcher
	assignM := AssignableToTypeOf(reflect.TypeOf(0))
	if !assignM.Matches(42) || assignM.Matches("str") || assignM.Matches(nil) ||
		assignM.String() != "is assignable to int" {
		t.Fatalf("AssignableToTypeOf matcher failed with reflect.Type")
	}
	assignVal := AssignableToTypeOf(int(0))
	if !assignVal.Matches(42) || assignVal.Matches("str") {
		t.Fatalf("AssignableToTypeOf matcher failed with sample value")
	}

	// 6. Len Matcher
	lenM := Len(3)
	if !lenM.Matches("abc") || !lenM.Matches([]int{1, 2, 3}) ||
		!lenM.Matches([3]int{1, 2, 3}) || !lenM.Matches(map[int]int{1: 1, 2: 2, 3: 3}) {
		t.Fatalf("Len matcher failed on valid items")
	}
	if lenM.Matches("ab") || lenM.Matches(nil) || lenM.Matches(123) || lenM.String() != "has length 3" {
		t.Fatalf("Len matcher failed on invalid items")
	}
}

func TestMockController_CallFeatures(t *testing.T) {
	t.Parallel()

	rep := &nonTestReporter{}
	ctrl := NewController(rep)
	mock := newMockTestService(ctrl)

	// 1. SetArg
	var updatedStatus string
	mock.EXPECT().SetStatus(Any()).SetArg(0, "active").Times(1)
	mock.SetStatus(&updatedStatus)
	if updatedStatus != "active" {
		t.Fatalf("SetArg failed, got %q", updatedStatus)
	}

	// 2. Do & DoAndReturn
	didRun := false
	mock.EXPECT().Ping().Do(func() { didRun = true }).Times(1)
	mock.Ping()
	if !didRun {
		t.Fatalf("Do action was not executed")
	}

	mock.EXPECT().DoWork(Any(), Any()).DoAndReturn(func(msg string, count int) (int, error) {
		return count * 2, nil
	}).Times(1)
	res, err := mock.DoWork("hi", 10)
	if err != nil || res != 20 {
		t.Fatalf("DoAndReturn failed, got res=%d, err=%v", res, err)
	}

	// 3. MinTimes, MaxTimes, AnyTimes
	callMin := mock.EXPECT().Ping().MinTimes(2)
	if callMin.minCalls != 2 || callMin.maxCalls != -1 {
		t.Fatalf("MinTimes failed")
	}
	mock.Ping()
	mock.Ping()

	callMax := mock.EXPECT().Ping().MaxTimes(5)
	if callMax.minCalls != 0 || callMax.maxCalls != 5 {
		t.Fatalf("MaxTimes failed")
	}
	mock.Ping()

	callAny := mock.EXPECT().Ping().AnyTimes()
	if callAny.minCalls != 0 || callAny.maxCalls != -1 {
		t.Fatalf("AnyTimes failed")
	}
	_ = callAny.String()

	// 4. InOrder
	c1 := mock.EXPECT().Ping()
	c2 := mock.EXPECT().Ping()
	InOrder(c1, c2)
	mock.Ping()
	mock.Ping()

	// Test InOrder with wrapper struct and nil
	type callWrapper struct{ Call *Call }
	InOrder(&callWrapper{Call: c1}, nil, 123)

	// Finish
	ctrl.Finish()
	ctrl.Finish() // second call should be no-op
}

func TestMockController_UnexpectedAndUnsatisfied(t *testing.T) {
	t.Parallel()

	rep := &nonTestReporter{}
	ctrl := NewController(rep)
	mock := newMockTestService(ctrl)

	// 1. Unsatisfied expectation
	mock.EXPECT().DoWork("needed", 1).Return(1, nil).Times(2)
	mock.DoWork("needed", 1) // only 1 of 2 called

	if ctrl.Satisfied() {
		t.Fatalf("controller should not be satisfied")
	}

	ctrl.Finish()
	if len(rep.errors) == 0 {
		t.Fatalf("Finish should report missing expected calls")
	}

	// 2. Unexpected call
	mock.DoWork("unexpected", 999)
	if len(rep.fatals) == 0 {
		t.Fatalf("unexpected call should trigger Fatalf")
	}
}

type minimalReporter struct {
	errors []string
	fatals []string
}

func (m *minimalReporter) Errorf(format string, args ...any) {
	m.errors = append(m.errors, fmt.Sprintf(format, args...))
}

func (m *minimalReporter) Fatalf(format string, args ...any) {
	m.fatals = append(m.fatals, fmt.Sprintf(format, args...))
}

func TestWrappedReporter(t *testing.T) {
	t.Parallel()

	minRep := &minimalReporter{}
	wr := wrapReporter(minRep)
	wr.Helper()
	wr.Errorf("error %d", 1)
	wr.Fatalf("fatal %d", 2)

	if len(minRep.errors) != 1 || len(minRep.fatals) != 1 {
		t.Fatalf("wrappedReporter failed to forward calls")
	}
}
