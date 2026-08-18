// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package pool provides a dynamic, auto-scaling worker pool.
//
// The worker pool manages worker goroutines dynamically.
// It scales up workers under load (up to a configured maximum)
// and scales them down (to a configured minimum) after a period of idleness,
// preventing resources from leaking when the system is quiet.
//
// # Scaling Behavior
//
// The Pool scales dynamically based on the [Config] parameters:
//   - MinWorkers: The baseline number of workers that always remain active.
//   - MaxWorkers: The absolute upper bound on the number of concurrently running workers.
//   - IdleTimeout: How long a worker above the MinWorkers threshold can remain idle
//     without receiving a task before it gracefully shuts down.
//
// When a task is submitted via [Pool.Submit]:
//  1. If the current worker count is below MaxWorkers, and all current workers
//     are busy OR there are tasks accumulating in the queue, a new worker is spawned.
//  2. If the current worker count has reached MaxWorkers, the task is queued until
//     an active worker becomes available.
//  3. If the queue is full (exceeds QueueLimit), [Pool.Submit] returns [ErrQueueFull].
//
// # Task Panic Safety
//
// If a user task panics during execution, the worker goroutine recovers safely,
// records the panic value into the task's [Future], and remains alive to process
// subsequent tasks. The panic does not crash the worker or the pool.
//
// # Graceful Shutdown
//
// Calling [Pool.Close] stops the pool from accepting new tasks and closes the queue.
// It blocks the caller until all currently queued and executing tasks are completed,
// ensuring no data or progress is lost.
//
// # Example
//
//	package main
//
//	import (
//		"context"
//		"fmt"
//		"time"
//
//		"github.com/lemon4ksan/foundation/async/pool"
//	)
//
//	func main() {
//		// Initialize a pool that can scale from 2 to 10 workers.
//		p := pool.New[string](pool.Config{
//			MinWorkers:  2,
//			MaxWorkers:  10,
//			IdleTimeout: 2 * time.Second,
//			QueueLimit:  50,
//		})
//		defer p.Close()
//
//		ctx := context.Background()
//
//		// Submit a task
//		future, err := p.Submit(ctx, func(taskCtx context.Context) (string, error) {
//			time.Sleep(100 * time.Millisecond)
//			return "task-result", nil
//		})
//		if err != nil {
//			fmt.Printf("Failed to submit task: %v\n", err)
//			return
//		}
//
//		// Block and wait for the result
//		res, err := future.Get(ctx)
//		if err != nil {
//			fmt.Printf("Task failed: %v\n", err)
//			return
//		}
//		fmt.Printf("Got result: %s\n", res)
//	}
package pool
