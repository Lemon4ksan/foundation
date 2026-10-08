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

func ExampleGuarded() {
	g := generic.NewGuarded(100)
	g.Update(func(current int) int {
		return current + 50
	})
	fmt.Println("Value:", g.Get())
	// Output:
	// Value: 150
}

func ExampleBatchLoader() {
	loader := generic.NewBatchLoader(
		10*time.Millisecond,
		func(ctx context.Context, keys []string) (map[string]int, error) {
			res := make(map[string]int, len(keys))
			for _, k := range keys {
				res[k] = len(k)
			}
			return res, nil
		},
	)

	val, err := loader.Load(context.Background(), "hello")
	if err != nil {
		panic(err)
	}
	fmt.Println("Length:", val)
	// Output:
	// Length: 5
}

func ExampleLRU_Values() {
	cache := generic.NewLRU[string, int](3)
	cache.Put("a", 1)
	cache.Put("b", 2)
	cache.Put("c", 3)

	// Access "a" to make it most recently used
	cache.Get("a")

	// Values returns items from most recently used to least recently used
	fmt.Println("Values:", cache.Values())

	// Output:
	// Values: [1 3 2]
}

func ExampleSet_Intersect() {
	s1 := generic.NewSet(1, 2, 3)
	s2 := generic.NewSet(2, 3, 4)

	intersection := s1.Intersect(s2)
	fmt.Println("Has 2:", intersection.Has(2))
	fmt.Println("Has 3:", intersection.Has(3))
	fmt.Println("Has 1:", intersection.Has(1))

	// Output:
	// Has 2: true
	// Has 3: true
	// Has 1: false
}
