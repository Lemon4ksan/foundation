package brotli_test

import (
	"bytes"
	"fmt"
	"io"

	"github.com/lemon4ksan/foundation/codec/compress/brotli"
)

func ExampleNewWriter() {
	var buf bytes.Buffer

	w := brotli.NewWriter(&buf)
	_, err := w.Write([]byte("hello brotli world"))
	if err != nil {
		panic(err)
	}
	w.Close()

	r := brotli.NewReader(&buf)
	out, err := io.ReadAll(r)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(out))
	// Output: hello brotli world
}
