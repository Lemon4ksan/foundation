// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package task_test

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/lemon4ksan/foundation/async/task"
)

func ExampleTracker_callback() {
	ctx := context.Background()
	mgr := task.NewTracker[uint64, string](100)
	id := mgr.NextID()

	var wg sync.WaitGroup
	wg.Add(1)

	err := mgr.Add(id, func(ctx context.Context, res string, err error) {
		defer wg.Done()
		if err != nil {
			log.Printf("Job %d failed: %v", id, err)
			return
		}
		fmt.Printf("Job %d received: %s\n", id, res)
	})
	if err != nil {
		log.Fatalf("Failed to add job: %v", err)
	}

	// Simulate asynchronous response arrival
	mgr.Resolve(ctx, id, "Hello, World!", nil)

	wg.Wait()
	// Output:
	// Job 1 received: Hello, World!
}

func ExampleTracker_blocking() {
	ctx := context.Background()
	mgr := task.NewTracker[uint64, string](0)
	id := mgr.NextID()

	// Configure with WithWait and a 1-second timeout limit
	err := mgr.Add(id, nil, task.WithWait[string](), task.WithTimeout[string](time.Second))
	if err != nil {
		log.Fatalf("Failed to add job: %v", err)
	}

	// Block and wait for resolution synchronously
	go func() {
		time.Sleep(10 * time.Millisecond)
		mgr.Resolve(ctx, id, "Hello from background!", nil)
	}()

	res, err := mgr.WaitFor(ctx, id)
	if err != nil {
		log.Fatalf("Job failed: %v", err)
	}
	fmt.Println("Result:", res)
	// Output:
	// Result: Hello from background!
}
