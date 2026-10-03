package dedup_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lemon4ksan/foundation/async/dedup"
)

func ExampleGroup_Do() {
	g := &dedup.Group[string, string]{}
	ctx := context.Background()

	var callCounter int32
	var wg sync.WaitGroup

	fn := func(workerCtx context.Context) (string, error) {
		atomic.AddInt32(&callCounter, 1)
		time.Sleep(50 * time.Millisecond)
		return "shared-value", nil
	}

	for i := range 5 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			res, err := g.Do(ctx, "my-key", fn)
			if err != nil {
				return
			}
			// Just printing one of them to keep output deterministic
			if id == 0 {
				fmt.Printf("Worker %d got: %s\n", id, res)
			}
		}(i)
	}

	wg.Wait()
	fmt.Printf("Total executions: %d\n", atomic.LoadInt32(&callCounter))

	// Output:
	// Worker 0 got: shared-value
	// Total executions: 1
}
