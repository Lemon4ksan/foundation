package mock_test

// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import (
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/mock"
)

// Service interface for adversarial testing.
type greeterService interface {
	Greet(name string, times int) (string, error)
	Reset(flag *bool)
	Noop()
}

type mockGreeter struct {
	ctrl     *mock.Controller
	recorder *mockGreeterRecorder
}

type mockGreeterRecorder struct {
	mock *mockGreeter
}

func newMockGreeter(ctrl *mock.Controller) *mockGreeter {
	m := &mockGreeter{ctrl: ctrl}
	m.recorder = &mockGreeterRecorder{mock: m}
	return m
}

func (m *mockGreeter) EXPECT() *mockGreeterRecorder {
	return m.recorder
}

func (m *mockGreeter) Greet(name string, times int) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Greet", name, times)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *mockGreeterRecorder) Greet(name, times any) *mock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(
		mr.mock,
		"Greet",
		reflect.TypeOf((*mockGreeter)(nil).Greet),
		name,
		times,
	)
}

func (m *mockGreeter) Reset(flag *bool) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Reset", flag)
}

func (mr *mockGreeterRecorder) Reset(flag any) *mock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCall(mr.mock, "Reset", flag)
}

func (m *mockGreeter) Noop() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Noop")
}

func (mr *mockGreeterRecorder) Noop() *mock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCall(mr.mock, "Noop")
}

// Thread-safe recorder implementing TestReporter
type threadSafeReporter struct {
	mu     sync.Mutex
	errors []string
	fatals []string
}

func (r *threadSafeReporter) Errorf(format string, args ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.errors = append(r.errors, fmt.Sprintf(format, args...))
}

func (r *threadSafeReporter) Fatalf(format string, args ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.fatals = append(r.fatals, fmt.Sprintf(format, args...))
}

func (r *threadSafeReporter) Helper() {}

func (r *threadSafeReporter) ErrorCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.errors)
}

func (r *threadSafeReporter) FatalCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.fatals)
}

// Minimal reporter implementing ONLY TestingT (Helper only, no Errorf or Fatalf)
type helperOnlyReporter struct {
	helperCalled atomic.Int32
}

func (h *helperOnlyReporter) Helper() {
	h.helperCalled.Add(1)
}

func TestAdversarial_Mock_HighConcurrencyRace(t *testing.T) {
	t.Parallel()

	rep := &threadSafeReporter{}
	ctrl := mock.NewController(rep)
	svc := newMockGreeter(ctrl)

	const totalCalls = 200
	svc.EXPECT().Greet(mock.Any(), mock.Any()).Return("ok", nil).Times(totalCalls)

	var wg sync.WaitGroup
	const goroutines = 20
	const perGoroutine = totalCalls / goroutines

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(gid int) {
			defer wg.Done()
			for i := 0; i < perGoroutine; i++ {
				res, err := svc.Greet(fmt.Sprintf("user-%d", gid), i)
				assert.NoError(t, err)
				assert.Equal(t, "ok", res)
			}
		}(g)
	}

	wg.Wait()
	assert.True(t, ctrl.Satisfied())
	ctrl.Finish()

	assert.Equal(t, 0, rep.ErrorCount())
	assert.Equal(t, 0, rep.FatalCount())
}

func TestAdversarial_Mock_ConcurrentSatisfiedAndInvocations(t *testing.T) {
	t.Parallel()

	rep := &threadSafeReporter{}
	ctrl := mock.NewController(rep)
	svc := newMockGreeter(ctrl)

	const totalCalls = 100
	svc.EXPECT().Noop().Times(totalCalls)

	var wg sync.WaitGroup
	stopCheck := make(chan struct{})

	// Concurrently query ctrl.Satisfied() while calls are actively firing
	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stopCheck:
					return
				default:
					_ = ctrl.Satisfied()
				}
			}
		}()
	}

	for range totalCalls {
		wg.Add(1)
		go func() {
			defer wg.Done()
			svc.Noop()
		}()
	}

	// Wait for all invocations to finish
	timeOut := make(chan struct{})
	go func() {
		// Wait for the totalCalls invocations
		// but since wg has checkers too, wait in a loop
		for !ctrl.Satisfied() {
			// spin wait
		}
		close(stopCheck)
		close(timeOut)
	}()

	<-timeOut
	wg.Wait()

	assert.True(t, ctrl.Satisfied())
	ctrl.Finish()
	assert.Equal(t, 0, rep.ErrorCount())
}

func TestAdversarial_Mock_TestingTInterface_NoErrorfOrFatalf(t *testing.T) {
	t.Parallel()

	// TestingT interface implementation with only Helper()
	helperRep := &helperOnlyReporter{}
	var tInterface mock.TestingT = helperRep

	ctrl := mock.NewController(tInterface)
	svc := newMockGreeter(ctrl)

	// Expectation that will NOT be satisfied
	svc.EXPECT().Greet("expected", 1).Return("hello", nil).Times(1)

	assert.False(t, ctrl.Satisfied())

	// Calling Finish() must not panic even though helperRep has no Errorf
	ctrl.Finish()

	// Unexpected call must not panic even though helperRep has no Fatalf
	res, err := svc.Greet("unexpected", 999)
	assert.Empty(t, res)
	assert.NoError(t, err)

	assert.True(t, helperRep.helperCalled.Load() > 0)
}

func TestAdversarial_Mock_NilReporter(t *testing.T) {
	t.Parallel()

	// Passing nil reporter must not panic
	ctrl := mock.NewController(nil)
	svc := newMockGreeter(ctrl)

	svc.EXPECT().Noop().Times(1)

	// Unsatisfied finish with nil reporter
	ctrl.Finish()

	// Unexpected call with nil reporter
	svc.Noop() // now satisfied
	svc.Noop() // now unexpected (exceeded maxCalls)
}

func TestAdversarial_Mock_VerificationFailures(t *testing.T) {
	t.Parallel()

	t.Run("MissingExpectedCall_ReportsErrorOnFinish", func(t *testing.T) {
		t.Parallel()
		rep := &threadSafeReporter{}
		ctrl := mock.NewController(rep)
		svc := newMockGreeter(ctrl)

		svc.EXPECT().Greet("must_call", 1).Return("ok", nil).Times(2)
		// Only call once
		_, _ = svc.Greet("must_call", 1)

		assert.False(t, ctrl.Satisfied())
		ctrl.Finish()

		assert.Equal(t, 1, rep.ErrorCount())
		assert.Contains(t, rep.errors[0], "missing expected call")
		assert.Contains(t, rep.errors[0], "expected 2 times, actual 1 times")
	})

	t.Run("ExceededCallCount_ReportsFatalOnExtraCall", func(t *testing.T) {
		t.Parallel()
		rep := &threadSafeReporter{}
		ctrl := mock.NewController(rep)
		svc := newMockGreeter(ctrl)

		svc.EXPECT().Noop().Times(1)
		svc.Noop() // 1st call: OK

		assert.Equal(t, 0, rep.FatalCount())

		svc.Noop() // 2nd call: Unexpected!
		assert.Equal(t, 1, rep.FatalCount())
		assert.Contains(t, rep.fatals[0], "unexpected call")
	})

	t.Run("MismatchedArgumentValues_ReportsFatal", func(t *testing.T) {
		t.Parallel()
		rep := &threadSafeReporter{}
		ctrl := mock.NewController(rep)
		svc := newMockGreeter(ctrl)

		svc.EXPECT().Greet("alice", 10).Return("hello", nil)

		// Called with "bob" instead of "alice"
		_, _ = svc.Greet("bob", 10)

		assert.Equal(t, 1, rep.FatalCount())
		assert.Contains(t, rep.fatals[0], "unexpected call")
	})

	t.Run("InOrder_OutOfOrder_ReportsFatal", func(t *testing.T) {
		t.Parallel()
		rep := &threadSafeReporter{}
		ctrl := mock.NewController(rep)
		svc := newMockGreeter(ctrl)

		c1 := svc.EXPECT().Greet("first", 1).Return("1", nil)
		c2 := svc.EXPECT().Greet("second", 2).Return("2", nil)
		c3 := svc.EXPECT().Greet("third", 3).Return("3", nil)

		mock.InOrder(c1, c2, c3)

		// Call c2 first -> prereq c1 not satisfied -> unexpected call!
		_, _ = svc.Greet("second", 2)

		assert.Equal(t, 1, rep.FatalCount())
		assert.Contains(t, rep.fatals[0], "unexpected call")
	})

	t.Run("InOrder_CorrectOrder_Succeeds", func(t *testing.T) {
		t.Parallel()
		rep := &threadSafeReporter{}
		ctrl := mock.NewController(rep)
		svc := newMockGreeter(ctrl)

		c1 := svc.EXPECT().Greet("first", 1).Return("1", nil)
		c2 := svc.EXPECT().Greet("second", 2).Return("2", nil)
		c3 := svc.EXPECT().Greet("third", 3).Return("3", nil)

		mock.InOrder(c1, c2, c3)

		r1, _ := svc.Greet("first", 1)
		r2, _ := svc.Greet("second", 2)
		r3, _ := svc.Greet("third", 3)

		assert.Equal(t, "1", r1)
		assert.Equal(t, "2", r2)
		assert.Equal(t, "3", r3)

		assert.True(t, ctrl.Satisfied())
		ctrl.Finish()
		assert.Equal(t, 0, rep.ErrorCount())
		assert.Equal(t, 0, rep.FatalCount())
	})

	t.Run("ConcurrentFinish_Idempotent", func(t *testing.T) {
		t.Parallel()
		rep := &threadSafeReporter{}
		ctrl := mock.NewController(rep)
		svc := newMockGreeter(ctrl)

		svc.EXPECT().Noop().Times(1)
		// Missing call

		const goroutines = 20
		var wg sync.WaitGroup
		wg.Add(goroutines)

		for range goroutines {
			go func() {
				defer wg.Done()
				ctrl.Finish()
			}()
		}

		wg.Wait()

		// Only exactly 1 error should be recorded, even under concurrent Finish()
		assert.Equal(t, 1, rep.ErrorCount())
	})
}
