// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package fsm implements a strictly typed, thread-safe finite state machine (FSM).
//
// It provides a generic FSM framework parameterized over comparable [State] and
// [Event] types, ensuring that invalid transitions are caught at compile time.
// All state transitions are atomic, thread-safe, and support transactional
// cancellation via pre-transition hooks.
//
// # Concurrency and Transaction Safety
//
// To prevent common race conditions and state inconsistencies under high concurrent
// load, the FSM separates transition execution from state reads using a dual-lock
// architecture:
//
//  1. Transition Serialization: An internal mutex (transMu) serializes all active
//     state transitions. This guarantees that only one transition can execute its
//     before-hooks at any given time, completely eliminating the "optimistic side-effect leak"
//     where concurrent tasks execute mutating before-hooks only to have the transition
//     aborted later at the write stage.
//
//  2. Non-blocking State Reads: Read-only operations (such as [FSM.CurrentState],
//     [FSM.Validate], and [FSM.ToDOT]) use a shared read-write mutex ([sync.RWMutex]).
//     This ensures that readers are never blocked by ongoing transition hook executions.
//
// # Transition Lifecycle
//
// A state transition executed via [FSM.Transition] goes through the following phases:
//  1. Serialization: Acquire the global transition mutex (transMu).
//  2. Validation: Under a shared read lock (mu RLock), verify that a transition
//     rule exists for the current state and trigger event. If invalid, the read lock
//     is released, and an error is returned.
//  3. Hook Copying: Still under the read lock, safe local copies of the before-hooks
//     and after-hooks registered for the event are created. The read lock is then released.
//  4. Before-Hook Execution: Run all copied before-hooks sequentially. If any hook
//     returns an error, the transition is aborted, and the error is propagated.
//  5. State Commit: Acquire the write lock (mu Lock), update the current state
//     to the target state, and release the write lock.
//  6. After-Hook Execution: Run all copied after-hooks sequentially. Errors returned
//     by after-hooks are ignored since the state change has already been committed.
//  7. Release: Release the global transition mutex (transMu).
//
// # Example
//
//	package main
//
//	import (
//		"context"
//		"fmt"
//
//		"github.com/lemon4ksan/foundation/async/fsm"
//	)
//
//	type State int
//	const (
//		Idle State = iota
//		Running
//		Stopped
//	)
//
//	type Event int
//	const (
//		Start Event = iota
//		Stop
//	)
//
//	func (s State) String() string {
//		return [...]string{"Idle", "Running", "Stopped"}[s]
//	}
//
//	func (e Event) String() string {
//		return [...]string{"Start", "Stop"}[e]
//	}
//
//	func main() {
//		fsm := kata.NewFSM[State, Event](Idle)
//
//		fsm.AddRules(
//			kata.TransitionRule[State, Event]{From: Idle, Event: Start, To: Running},
//			kata.TransitionRule[State, Event]{From: Running, Event: Stop, To: Stopped},
//		)
//
//		fsm.OnBefore(Start, func(ctx context.Context, from State, event Event, to State) error {
//			fmt.Printf("Transitioning: %v -> %v\n", from, to)
//			return nil
//		})
//
//		if err := fsm.Transition(context.Background(), Start); err != nil {
//			panic(err)
//		}
//
//		fmt.Println(fsm.ToDOT())
//	}
package fsm
