// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mock

import (
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

func TestMockController_Basic(t *testing.T) {
	ctrl := NewController(t)
	mock := newMockTestService(ctrl)

	mock.EXPECT().DoWork("hello", 5).Return(10, nil).Times(1)

	n, err := mock.DoWork("hello", 5)
	if err != nil || n != 10 {
		t.Fatalf("expected 10, nil; got %d, %v", n, err)
	}

	if !ctrl.Satisfied() {
		t.Fatalf("expected controller to be satisfied")
	}
}

func TestMockController_Matchers(t *testing.T) {
	ctrl := NewController(t)
	mock := newMockTestService(ctrl)

	mock.EXPECT().DoWork(Len(4), Eq(100)).Return(42, nil).AnyTimes()

	n, err := mock.DoWork("test", 100)
	if err != nil || n != 42 {
		t.Fatalf("expected 42, nil; got %d, %v", n, err)
	}
}

func TestMockController_SetArg(t *testing.T) {
	ctrl := NewController(t)
	mock := newMockTestService(ctrl)

	mock.EXPECT().SetStatus(Any()).SetArg(0, "active").Times(1)

	status := "pending"
	mock.SetStatus(&status)

	if status != "active" {
		t.Fatalf("expected status to be 'active', got '%s'", status)
	}
}

func TestMockController_InOrder(t *testing.T) {
	ctrl := NewController(t)
	mock := newMockTestService(ctrl)

	c1 := mock.EXPECT().Ping().Times(1)
	c2 := mock.EXPECT().DoWork("second", 2).Return(2, nil).Times(1)

	InOrder(c1, c2)

	mock.Ping()
	n, err := mock.DoWork("second", 2)
	if err != nil || n != 2 {
		t.Fatalf("expected 2, nil; got %d, %v", n, err)
	}
}
