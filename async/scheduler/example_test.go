package scheduler_test

import (
	"context"
	"fmt"
	"time"

	"github.com/lemon4ksan/foundation/async/scheduler"
)

func ExampleScheduler() {
	s := scheduler.New()

	t1 := s.AcquireTask()
	t1.NextRun = time.Now().Add(time.Minute)
	t1.Interval = time.Minute
	t1.Execute = func(ctx context.Context) error {
		fmt.Println("Running cache eviction")
		return nil
	}
	s.Schedule(t1)

	t2 := s.AcquireTask()
	t2.NextRun = time.Now().Add(5 * time.Second)
	t2.Execute = func(ctx context.Context) error {
		fmt.Println("Warming up database connection pool...")
		return nil
	}
	s.Schedule(t2)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start scheduler in background
	go s.Start(ctx)

	time.Sleep(10 * time.Millisecond) // Give it time to schedule
}
