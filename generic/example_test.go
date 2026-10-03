package generic_test

import (
	"context"
	"fmt"
	"time"

	"github.com/lemon4ksan/foundation/generic"
)

func ExampleParallelMap() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	inputs := []string{"item1", "item2"}

	// Process items concurrently, limiting to 2 parallel goroutines
	results := generic.ParallelMap(ctx, inputs, 2, func(c context.Context, item string) string {
		return item + "_processed"
	})

	fmt.Println("Processed:", results)

	// Output:
	// Processed: [item1_processed item2_processed]
}
