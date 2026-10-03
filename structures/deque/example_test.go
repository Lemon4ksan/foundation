package deque_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/structures/deque"
)

func ExampleDeque() {
	d := deque.New[int]()
	d.PushBack(1)
	d.PushFront(2)

	front, _ := d.PopFront()
	back, _ := d.PopBack()

	fmt.Printf("Front: %d, Back: %d\n", front, back)
	// Output: Front: 2, Back: 1
}
