package flate_test

import (
	"bytes"
	"fmt"
	"io"

	"github.com/lemon4ksan/foundation/codec/compress/flate"
)

func ExampleNewWriter() {
	var buf bytes.Buffer

	w, err := flate.NewWriter(&buf, 1)
	if err != nil {
		panic(err)
	}
	_, err = w.Write([]byte("hello flate world"))
	if err != nil {
		panic(err)
	}
	w.Close()

	r := flate.NewReader(&buf)
	defer r.Close()

	out, err := io.ReadAll(r)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(out))
	// Output: hello flate world
}
