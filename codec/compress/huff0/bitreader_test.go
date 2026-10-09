//go:build !race

package huff0_test

import (
	"bytes"
	"testing"

	"github.com/lemon4ksan/foundation/codec/compress/huff0"
)

func TestCompressAllocs(t *testing.T) {
	data := bytes.Repeat([]byte("hello huff0 "), 100)
	var s huff0.Scratch

	// Compress once to initialize the scratch
	_, _, _ = huff0.Compress1X(data, &s)

	allocs := testing.AllocsPerRun(10, func() {
		_, _, _ = huff0.Compress1X(data, &s)
	})

	if allocs > 0 {
		t.Errorf("Compress1X allocated %v times", allocs)
	}
}
