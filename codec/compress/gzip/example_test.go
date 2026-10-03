package gzip_test

import (
	"bytes"
	"fmt"
	"io"
	"log"

	"github.com/lemon4ksan/foundation/codec/compress/gzip"
)

func ExampleWriter() {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	
	_, err := w.Write([]byte("compress this with gzip"))
	if err != nil {
		log.Fatal(err)
	}
	
	if err := w.Close(); err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("compressed len > 0: %v", buf.Len() > 0)
	// Output: compressed len > 0: true
}

func ExampleReader() {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, _ = w.Write([]byte("hello world"))
	w.Close()

	r, err := gzip.NewReader(&buf)
	if err != nil {
		log.Fatal(err)
	}
	
	data, err := io.ReadAll(r)
	if err != nil {
		log.Fatal(err)
	}
	
	if err := r.Close(); err != nil {
		log.Fatal(err)
	}
	
	fmt.Println(string(data))
	// Output: hello world
}
