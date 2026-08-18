// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fsm

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// TransitionCallback represents the function signature for transition hooks.
//
// It receives the execution context, source state, triggering event, and target state.
// If a before-hook returns a non-nil error, the transition is cancelled,
// subsequent hooks are skipped, and the state remains unchanged.
type TransitionCallback[State comparable, Event comparable] func(
	ctx context.Context,
	from State,
	event Event,
	to State,
) error

// TransitionRule defines a valid state transition triggered by a specific event.
type TransitionRule[State comparable, Event comparable] struct {
	// From represents the required starting state for the transition.
	From State
	// Event represents the triggering event that initiates the transition.
	Event Event
	// To represents the target state resulting from a successful transition.
	To State
}

// FSM is a strictly typed, thread-safe finite state machine parameterized
// over State and Event comparable types.
//
// It enforces transactional transition safety using a dual-mutex coordination model,
// separating transition serialization from fast, concurrent state reads.
// New instances of FSM must be created using the [NewFSM] constructor function.
type FSM[State, Event comparable] struct {
	transMu     sync.Mutex
	mu          sync.RWMutex
	current     State
	rules       map[State]map[Event]State
	beforeHooks map[Event][]TransitionCallback[State, Event]
	afterHooks  map[Event][]TransitionCallback[State, Event]
}

// NewFSM instantiates and returns a new finite state machine with the given initial state.
func NewFSM[State, Event comparable](initial State) *FSM[State, Event] {
	return &FSM[State, Event]{
		current:     initial,
		rules:       make(map[State]map[Event]State),
		beforeHooks: make(map[Event][]TransitionCallback[State, Event]),
		afterHooks:  make(map[Event][]TransitionCallback[State, Event]),
	}
}

// AddRules registers one or more valid transition rules in the state machine.
//
// If a duplicate rule for the same (From, Event) pair is registered, it is
// overwritten by the last entry. AddRules is safe for concurrent use.
func (f *FSM[State, Event]) AddRules(rules ...TransitionRule[State, Event]) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, r := range rules {
		if f.rules[r.From] == nil {
			f.rules[r.From] = make(map[Event]State)
		}

		f.rules[r.From][r.Event] = r.To
	}
}

// CurrentState returns the current state of the FSM.
//
// It is non-blocking and safe to call concurrently while transitions are in progress.
func (f *FSM[State, Event]) CurrentState() State {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.current
}

// ForceSet bypasses the transition rules and sets the current state directly.
//
// This is intended primarily for test setup where placing the FSM into a specific
// precondition state is required. Using ForceSet in production code is strongly discouraged.
func (f *FSM[State, Event]) ForceSet(state State) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.current = state
}

// OnBefore registers a callback that is invoked before a transition
// for the given event is applied.
//
// If any before-callback returns an error, the transition is cancelled,
// subsequent callbacks are skipped, and the state remains unchanged.
func (f *FSM[State, Event]) OnBefore(event Event, cb TransitionCallback[State, Event]) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.beforeHooks[event] = append(f.beforeHooks[event], cb)
}

// OnAfter registers a callback that is invoked after a transition
// for the given event has been applied.
//
// Errors returned by after-hooks are ignored since the state change has already
// been committed.
func (f *FSM[State, Event]) OnAfter(event Event, cb TransitionCallback[State, Event]) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.afterHooks[event] = append(f.afterHooks[event], cb)
}

// Validate checks whether a transition from the current state on the
// given event is defined in the registered rules.
//
// It returns the target state and true if a rule exists, or the zero value
// of State and false if no such transition is defined. Validate is safe
// for concurrent use and does not mutate the state machine.
//
// # Complexity
//
// Time Complexity: O(1)
func (f *FSM[State, Event]) Validate(event Event) (State, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	events, ok := f.rules[f.current]
	if !ok {
		var zero State
		return zero, false
	}

	to, ok := events[event]

	return to, ok
}

// Transition atomically moves the FSM from its current state to the next
// state determined by the given event.
//
// Transition is fully thread-safe and serialized. If a transition is already
// in progress, subsequent calls to Transition will block until the active
// transition completes, protecting before-hooks from executing on an outdated
// state context.
//
// # Complexity
//
// Time Complexity: O(N+1) for transition validation, where N is the
// number of registered before/after hooks for the given event.
func (f *FSM[State, Event]) Transition(ctx context.Context, event Event) error {
	f.transMu.Lock()
	defer f.transMu.Unlock()

	f.mu.RLock()
	from := f.current

	events, ok := f.rules[from]
	if !ok {
		f.mu.RUnlock()
		return fmt.Errorf("kata: no transitions defined from state %v", from)
	}

	to, exists := events[event]
	if !exists {
		f.mu.RUnlock()
		return fmt.Errorf("kata: invalid transition from state %v on event %v", from, event)
	}

	// Copy the hooks under the read-lock to safely execute them outside
	// the lock later, avoiding deadlocks if hooks call back into the FSM.
	beforeHooks := make([]TransitionCallback[State, Event], len(f.beforeHooks[event]))
	copy(beforeHooks, f.beforeHooks[event])

	afterHooks := make([]TransitionCallback[State, Event], len(f.afterHooks[event]))
	copy(afterHooks, f.afterHooks[event])

	f.mu.RUnlock()

	// Execute before-hooks safely outside the RWMutex.
	// Since transitions are serialized by transMu, we are guaranteed that
	// f.current has not changed since our validation step.
	for _, hook := range beforeHooks {
		if hook != nil {
			if err := hook(ctx, from, event, to); err != nil {
				return fmt.Errorf("kata: before hook aborted transition %v -> %v: %w", from, to, err)
			}
		}
	}

	f.mu.Lock()
	f.current = to
	f.mu.Unlock()

	// Execute after-hooks. Errors are swallowed as the state change is already committed.
	for _, hook := range afterHooks {
		if hook != nil {
			_ = hook(ctx, from, event, to)
		}
	}

	return nil
}

// ToDOT exports the state machine's transition graph as a Graphviz DOT representation.
//
// The resulting string can be rendered using the `dot` command-line utility
// (e.g., `dot -Tsvg`) or any online Graphviz viewer to visualize the state diagram.
// It is non-blocking and safe to call concurrently while transitions are in progress.
func (f *FSM[State, Event]) ToDOT() string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	var sb strings.Builder

	sb.WriteString("digraph FSM {\n")
	sb.WriteString("    rankdir=LR;\n")
	sb.WriteString("    node [shape=circle, style=filled, fillcolor=lightblue];\n\n")

	fmt.Fprintf(&sb, "    \"%v\" [fillcolor=lightgreen, style=filled, penwidth=2];\n\n", f.current)

	for from, events := range f.rules {
		for event, to := range events {
			fmt.Fprintf(&sb, "    \"%v\" -> \"%v\" [label=\"%v\"];\n", from, to, event)
		}
	}

	sb.WriteString("}\n")

	return sb.String()
}

// String returns a human-readable representation of the FSM for debugging.
func (f *FSM[State, Event]) String() string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return fmt.Sprintf("FSM{current=%v, rules=%d}", f.current, f.countRules())
}

func (f *FSM[State, Event]) countRules() int {
	n := 0
	for _, events := range f.rules {
		n += len(events)
	}

	return n
}
