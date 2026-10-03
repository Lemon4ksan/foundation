package rate_test

import (
	"context"
	"fmt"
	"time"

	"github.com/lemon4ksan/foundation/async/rate"
)

func ExampleLimiter() {
	// Limit to 10 operations per second with burst capability of 3
	limiter := rate.NewLimiter(rate.Limit(10), 3)
	ctx := context.Background()

	for i := range 5 {
		if err := limiter.Wait(ctx); err != nil {
			panic(err)
		}
		fmt.Printf("Processed request %d\n", i)
	}
	// Output:
	// Processed request 0
	// Processed request 1
	// Processed request 2
	// Processed request 3
	// Processed request 4
}

func ExampleSometimes() {
	sometimes := rate.Sometimes{First: 3, Interval: 10 * time.Second}
	var count int

	for i := 0; i < 5; i++ {
		sometimes.Do(func() {
			count++
		})
	}
	fmt.Println(count)
	// Output: 3
}
