//go:build !race

package brotli_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/lemon4ksan/foundation/codec/compress/brotli"
)

func TestAllocationsAmortized(t *testing.T) {
	var buf bytes.Buffer
	w := brotli.NewWriter(&buf)
	w.Write([]byte("test data for allocations"))
	w.Close()
	data := buf.Bytes()

	br := bytes.NewReader(data)
	r := brotli.NewReader(br)

	allocs := testing.AllocsPerRun(100, func() {
		br.Reset(data)
		r.Reset(br)
		io.Copy(io.Discard, r)
	})

	if allocs > 0 {
		t.Errorf("expected 0 allocs per run with Reset, got %v", allocs)
	}
}
