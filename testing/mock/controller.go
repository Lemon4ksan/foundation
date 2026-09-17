// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mock

import (
	"reflect"
	"sync"
	"testing"
)

// TestReporter is the interface for error reporting and helper annotations in tests.
type TestReporter interface {
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
	Helper()
}

// TestHelper is an interface for annotating test helper functions.
type TestHelper interface {
	Helper()
}

type wrappedReporter struct {
	t any
}

func (w *wrappedReporter) Errorf(format string, args ...any) {
	if tr, ok := w.t.(interface{ Errorf(string, ...any) }); ok {
		tr.Errorf(format, args...)
	}
}

func (w *wrappedReporter) Fatalf(format string, args ...any) {
	if tr, ok := w.t.(interface{ Fatalf(string, ...any) }); ok {
		tr.Fatalf(format, args...)
	}
}

func (w *wrappedReporter) Helper() {
	if th, ok := w.t.(interface{ Helper() }); ok {
		th.Helper()
	}
}

func wrapReporter(t any) TestReporter {
	if tr, ok := t.(TestReporter); ok {
		return tr
	}

	return &wrappedReporter{t: t}
}

// Controller defines the mock lifecycle controller.
type Controller struct {
	T        TestReporter
	mu       sync.Mutex
	expected []*Call
	finished bool
}

// NewController creates a new mock controller.
func NewController(t any) *Controller {
	tr := wrapReporter(t)
	ctrl := &Controller{T: tr}

	if tb, ok := t.(testing.TB); ok {
		tb.Cleanup(ctrl.Finish)
	}

	return ctrl
}

// Finish verifies all expected calls were made.
func (ctrl *Controller) Finish() {
	ctrl.mu.Lock()
	defer ctrl.mu.Unlock()

	if ctrl.finished {
		return
	}

	ctrl.finished = true

	for _, call := range ctrl.expected {
		if !call.satisfied() {
			if ctrl.T != nil {
				ctrl.T.Helper()
				ctrl.T.Errorf(
					"missing expected call: %v (expected %d times, actual %d times)",
					call,
					call.minCalls,
					call.actualCalls,
				)
			}
		}
	}
}

// Satisfied returns true if all recorded expectations have been satisfied.
func (ctrl *Controller) Satisfied() bool {
	ctrl.mu.Lock()
	defer ctrl.mu.Unlock()

	for _, call := range ctrl.expected {
		if !call.satisfied() {
			return false
		}
	}

	return true
}

// RecordCall registers an expected method invocation.
func (ctrl *Controller) RecordCall(receiver any, method string, args ...any) *Call {
	ctrl.mu.Lock()
	defer ctrl.mu.Unlock()

	call := newCall(ctrl, receiver, method, nil, args...)
	ctrl.expected = append(ctrl.expected, call)

	return call
}

// RecordCallWithMethodType registers an expected method invocation with reflection type info.
func (ctrl *Controller) RecordCallWithMethodType(
	receiver any,
	method string,
	methodType reflect.Type,
	args ...any,
) *Call {
	ctrl.mu.Lock()
	defer ctrl.mu.Unlock()

	call := newCall(ctrl, receiver, method, methodType, args...)
	ctrl.expected = append(ctrl.expected, call)

	return call
}

// Call executes an invoked mock method against registered expectations.
func (ctrl *Controller) Call(receiver any, method string, args ...any) []any {
	var targetCall *Call

	ctrl.mu.Lock()

	for _, call := range ctrl.expected {
		if call.matches(receiver, method, args) {
			targetCall = call
			targetCall.actualCalls++
			break
		}
	}

	ctrl.mu.Unlock()

	if targetCall != nil {
		return targetCall.execute(args)
	}

	if ctrl.T != nil {
		ctrl.T.Helper()
		ctrl.T.Fatalf("unexpected call: %T.%s(%v)", receiver, method, args)
	}

	return []any{nil, nil, nil, nil}
}

func toCall(v any) *Call {
	if c, ok := v.(*Call); ok {
		return c
	}

	if v == nil {
		return nil
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}

	if rv.Kind() == reflect.Struct {
		field := rv.FieldByName("Call")
		if field.IsValid() {
			if c, ok := field.Interface().(*Call); ok {
				return c
			}
		}
	}

	return nil
}

// InOrder declares that the given calls must be matched in order.
func InOrder(calls ...any) {
	var callList []*Call

	for _, c := range calls {
		if cl := toCall(c); cl != nil {
			callList = append(callList, cl)
		}
	}

	for i := 1; i < len(callList); i++ {
		callList[i].prereq = callList[i-1]
	}
}
