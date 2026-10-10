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

// TestingT is an interface for annotating test helper functions.
type TestingT interface {
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
//
// The zero value is empty and ready to use.
// Read-only operations are safe for concurrent use; mutations require external synchronization unless stated otherwise.
// Note that methods on [Controller] manage their own internal locks.
type Controller struct {
	T        TestReporter
	mu       sync.Mutex
	expected []*Call
	finished bool
}

// NewController creates a new mock controller.
//
// Executes in O(1) time and allocates memory.
func NewController(t any) *Controller {
	tr := wrapReporter(t)
	ctrl := &Controller{T: tr}

	if tb, ok := t.(testing.TB); ok {
		tb.Cleanup(ctrl.Finish)
	}

	return ctrl
}

// Finish verifies all expected calls were made.
//
// Executes in O(N) time where N is the number of expected calls.
func (c *Controller) Finish() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.finished {
		return
	}

	c.finished = true

	for _, call := range c.expected {
		if !call.satisfied() {
			if c.T != nil {
				c.T.Helper()
				c.T.Errorf(
					"missing expected call: %v (expected %d times, actual %d times)",
					call,
					call.minCalls,
					call.actualCalls,
				)
			}
		}
	}
}

// Satisfied reports whether all recorded expectations have been satisfied.
//
// Executes in O(N) time where N is the number of expected calls and is zero-allocation.
func (c *Controller) Satisfied() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, call := range c.expected {
		if !call.satisfied() {
			return false
		}
	}

	return true
}

// RecordCall registers an expected method invocation.
//
// Executes in O(1) time and allocates memory.
func (c *Controller) RecordCall(receiver any, method string, args ...any) *Call {
	c.mu.Lock()
	defer c.mu.Unlock()

	call := newCall(c, receiver, method, nil, args...)
	c.expected = append(c.expected, call)

	return call
}

// RecordCallWithMethodType registers an expected method invocation with reflection type info.
//
// Executes in O(1) time and allocates memory.
func (c *Controller) RecordCallWithMethodType(
	receiver any,
	method string,
	methodType reflect.Type,
	args ...any,
) *Call {
	c.mu.Lock()
	defer c.mu.Unlock()

	call := newCall(c, receiver, method, methodType, args...)
	c.expected = append(c.expected, call)

	return call
}

// Call executes an invoked mock method against registered expectations.
//
// Executes in O(N) time where N is the number of expected calls.
func (c *Controller) Call(receiver any, method string, args ...any) []any {
	var targetCall *Call

	c.mu.Lock()

	for _, call := range c.expected {
		if call.matches(receiver, method, args) {
			targetCall = call
			targetCall.actualCalls++
			break
		}
	}

	c.mu.Unlock()

	if targetCall != nil {
		return targetCall.execute(args)
	}

	if c.T != nil {
		c.T.Helper()
		c.T.Fatalf("unexpected call: %T.%s(%v)", receiver, method, args)
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
//
// Executes in O(N) time where N is the number of calls.
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
