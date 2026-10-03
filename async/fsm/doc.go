// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package fsm implements a strictly typed, thread-safe finite state machine (FSM).
//
// It provides a generic FSM framework parameterized over comparable [State] and
// [Event] types, ensuring that invalid transitions are caught at compile time.
// All state transitions are atomic, thread-safe, and support transactional
// cancellation via pre-transition hooks. The FSM separates transition execution
// from state reads using a dual-lock architecture. An internal mutex serializes
// transitions, preventing optimistic side-effect leaks. Non-blocking reads use
// a read-write lock so readers are not blocked by transition hooks.
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: type-safe state machines with transactional rollback
//
// Rejected compromise: typical FSMs use untyped strings and no rollback on hook failure.
//
// Accepted cost: extra locks and indirection during transitions.
//
// Allocations: bounded(1/op)
package fsm
