package ringbuffer_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/structures/ringbuffer"
)

func ExampleRingBuffer() {
	rb := new(ringbuffer.RingBuffer[int])
	rb.Init(2)
	rb.PushBack(1)
	rb.PushBack(2)

	fmt.Printf("Popped: %d\n", rb.PopFront())
	// Output: Popped: 1
}
