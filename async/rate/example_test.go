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

	for range 5 {
		sometimes.Do(func() {
			count++
		})
	}
	fmt.Println(count)
	// Output: 3
}

func ExampleSometimes_once() {
	var s rate.Sometimes
	s.Do(func() { fmt.Println("1") })
	s.Do(func() { fmt.Println("2") })
	s.Do(func() { fmt.Println("3") })
	// Output:
	// 1
}

func ExampleSometimes_first() {
	s := rate.Sometimes{First: 2}
	s.Do(func() { fmt.Println("1") })
	s.Do(func() { fmt.Println("2") })
	s.Do(func() { fmt.Println("3") })
	// Output:
	// 1
	// 2
}

func ExampleSometimes_every() {
	s := rate.Sometimes{Every: 2}
	s.Do(func() { fmt.Println("1") })
	s.Do(func() { fmt.Println("2") })
	s.Do(func() { fmt.Println("3") })
	// Output:
	// 1
	// 3
}
