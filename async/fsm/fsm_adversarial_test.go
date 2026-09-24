// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fsm_test

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/async/fsm"
)

// TestFSM_Adversarial_CyclicHighConcurrency stress tests 200 concurrent workers
// attempting transitions in a cyclic 5-state state machine.
func TestFSM_Adversarial_CyclicHighConcurrency(t *testing.T) {
	const numStates = 5
	f := fsm.New[int, int](0)

	// Build cyclic rules: 0 -> 1 -> 2 -> 3 -> 4 -> 0
	for i := 0; i < numStates; i++ {
		next := (i + 1) % numStates
		f.AddRules(fsm.TransitionRule[int, int]{
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

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(seed uint64) {
			defer wg.Done()
			rng := rand.New(rand.NewPCG(seed, seed^0x55555555))

			for i := 0; i < iterations; i++ {
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
	f := fsm.New[string, string]("init")

	f.AddRules(
		fsm.TransitionRule[string, string]{From: "init", Event: "run", To: "running"},
		fsm.TransitionRule[string, string]{From: "running", Event: "stop", To: "stopped"},
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
	f := fsm.New[int, int](1)

	f.AddRules(fsm.TransitionRule[int, int]{From: 1, Event: 10, To: 2})

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
	f := fsm.New[int, int](0)

	f.AddRules(
		fsm.TransitionRule[int, int]{From: 0, Event: 1, To: 1},
		fsm.TransitionRule[int, int]{From: 1, Event: 2, To: 0},
	)

	const duration = 200 * time.Millisecond
	stop := make(chan struct{})
	timer := time.AfterFunc(duration, func() { close(stop) })
	defer timer.Stop()

	var wg sync.WaitGroup

	// Goroutine 1: transitions 0 -> 1 -> 0
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				_ = f.Transition(context.Background(), 1)
				_ = f.Transition(context.Background(), 2)
			}
		}
	}()

	// Goroutine 2: dynamic hook registrations
	wg.Add(1)
	go func() {
		defer wg.Done()
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
	}()

	// Goroutine 3: dynamic rule registrations
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				f.AddRules(fsm.TransitionRule[int, int]{From: 0, Event: 1, To: 1})
				f.AddRules(fsm.TransitionRule[int, int]{From: 1, Event: 2, To: 0})
			}
		}
	}()

	// Goroutine 4: concurrent Validate & CurrentState
	wg.Add(1)
	go func() {
		defer wg.Done()
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
	}()

	wg.Wait()
}

// TestFSM_Adversarial_LargeScaleGraph verifies graph manipulation on 200 states.
func TestFSM_Adversarial_LargeScaleGraph(t *testing.T) {
	const numStates = 200
	f := fsm.NewFSM[int, int](0)

	rules := make([]fsm.TransitionRule[int, int], 0, numStates*2)
	for i := 0; i < numStates; i++ {
		rules = append(rules,
			fsm.TransitionRule[int, int]{From: i, Event: 1, To: (i + 1) % numStates},
			fsm.TransitionRule[int, int]{From: i, Event: 2, To: (i + 2) % numStates},
		)
	}

	f.AddRules(rules...)

	// Step through entire chain
	for i := 0; i < numStates; i++ {
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
	machine := fsm.NewFSM[string, string]("init")
	if machine == nil {
		t.Fatal("expected non-nil FSM instance from NewFSM")
	}
	if got := machine.CurrentState(); got != "init" {
		t.Fatalf("expected initial state 'init', got %q", got)
	}
}
