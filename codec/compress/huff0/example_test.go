package huff0_test

import (
	"bytes"
	"fmt"
	"log"

	"github.com/lemon4ksan/foundation/codec/compress/huff0"
)

func ExampleCompress1X() {
	data := bytes.Repeat([]byte("hello huff0 "), 100)

	var s huff0.Scratch

	compressed, _, err := huff0.Compress1X(data, &s)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("compressed size: %d", len(compressed))
	// Output: compressed size: 462
}
