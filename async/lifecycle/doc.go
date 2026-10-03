// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package lifecycle manages dependency-aware application startup, graceful
// shutdown, and concurrent background behavior loops.
//
// It provides two primary coordination mechanisms:
//
// 1. [Orchestrator] organizes registered services into a directed acyclic graph
// based on declared dependencies, executing their initialization, startup, and
// termination phases in topological order. On failure during StartAll, already
// started services are automatically rolled back in reverse order. Circular
// dependencies are detected and reported before initialization.
//
// 2. [BehaviorRunner] manages concurrent, long-running worker loops implementing
// the [Behavior] interface, supporting fail-fast cancellation and logging. Any
// [Behavior] can be adapted to a [Service] using [AsService], allowing the
// [Orchestrator] to manage concurrent execution loops within a dependency graph.
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: topological service startup and background loop management
//
// Rejected compromise: typical Go apps use fragile manual startup sequencing with no automatic rollback.
//
// Accepted cost: services must explicitly declare string-based dependencies and implement a strict interface.
//
// Allocations: unconstrained
package lifecycle
