package backoff_test

import (
	"context"
	"fmt"
	"time"

	"github.com/lemon4ksan/foundation/sync/backoff"
)

func ExampleExponentialBackoff() {
	bo := backoff.NewExponential(100*time.Millisecond, 5*time.Second, 2.0).WithFullJitter()
	ctx := context.Background()

	// In this example, we mock a retry loop. We cannot predict exact times,
	// so there is no Output comment.
	for attempt := 1; attempt <= 2; attempt++ {
		delay := bo.NextDelay(attempt)
		fmt.Printf("Attempt %d delay calculated\n", attempt)

		select {
		case <-time.After(time.Microsecond): // shortened for the test
			_ = delay
		case <-ctx.Done():
			return
		}
	}
}
