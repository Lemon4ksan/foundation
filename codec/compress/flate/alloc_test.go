//go:build !race

package flate_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/lemon4ksan/foundation/codec/compress/flate"
)

func TestAllocationsAmortized(t *testing.T) {
	var buf bytes.Buffer
	w, _ := flate.NewWriter(&buf, 1)
	w.Write([]byte("test data for allocations"))
	w.Close()
	data := buf.Bytes()

	br := bytes.NewReader(data)
	r := flate.NewReader(br)
	defer r.Close()

	// Ensure decompressor implements Reset
	if resetter, ok := r.(interface{ Reset(io.Reader, []byte) error }); ok {
		allocs := testing.AllocsPerRun(100, func() {
			br.Reset(data)
			resetter.Reset(br, nil)
			io.Copy(io.Discard, r)
		})
		if allocs > 0 {
			t.Errorf("expected 0 allocs per run with Reset, got %v", allocs)
		}
	} else {
		t.Skip("Reader does not implement Reset(io.Reader, []byte) error")
	}
}
