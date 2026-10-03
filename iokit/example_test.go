package iokit_test

import (
	"bytes"
	"fmt"
	"io"

	"github.com/lemon4ksan/foundation/iokit"
)

func ExampleMultiReadBody() {
	data := []byte("payload to inspect and replay")

	// Create a replayable body that falls back to disk if > 1024 bytes.
	// NewMultiReadBody returns (io.ReadCloser, error)
	rc, err := iokit.NewMultiReadBody(io.NopCloser(bytes.NewReader(data)), 1024, false)
	if err != nil {
		panic(err)
	}
	body, ok := rc.(interface {
		io.ReadCloser
		ReallyClose()
	})
	if !ok {
		panic("expected ReallyClose method")
	}
	defer body.ReallyClose()

	// First pass: inspect prefix
	prefix := make([]byte, 7)
	_, _ = io.ReadFull(body, prefix)
	fmt.Printf("Prefix: %s\n", prefix)

	// Reset cursor for full downstream replay by calling Close()
	_ = body.Close()
	all, _ := io.ReadAll(body)
	fmt.Printf("Replayed: %s\n", all)

	// Output:
	// Prefix: payload
	// Replayed: payload to inspect and replay
}
