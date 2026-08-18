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
// started services are automatically rolled back in reverse order.
//
// 2. [BehaviorRunner] manages concurrent, long-running worker loops implementing
// the [Behavior] interface, supporting Fail-Fast cancellation and logging.
//
// # Service Orchestration (DAG)
//
// The [Orchestrator] performs a DFS-based topological sort of all registered
// [Service] instances that implement [Dependent]. The sort order determines the
// Init and Start sequences; Stop runs in the exact reverse. Circular dependencies
// are detected and reported as errors before any service is initialized.
//
// # Behavior Runner (Concurrent Loops)
//
// The [BehaviorRunner] runs continuous, blocking execution loops ([Behavior]) in
// separate goroutines. It blocks on shutdown until all behaviors have fully exited.
//
// Any [Behavior] can be adapted to a [Service] using [AsService], allowing the
// [Orchestrator] to manage concurrent execution loops within a dependency graph.
//
// # Error Handling
//
// If [Orchestrator.InitAll] fails, no services are started and no rollback is
// needed. If [Orchestrator.StartAll] fails, all successfully started services
// are stopped in reverse order (rollback). [Orchestrator.StopAll] is idempotent
// and can be called multiple times safely.
//
// # Example (Orchestrator)
//
//	package main
//
//	import (
//	    "context"
//	    "fmt"
//	    "time"
//
//	    "github.com/lemon4ksan/foundation/async/lifecycle"
//	)
//
//	type DatabaseService struct{}
//	func (d *DatabaseService) Name() string            { return "db" }
//	func (d *DatabaseService) Init(ctx context.Context) error  { return nil }
//	func (d *DatabaseService) Start(ctx context.Context) error { return nil }
//	func (d *DatabaseService) Stop(ctx context.Context) error  { return nil }
//
//	type APIService struct{}
//	func (a *APIService) Name() string                      { return "api" }
//	func (a *APIService) Init(ctx context.Context) error    { return nil }
//	func (a *APIService) Start(ctx context.Context) error   { return nil }
//	func (a *APIService) Stop(ctx context.Context) error    { return nil }
//	func (a *APIService) Dependencies() []string            { return []string{"db"} }
//
//	func main() {
//	    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	    defer cancel()
//
//	    orch := lifecycle.NewOrchestrator()
//	    orch.Register(&DatabaseService{})
//	    orch.Register(&APIService{})
//
//	    if err := orch.InitAll(ctx); err != nil {
//	        panic(err)
//	    }
//	    if err := orch.StartAll(ctx); err != nil {
//	        panic(err)
//	    }
//	    defer orch.StopAll(ctx)
//	}
//
// # Example (Behavior Adaptation)
//
//	type MyLoop struct{}
//	func (m *MyLoop) Name() string { return "my-loop" }
//	func (m *MyLoop) Run(ctx context.Context) error {
//	    for {
//	        select {
//	        case <-ctx.Done():
//	            return ctx.Err()
//	        default:
//	            // Perform work...
//	        }
//	    }
//	}
//
//	// Register as a lifecycle service:
//	orch.Register(lifecycle.AsService(&MyLoop{}))
package lifecycle
