// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomock_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/gomock"
)

type dummyReporter struct {
	failed bool
}

func (d *dummyReporter) Errorf(format string, args ...any) {
	d.failed = true
}

func (d *dummyReporter) Fatalf(format string, args ...any) {
	d.failed = true
}

func (d *dummyReporter) Helper() {}

func TestGomockMatchers(t *testing.T) {
	// Any matcher
	anyMatcher := gomock.Any()
	if !anyMatcher.Matches(123) || !anyMatcher.Matches("hello") || !anyMatcher.Matches(nil) {
		t.Fatal("Any() should match any value")
	}
	if anyMatcher.String() != "is anything" {
		t.Fatalf("unexpected string: %s", anyMatcher.String())
	}

	// Eq matcher
	eqMatcher := gomock.Eq(42)
	if !eqMatcher.Matches(42) || eqMatcher.Matches(43) {
		t.Fatal("Eq() matching failed")
	}

	// Nil matcher
	nilMatcher := gomock.Nil()
	var p *int
	if !nilMatcher.Matches(nil) || !nilMatcher.Matches(p) || nilMatcher.Matches(42) {
		t.Fatal("Nil() matching failed")
	}

	// Not matcher
	notNil := gomock.Not(nilMatcher)
	if notNil.Matches(nil) || !notNil.Matches(42) {
		t.Fatal("Not() matching failed")
	}

	// Len matcher
	lenMatcher := gomock.Len(3)
	if !lenMatcher.Matches([]int{1, 2, 3}) || lenMatcher.Matches([]int{1, 2}) {
		t.Fatal("Len() matching failed")
	}

	// AssignableToTypeOf matcher
	assignMatcher := gomock.AssignableToTypeOf(42)
	if !assignMatcher.Matches(100) || assignMatcher.Matches("string") {
		t.Fatal("AssignableToTypeOf() matching failed")
	}
}

func TestGomockController(t *testing.T) {
	rep := &dummyReporter{}
	ctrl := gomock.NewController(rep)
	if ctrl == nil {
		t.Fatal("expected non-nil Controller")
	}
	ctrl.Finish()
	if rep.failed {
		t.Fatal("controller reported unexpected error")
	}
}

func TestGomockInOrder(t *testing.T) {
	gomock.InOrder()
}

type threadSafeGomockReporter struct {
	mu     sync.Mutex
	errors []string
	fatals []string
}

func (r *threadSafeGomockReporter) Errorf(format string, args ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.errors = append(r.errors, fmt.Sprintf(format, args...))
}

func (r *threadSafeGomockReporter) Fatalf(format string, args ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.fatals = append(r.fatals, fmt.Sprintf(format, args...))
}

func (r *threadSafeGomockReporter) Helper() {}

func TestAdversarial_Gomock_DropInCompatibility(t *testing.T) {
	t.Parallel()

	rep := &threadSafeGomockReporter{}
	ctrl := gomock.NewController(rep)

	// Verify Controller type alias and methods
	assert.NotNil(t, ctrl)

	// Record an expected call
	receiver := "dummyService"
	call := ctrl.RecordCall(receiver, "Execute", gomock.Any(), gomock.Eq(100))
	call.Return("result", nil).Times(10)

	// Concurrent invocation through ctrl.Call
	const count = 10
	var wg sync.WaitGroup
	wg.Add(count)

	for i := range count {
		go func(idx int) {
			defer wg.Done()
			rets := ctrl.Call(receiver, "Execute", "param", 100)
			assert.Equal(t, "result", rets[0])
			assert.Nil(t, rets[1])
		}(i)
	}

	wg.Wait()
	assert.True(t, ctrl.Satisfied())
	ctrl.Finish()

	assert.Equal(t, 0, len(rep.errors))
	assert.Equal(t, 0, len(rep.fatals))
}

func TestAdversarial_Gomock_VerificationFailure(t *testing.T) {
	t.Parallel()

	rep := &threadSafeGomockReporter{}
	ctrl := gomock.NewController(rep)

	receiver := "dummy"
	ctrl.RecordCall(receiver, "MustBeCalled").Times(1)

	// Don't call -> Finish() should report Errorf
	ctrl.Finish()

	rep.mu.Lock()
	defer rep.mu.Unlock()
	assert.Equal(t, 1, len(rep.errors))
	assert.Contains(t, rep.errors[0], "missing expected call")
}
