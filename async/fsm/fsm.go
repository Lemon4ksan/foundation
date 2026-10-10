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
// The zero value is nil and not ready to use.
type TransitionCallback[State comparable, Event comparable] func(
	ctx context.Context,
	from State,
	event Event,
	to State,
) error

// TransitionRule defines a valid state transition triggered by a specific event.
//
// The zero value is empty and ready to use.
type TransitionRule[State comparable, Event comparable] struct {
	// From represents the required starting state for the transition.
	From State
	// Event represents the triggering event that initiates the transition.
	Event Event
	// To represents the target state resulting from a successful transition.
	To State
}

// Machine is a strictly typed, thread-safe finite state machine parameterized over State and Event comparable types.
//
// The zero value is empty and not ready to use.
// Read-only operations are safe for concurrent use; mutations require external synchronization unless stated otherwise.
// Note that methods on [Machine] manage their own internal locks.
type Machine[State, Event comparable] struct {
	transMu     sync.Mutex
	mu          sync.RWMutex
	current     State
	rules       map[State]map[Event]State
	beforeHooks map[Event][]TransitionCallback[State, Event]
	afterHooks  map[Event][]TransitionCallback[State, Event]
}

// New instantiates and returns a new [Machine] with the given initial state.
func New[State, Event comparable](initial State) *Machine[State, Event] {
	return &Machine[State, Event]{
		current:     initial,
		rules:       make(map[State]map[Event]State),
		beforeHooks: make(map[Event][]TransitionCallback[State, Event]),
		afterHooks:  make(map[Event][]TransitionCallback[State, Event]),
	}
}

// AddRules registers one or more valid [TransitionRule] objects in the state machine.
func (f *Machine[State, Event]) AddRules(rules ...TransitionRule[State, Event]) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.rules == nil {
		f.rules = make(map[State]map[Event]State)
	}

	for _, r := range rules {
		if f.rules[r.From] == nil {
			f.rules[r.From] = make(map[Event]State)
		}

		f.rules[r.From][r.Event] = r.To
	}
}

// CurrentState returns the current state of the machine.
func (f *Machine[State, Event]) CurrentState() State {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.current
}

// ForceSet bypasses the transition rules and sets the current state directly.
func (f *Machine[State, Event]) ForceSet(state State) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.current = state
}

// OnBefore registers a callback that is invoked before a transition for the given event is applied.
func (f *Machine[State, Event]) OnBefore(event Event, cb TransitionCallback[State, Event]) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.beforeHooks == nil {
		f.beforeHooks = make(map[Event][]TransitionCallback[State, Event])
	}

	f.beforeHooks[event] = append(f.beforeHooks[event], cb)
}

// OnAfter registers a callback that is invoked after a transition for the given event has been applied.
func (f *Machine[State, Event]) OnAfter(event Event, cb TransitionCallback[State, Event]) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.afterHooks == nil {
		f.afterHooks = make(map[Event][]TransitionCallback[State, Event])
	}

	f.afterHooks[event] = append(f.afterHooks[event], cb)
}

// Validate checks whether a transition from the current state on the given event is defined in the registered rules.
func (f *Machine[State, Event]) Validate(event Event) (State, bool) {
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

// Transition atomically moves the machine from its current state to the next state determined by the given event.
func (f *Machine[State, Event]) Transition(ctx context.Context, event Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	f.transMu.Lock()
	defer f.transMu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}

	f.mu.RLock()
	from := f.current

	events, ok := f.rules[from]
	if !ok {
		f.mu.RUnlock()
		return fmt.Errorf("fsm: no transitions defined from state %v", from)
	}

	to, exists := events[event]
	if !exists {
		f.mu.RUnlock()
		return fmt.Errorf("fsm: invalid transition from state %v on event %v", from, event)
	}

	// Copy the hooks under the read-lock to safely execute them outside
	// the lock later, avoiding deadlocks if hooks call back into the machine.
	beforeHooks := make([]TransitionCallback[State, Event], len(f.beforeHooks[event]))
	copy(beforeHooks, f.beforeHooks[event])

	afterHooks := make([]TransitionCallback[State, Event], len(f.afterHooks[event]))
	copy(afterHooks, f.afterHooks[event])

	f.mu.RUnlock()

	// Execute before-hooks safely outside the RWMutex.
	// Since transitions are serialized by transMu, we expect that
	// f.current has not changed unless forcefully mutated.
	for _, hook := range beforeHooks {
		if hook != nil {
			if err := hook(ctx, from, event, to); err != nil {
				return fmt.Errorf("fsm: before hook aborted transition %v -> %v: %w", from, to, err)
			}
		}
	}

	f.mu.Lock()
	if f.current != from {
		current := f.current
		f.mu.Unlock()
		return fmt.Errorf("fsm: state mutated from %v to %v during transition", from, current)
	}
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
func (f *Machine[State, Event]) ToDOT() string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	var sb strings.Builder

	sb.WriteString("digraph Machine {\n")
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

// String returns a human-readable representation of the machine for debugging.
func (f *Machine[State, Event]) String() string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return fmt.Sprintf("Machine{current=%v, rules=%d}", f.current, f.countRules())
}

func (f *Machine[State, Event]) countRules() int {
	n := 0
	for _, events := range f.rules {
		n += len(events)
	}

	return n
}
