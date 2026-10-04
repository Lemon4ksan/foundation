// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fsm

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type testState int

const (
	stateA testState = iota
	stateB
	stateC
)

type testEvent int

const (
	eventGo testEvent = iota
	eventBack
	eventReset
)

// TestFSM_BasicTransition verifies the fundamental state transition flow,
// ensuring that valid transition rules update the state machine correctly.
func TestFSM_BasicTransition(t *testing.T) {
	fsm := New[testState, testEvent](stateA)

	fsm.AddRules(
		TransitionRule[testState, testEvent]{From: stateA, Event: eventGo, To: stateB},
		TransitionRule[testState, testEvent]{From: stateB, Event: eventGo, To: stateC},
		TransitionRule[testState, testEvent]{From: stateB, Event: eventBack, To: stateA},
		TransitionRule[testState, testEvent]{From: stateC, Event: eventReset, To: stateA},
	)

	if got := fsm.CurrentState(); got != stateA {
		t.Fatalf("expected initial state %v, got %v", stateA, got)
	}

	if err := fsm.Transition(context.Background(), eventGo); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := fsm.CurrentState(); got != stateB {
		t.Fatalf("expected state %v after transition, got %v", stateB, got)
	}

	if err := fsm.Transition(context.Background(), eventGo); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := fsm.CurrentState(); got != stateC {
		t.Fatalf("expected state %v after second transition, got %v", stateC, got)
	}

	if err := fsm.Transition(context.Background(), eventReset); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := fsm.CurrentState(); got != stateA {
		t.Fatalf("expected state %v after reset, got %v", stateA, got)
	}
}

// TestFSM_InvalidTransition verifies that attempting to execute an undefined
// transition from the current state returns an error and leaves the state unchanged.
func TestFSM_InvalidTransition(t *testing.T) {
	fsm := New[testState, testEvent](stateA)

	fsm.AddRules(
		TransitionRule[testState, testEvent]{From: stateA, Event: eventGo, To: stateB},
	)

	err := fsm.Transition(context.Background(), eventBack)
	if err == nil {
		t.Fatal("expected error for invalid transition, got nil")
	}

	if !strings.Contains(err.Error(), "invalid transition") {
		t.Fatalf("expected 'invalid transition' in error, got: %v", err)
	}
}

// TestFSM_BeforeHookRollback verifies that if a registered before-hook returns
// an error, the transition is aborted, and the state machine safely rolls back
// to its original state.
func TestFSM_BeforeHookRollback(t *testing.T) {
	fsm := New[testState, testEvent](stateA)

	fsm.AddRules(
		TransitionRule[testState, testEvent]{From: stateA, Event: eventGo, To: stateB},
	)

	abortErr := errors.New("aborting transition")

	fsm.OnBefore(eventGo, func(_ context.Context, _ testState, _ testEvent, _ testState) error {
		return abortErr
	})

	err := fsm.Transition(context.Background(), eventGo)
	if err == nil {
		t.Fatal("expected error from before-hook, got nil")
	}

	if !errors.Is(err, abortErr) {
		t.Fatalf("expected wrapped abort error, got: %v", err)
	}

	if got := fsm.CurrentState(); got != stateA {
		t.Fatalf("state should remain %v after rollback, got %v", stateA, got)
	}
}

// TestFSM_BeforeAndAfterHooks verifies that both before-hooks and after-hooks
// are triggered with the correct state parameters during a successful transition.
func TestFSM_BeforeAndAfterHooks(t *testing.T) {
	fsm := New[testState, testEvent](stateA)

	fsm.AddRules(
		TransitionRule[testState, testEvent]{From: stateA, Event: eventGo, To: stateB},
	)

	var beforeCalled, afterCalled atomic.Bool

	fsm.OnBefore(eventGo, func(_ context.Context, from testState, event testEvent, to testState) error {
		if from != stateA || to != stateB || event != eventGo {
			t.Errorf("before hook received wrong args: from=%v, event=%v, to=%v", from, event, to)
		}

		beforeCalled.Store(true)

		return nil
	})

	fsm.OnAfter(eventGo, func(_ context.Context, from testState, event testEvent, to testState) error {
		if from != stateA || to != stateB || event != eventGo {
			t.Errorf("after hook received wrong args: from=%v, event=%v, to=%v", from, event, to)
		}

		afterCalled.Store(true)

		return nil
	})

	if err := fsm.Transition(context.Background(), eventGo); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !beforeCalled.Load() {
		t.Fatal("before hook was not called")
	}

	if !afterCalled.Load() {
		t.Fatal("after hook was not called")
	}

	if got := fsm.CurrentState(); got != stateB {
		t.Fatalf("expected state %v, got %v", stateB, got)
	}
}

// TestFSM_AfterHookErrorIgnored verifies that errors returned by after-hooks
// do not block or roll back the transition, as the state change has already
// been committed.
func TestFSM_AfterHookErrorIgnored(t *testing.T) {
	fsm := New[testState, testEvent](stateA)

	fsm.AddRules(
		TransitionRule[testState, testEvent]{From: stateA, Event: eventGo, To: stateB},
	)

	fsm.OnAfter(eventGo, func(_ context.Context, _ testState, _ testEvent, _ testState) error {
		return errors.New("after hook error")
	})

	err := fsm.Transition(context.Background(), eventGo)
	if err != nil {
		t.Fatalf("transition should succeed even if after-hook fails, got: %v", err)
	}

	if got := fsm.CurrentState(); got != stateB {
		t.Fatalf("expected state %v, got %v", stateB, got)
	}
}

// TestFSM_MultipleBeforeHooksOrder verifies that multiple registered before-hooks
// for the same event are executed sequentially in their exact registration order.
func TestFSM_MultipleBeforeHooksOrder(t *testing.T) {
	fsm := New[testState, testEvent](stateA)

	fsm.AddRules(
		TransitionRule[testState, testEvent]{From: stateA, Event: eventGo, To: stateB},
	)

	var order []int

	fsm.OnBefore(eventGo, func(_ context.Context, _ testState, _ testEvent, _ testState) error {
		order = append(order, 1)
		return nil
	})

	fsm.OnBefore(eventGo, func(_ context.Context, _ testState, _ testEvent, _ testState) error {
		order = append(order, 2)
		return nil
	})

	if err := fsm.Transition(context.Background(), eventGo); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(order) != 2 || order[0] != 1 || order[1] != 2 {
		t.Fatalf("expected hooks called in order [1, 2], got %v", order)
	}
}

// TestFSM_ConcurrentTransitions verifies the thread-safety of the state machine
// under high concurrent load, ensuring that only a single transition can succeed
// from a given state, while all others are rejected.
func TestFSM_ConcurrentTransitions(t *testing.T) {
	fsm := New[testState, testEvent](stateA)

	fsm.AddRules(
		TransitionRule[testState, testEvent]{From: stateA, Event: eventGo, To: stateB},
		TransitionRule[testState, testEvent]{From: stateB, Event: eventBack, To: stateA},
	)

	const goroutines = 100

	var (
		wg        sync.WaitGroup
		succCount atomic.Int64
		errCount  atomic.Int64
	)

	for range goroutines {
		wg.Go(func() {
			err := fsm.Transition(context.Background(), eventGo)
			if err != nil {
				errCount.Add(1)
			} else {
				succCount.Add(1)
			}
		})
	}

	wg.Wait()

	if succCount.Load() != 1 {
		t.Fatalf("expected exactly 1 successful transition, got %d", succCount.Load())
	}

	if errCount.Load() != goroutines-1 {
		t.Fatalf("expected %d errors, got %d", goroutines-1, errCount.Load())
	}

	state := fsm.CurrentState()
	if state != stateB {
		t.Fatalf("expected final state %v, got %v", stateB, state)
	}
}

// TestFSM_ConcurrentTransitionsRace executes parallel alternating transitions
// concurrently to ensure there are no data races or invalid states during
// high-contention read/write races.
func TestFSM_ConcurrentTransitionsRace(t *testing.T) {
	fsm := New[testState, testEvent](stateA)

	fsm.AddRules(
		TransitionRule[testState, testEvent]{From: stateA, Event: eventGo, To: stateB},
		TransitionRule[testState, testEvent]{From: stateB, Event: eventBack, To: stateA},
	)

	const rounds = 50

	var wg sync.WaitGroup

	for range rounds {
		wg.Go(func() {
			_ = fsm.Transition(context.Background(), eventGo)
		})

		wg.Go(func() {
			_ = fsm.Transition(context.Background(), eventBack)
		})
	}

	wg.Wait()

	state := fsm.CurrentState()
	if state != stateA && state != stateB {
		t.Fatalf("unexpected final state after race: %v", state)
	}
}

// TestFSM_VerifyStateAfterAlternatingTransitions verifies that sequential,
// alternating transitions execute cleanly without state corruption.
func TestFSM_VerifyStateAfterAlternatingTransitions(t *testing.T) {
	fsm := New[testState, testEvent](stateA)

	fsm.AddRules(
		TransitionRule[testState, testEvent]{From: stateA, Event: eventGo, To: stateB},
		TransitionRule[testState, testEvent]{From: stateB, Event: eventBack, To: stateA},
	)

	for range 100 {
		if err := fsm.Transition(context.Background(), eventGo); err != nil {
			t.Fatalf("transition A->B failed: %v", err)
		}

		if got := fsm.CurrentState(); got != stateB {
			t.Fatalf("expected state %v, got %v", stateB, got)
		}

		if err := fsm.Transition(context.Background(), eventBack); err != nil {
			t.Fatalf("transition B->A failed: %v", err)
		}

		if got := fsm.CurrentState(); got != stateA {
			t.Fatalf("expected state %v, got %v", stateA, got)
		}
	}
}

// TestFSM_Validate verifies the dry-run validation API, ensuring it accurately
// reports transition destinations without mutating the state.
func TestFSM_Validate(t *testing.T) {
	fsm := New[testState, testEvent](stateA)

	fsm.AddRules(
		TransitionRule[testState, testEvent]{From: stateA, Event: eventGo, To: stateB},
	)

	to, ok := fsm.Validate(eventGo)
	if !ok || to != stateB {
		t.Fatalf("expected valid transition to %v, got %v (ok=%v)", stateB, to, ok)
	}

	_, ok = fsm.Validate(eventBack)
	if ok {
		t.Fatal("expected invalid transition for eventBack from stateA")
	}
}

// TestFSM_Validate_NoRulesForState verifies that Validate returns false if
// no transition rules have been registered for the current state.
func TestFSM_Validate_NoRulesForState(t *testing.T) {
	fsm := New[testState, testEvent](stateC)

	fsm.AddRules(
		TransitionRule[testState, testEvent]{From: stateA, Event: eventGo, To: stateB},
	)

	_, ok := fsm.Validate(eventGo)
	if ok {
		t.Fatal("expected invalid transition when no rules exist for current state")
	}
}

// TestFSM_ToDOT verifies that the DOT-graph generator produces a valid Graphviz
// representation containing all states, transitions, and labels.
func TestFSM_ToDOT(t *testing.T) {
	fsm := New[testState, testEvent](stateA)

	fsm.AddRules(
		TransitionRule[testState, testEvent]{From: stateA, Event: eventGo, To: stateB},
		TransitionRule[testState, testEvent]{From: stateB, Event: eventGo, To: stateC},
	)

	dot := fsm.ToDOT()

	if !strings.Contains(dot, "digraph FSM") {
		t.Fatal("DOT output missing digraph header")
	}

	if !strings.Contains(dot, `"0"`) {
		t.Fatal("DOT output missing stateA (0) node")
	}

	if !strings.Contains(dot, `"1"`) {
		t.Fatal("DOT output missing stateB (1) node")
	}

	if !strings.Contains(dot, `"2"`) {
		t.Fatal("DOT output missing stateC (2) node")
	}

	if !strings.Contains(dot, "[label=\"0\"]") {
		t.Fatal("DOT output missing eventGo transition label")
	}
}

// TestFSM_CurrentState_Concurrent verifies that reading the current state
// is safe for concurrent execution by multiple goroutines.
func TestFSM_CurrentState_Concurrent(t *testing.T) {
	fsm := New[testState, testEvent](stateA)

	fsm.AddRules(
		TransitionRule[testState, testEvent]{From: stateA, Event: eventGo, To: stateB},
	)

	var wg sync.WaitGroup

	for range 50 {
		wg.Go(func() {
			_ = fsm.CurrentState()
		})
	}

	wg.Wait()
}

// TestFSM_String verifies that the Stringer interface implementation
// produces a clear, informative debugging representation.
func TestFSM_String(t *testing.T) {
	fsm := New[testState, testEvent](stateA)

	fsm.AddRules(
		TransitionRule[testState, testEvent]{From: stateA, Event: eventGo, To: stateB},
	)

	s := fsm.String()
	if !strings.Contains(s, "current=0") {
		t.Fatalf("String() should contain current state, got: %s", s)
	}
}

// TestFSM_ContextCancellation verifies that before-hooks respect context cancellation,
// immediately aborting the transition if the context is cancelled.
func TestFSM_ContextCancellation(t *testing.T) {
	fsm := New[testState, testEvent](stateA)

	fsm.AddRules(
		TransitionRule[testState, testEvent]{From: stateA, Event: eventGo, To: stateB},
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	fsm.OnBefore(eventGo, func(ctx context.Context, _ testState, _ testEvent, _ testState) error {
		return ctx.Err()
	})

	err := fsm.Transition(ctx, eventGo)
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}

	if got := fsm.CurrentState(); got != stateA {
		t.Fatalf("state should remain %v after cancelled before-hook, got %v", stateA, got)
	}
}

// TestFSM_NoTransitionsDefined verifies that attempting a transition on an FSM
// with no rules configured returns a descriptive error.
func TestFSM_NoTransitionsDefined(t *testing.T) {
	fsm := New[testState, testEvent](stateA)

	err := fsm.Transition(context.Background(), eventGo)
	if err == nil {
		t.Fatal("expected error when no rules are defined, got nil")
	}

	if !strings.Contains(err.Error(), "no transitions defined") {
		t.Fatalf("expected 'no transitions defined' in error, got: %v", err)
	}
}

// TestFSM_ForceSet verifies that the ForceSet debug method correctly bypasses all rules
// to place the FSM in a specific target state.
func TestFSM_ForceSet(t *testing.T) {
	fsm := New[testState, testEvent](stateA)

	fsm.AddRules(
		TransitionRule[testState, testEvent]{From: stateA, Event: eventGo, To: stateB},
		TransitionRule[testState, testEvent]{From: stateC, Event: eventReset, To: stateA},
	)

	fsm.ForceSet(stateC)

	if got := fsm.CurrentState(); got != stateC {
		t.Fatalf("expected state %v after ForceSet, got %v", stateC, got)
	}

	if err := fsm.Transition(context.Background(), eventReset); err != nil {
		t.Fatalf("transition from forced state should work: %v", err)
	}

	if got := fsm.CurrentState(); got != stateA {
		t.Fatalf("expected state %v after transition, got %v", stateA, got)
	}
}

// TestFSM_Adversarial_CyclicHighConcurrency stress tests 200 concurrent workers
// attempting transitions in a cyclic 5-state state machine.
func TestFSM_Adversarial_CyclicHighConcurrency(t *testing.T) {
	const numStates = 5
	f := New[int, int](0)

	// Build cyclic rules: 0 -> 1 -> 2 -> 3 -> 4 -> 0
	for i := range numStates {
		next := (i + 1) % numStates
		f.AddRules(TransitionRule[int, int]{
			From:  i,
			Event: i, // event i triggers i -> (i+1)%numStates
			To:    next,
		})
	}

	const (
		numGoroutines = 100
		iterations    = 1000
	)

	var (
		wg        sync.WaitGroup
		successes atomic.Int64
		failures  atomic.Int64
	)

	ctx := context.Background()

	for g := range numGoroutines {
		wg.Add(1)
		go func(seed uint64) {
			defer wg.Done()
			rng := rand.New(rand.NewPCG(seed, seed^0x55555555))

			for range iterations {
				// Random event from [0, numStates)
				ev := rng.IntN(numStates)
				err := f.Transition(ctx, ev)
				if err == nil {
					successes.Add(1)
				} else {
					failures.Add(1)
				}

				// Interleave non-blocking read
				cur := f.CurrentState()
				if cur < 0 || cur >= numStates {
					t.Errorf("corrupted state observed: %d", cur)
				}
			}
		}(uint64(g + 1))
	}

	wg.Wait()

	finalState := f.CurrentState()
	if finalState < 0 || finalState >= numStates {
		t.Fatalf("final state %d out of bounds [0, %d)", finalState, numStates)
	}

	if successes.Load() == 0 {
		t.Fatal("expected at least some successful transitions under concurrency")
	}

	if successes.Load()+failures.Load() != numGoroutines*iterations {
		t.Fatalf("expected total attempts %d, got %d", numGoroutines*iterations, successes.Load()+failures.Load())
	}
}

// TestFSM_Adversarial_HookReentrantReads ensures hooks calling CurrentState,
// Validate, String, and ToDOT do not cause deadlocks.
func TestFSM_Adversarial_HookReentrantReads(t *testing.T) {
	f := New[string, string]("init")

	f.AddRules(
		TransitionRule[string, string]{From: "init", Event: "run", To: "running"},
		TransitionRule[string, string]{From: "running", Event: "stop", To: "stopped"},
	)

	var (
		beforeCalled atomic.Bool
		afterCalled  atomic.Bool
	)

	f.OnBefore("run", func(_ context.Context, from, _, to string) error {
		beforeCalled.Store(true)
		// Hook calls methods on fsm to verify no deadlock
		if cur := f.CurrentState(); cur != from {
			return fmt.Errorf("unexpected state in before-hook: %s != %s", cur, from)
		}
		if target, ok := f.Validate("run"); !ok || target != to {
			return fmt.Errorf("validate inside before-hook failed: target=%s, ok=%v", target, ok)
		}
		_ = f.String()
		_ = f.ToDOT()
		return nil
	})

	f.OnAfter("run", func(_ context.Context, from, _, to string) error {
		afterCalled.Store(true)
		// Hook calls methods on fsm after state commitment
		if cur := f.CurrentState(); cur != to {
			return fmt.Errorf("unexpected state in after-hook: %s != %s", cur, to)
		}
		_ = f.String()
		_ = f.ToDOT()
		return nil
	})

	err := f.Transition(context.Background(), "run")
	if err != nil {
		t.Fatalf("transition failed: %v", err)
	}

	if !beforeCalled.Load() || !afterCalled.Load() {
		t.Fatalf("expected both hooks to be called: before=%v, after=%v", beforeCalled.Load(), afterCalled.Load())
	}

	if cur := f.CurrentState(); cur != "running" {
		t.Fatalf("expected current state 'running', got '%s'", cur)
	}
}

// TestFSM_Adversarial_HookOrderAndAbortVerification tests that if an intermediate
// before-hook fails, subsequent before-hooks and after-hooks are never invoked,
// and state remains strictly unchanged.
func TestFSM_Adversarial_HookOrderAndAbortVerification(t *testing.T) {
	f := New[int, int](1)

	f.AddRules(TransitionRule[int, int]{From: 1, Event: 10, To: 2})

	var executionLog []int
	var mu sync.Mutex

	record := func(id int) {
		mu.Lock()
		defer mu.Unlock()
		executionLog = append(executionLog, id)
	}

	errAbort := errors.New("deliberate abort at hook 3")

	f.OnBefore(10, func(_ context.Context, _, _, _ int) error {
		record(1)
		return nil
	})
	f.OnBefore(10, func(_ context.Context, _, _, _ int) error {
		record(2)
		return nil
	})
	f.OnBefore(10, func(_ context.Context, _, _, _ int) error {
		record(3)
		return errAbort
	})
	f.OnBefore(10, func(_ context.Context, _, _, _ int) error {
		record(4)
		return nil
	})

	f.OnAfter(10, func(_ context.Context, _, _, _ int) error {
		record(5)
		return nil
	})

	err := f.Transition(context.Background(), 10)
	if err == nil {
		t.Fatal("expected error from aborted transition, got nil")
	}

	if !errors.Is(err, errAbort) {
		t.Fatalf("expected wrapped errAbort, got: %v", err)
	}

	if cur := f.CurrentState(); cur != 1 {
		t.Fatalf("state rolled back incorrectly: got %d, expected 1", cur)
	}

	mu.Lock()
	defer mu.Unlock()

	expectedLog := []int{1, 2, 3}
	if len(executionLog) != len(expectedLog) {
		t.Fatalf("expected execution log %v, got %v", expectedLog, executionLog)
	}
	for i, v := range expectedLog {
		if executionLog[i] != v {
			t.Fatalf("mismatch at index %d: expected %d, got %d", i, v, executionLog[i])
		}
	}
}

// TestFSM_Adversarial_ConcurrentHookRegistrationAndTransitions runs concurrent
// hook registrations, rule additions, and transitions to ensure no data races.
func TestFSM_Adversarial_ConcurrentHookRegistrationAndTransitions(t *testing.T) {
	f := New[int, int](0)

	f.AddRules(
		TransitionRule[int, int]{From: 0, Event: 1, To: 1},
		TransitionRule[int, int]{From: 1, Event: 2, To: 0},
	)

	const duration = 200 * time.Millisecond
	stop := make(chan struct{})
	timer := time.AfterFunc(duration, func() { close(stop) })
	defer timer.Stop()

	var wg sync.WaitGroup

	// Goroutine 1: transitions 0 -> 1 -> 0
	wg.Go(func() {
		for {
			select {
			case <-stop:
				return
			default:
				_ = f.Transition(context.Background(), 1)
				_ = f.Transition(context.Background(), 2)
			}
		}
	})

	// Goroutine 2: dynamic hook registrations
	wg.Go(func() {
		for {
			select {
			case <-stop:
				return
			default:
				f.OnBefore(1, func(_ context.Context, _, _, _ int) error { return nil })
				f.OnAfter(1, func(_ context.Context, _, _, _ int) error { return nil })
				f.OnBefore(2, nil) // test nil hook safety
			}
		}
	})

	// Goroutine 3: dynamic rule registrations
	wg.Go(func() {
		for {
			select {
			case <-stop:
				return
			default:
				f.AddRules(TransitionRule[int, int]{From: 0, Event: 1, To: 1})
				f.AddRules(TransitionRule[int, int]{From: 1, Event: 2, To: 0})
			}
		}
	})

	// Goroutine 4: concurrent Validate & CurrentState
	wg.Go(func() {
		for {
			select {
			case <-stop:
				return
			default:
				_ = f.CurrentState()
				_, _ = f.Validate(1)
				_, _ = f.Validate(2)
			}
		}
	})

	wg.Wait()
}

// TestFSM_Adversarial_LargeScaleGraph verifies graph manipulation on 200 states.
func TestFSM_Adversarial_LargeScaleGraph(t *testing.T) {
	const numStates = 200
	f := NewFSM[int, int](0)

	rules := make([]TransitionRule[int, int], 0, numStates*2)
	for i := range numStates {
		rules = append(rules,
			TransitionRule[int, int]{From: i, Event: 1, To: (i + 1) % numStates},
			TransitionRule[int, int]{From: i, Event: 2, To: (i + 2) % numStates},
		)
	}

	f.AddRules(rules...)

	// Step through entire chain
	for i := range numStates {
		next, ok := f.Validate(1)
		if !ok || next != (i+1)%numStates {
			t.Fatalf("expected step to %d, got %d (ok=%v)", (i+1)%numStates, next, ok)
		}
		if err := f.Transition(context.Background(), 1); err != nil {
			t.Fatalf("transition at state %d failed: %v", i, err)
		}
	}

	if cur := f.CurrentState(); cur != 0 {
		t.Fatalf("expected return to state 0, got %d", cur)
	}

	dot := f.ToDOT()
	if len(dot) == 0 {
		t.Fatal("empty DOT generated")
	}

	str := f.String()
	if len(str) == 0 {
		t.Fatal("empty String() generated")
	}
}

// TestNewFSM_Alias verifies that NewFSM functions identically to New as a backwards-compatible constructor alias.
func TestNewFSM_Alias(t *testing.T) {
	machine := NewFSM[string, string]("init")
	if machine == nil {
		t.Fatal("expected non-nil FSM instance from NewFSM")
	}
	if got := machine.CurrentState(); got != "init" {
		t.Fatalf("expected initial state 'init', got %q", got)
	}
}
