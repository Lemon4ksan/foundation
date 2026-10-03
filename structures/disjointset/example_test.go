package disjointset_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/structures/disjointset"
)

func ExampleDisjointSet() {
	ds := disjointset.New(5)
	ds.Union(0, 1)
	ds.Union(1, 2)

	fmt.Printf("0 and 2 connected: %v\n", ds.Connected(0, 2))
	fmt.Printf("0 and 3 connected: %v\n", ds.Connected(0, 3))
	// Output:
	// 0 and 2 connected: true
	// 0 and 3 connected: false
}
