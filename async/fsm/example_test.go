// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fsm_test

import (
	"context"
	"fmt"

	"github.com/lemon4ksan/foundation/async/fsm"
)

type (
	State string
	Event string
)

const (
	StateIdle       State = "idle"
	StateProcessing State = "processing"
	StateCompleted  State = "completed"
	StateFailed     State = "failed"

	EventStart Event = "start"
	EventDone  Event = "done"
	EventFail  Event = "fail"
	EventRetry Event = "retry"
)

func ExampleNewFSM() {
	machine := fsm.New[State, Event](StateIdle)

	machine.AddRules(
		fsm.TransitionRule[State, Event]{From: StateIdle, Event: EventStart, To: StateProcessing},
		fsm.TransitionRule[State, Event]{From: StateProcessing, Event: EventDone, To: StateCompleted},
		fsm.TransitionRule[State, Event]{From: StateProcessing, Event: EventFail, To: StateFailed},
		fsm.TransitionRule[State, Event]{From: StateFailed, Event: EventRetry, To: StateProcessing},
	)

	// Validate valid transition
	if next, ok := machine.Validate(EventStart); ok {
		fmt.Printf("Can transition from %s to %s on %s\n", machine.CurrentState(), next, EventStart)
	}

	// Execute transition
	ctx := context.Background()
	if err := machine.Transition(ctx, EventStart); err != nil {
		fmt.Printf("Transition error: %v\n", err)
		return
	}

	fmt.Printf("Current state: %s\n", machine.CurrentState())

	// Output:
	// Can transition from idle to processing on start
	// Current state: processing
}

func ExampleFSM_OnBefore() {
	machine := fsm.New[State, Event](StateIdle)

	machine.AddRules(
		fsm.TransitionRule[State, Event]{From: StateIdle, Event: EventStart, To: StateProcessing},
	)

	machine.OnBefore(EventStart, func(ctx context.Context, from State, event Event, to State) error {
		fmt.Printf("Hook: Preparing transition %s -> %s on %s\n", from, to, event)
		return nil
	})

	machine.OnAfter(EventStart, func(ctx context.Context, from State, event Event, to State) error {
		fmt.Printf("Hook: Committed transition %s -> %s\n", from, to)
		return nil
	})

	_ = machine.Transition(context.Background(), EventStart)

	// Output:
	// Hook: Preparing transition idle -> processing on start
	// Hook: Committed transition idle -> processing
}
