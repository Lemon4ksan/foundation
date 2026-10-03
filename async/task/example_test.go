package task_test

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/lemon4ksan/foundation/async/task"
)

func ExampleManager_callback() {
	mgr := task.NewManager[uint64, string](100)
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
	mgr.Resolve(id, "Hello, World!", nil)

	wg.Wait()
	// Output:
	// Job 1 received: Hello, World!
}

func ExampleManager_blocking() {
	mgr := task.NewManager[uint64, string](0)
	id := mgr.NextID()

	// Configure with WithWait and a 1-second timeout limit
	err := mgr.Add(id, nil, task.WithWait[string](), task.WithTimeout[string](time.Second))
	if err != nil {
		log.Fatalf("Failed to add job: %v", err)
	}

	// Block and wait for resolution synchronously
	go func() {
		time.Sleep(10 * time.Millisecond)
		mgr.Resolve(id, "Hello from background!", nil)
	}()

	res, err := mgr.WaitFor(context.Background(), id)
	if err != nil {
		log.Fatalf("Job failed: %v", err)
	}
	fmt.Println("Result:", res)
	// Output:
	// Result: Hello from background!
}
