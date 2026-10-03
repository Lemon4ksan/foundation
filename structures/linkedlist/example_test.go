package list_test

import (
	"fmt"

	list "github.com/lemon4ksan/foundation/structures/linkedlist"
)

func ExampleList() {
	l := list.New[string]()
	l.PushBack("hello")
	l.PushFront("world")

	for v := range l.Values() {
		fmt.Println(v)
	}
	// Output:
	// world
	// hello
}
