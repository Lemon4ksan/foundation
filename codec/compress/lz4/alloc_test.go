//go:build !race

package lz4_test

import (
	"bytes"
	"testing"

	"github.com/lemon4ksan/foundation/codec/compress/lz4"
)

func TestCompressBlockAllocs(t *testing.T) {
	data := bytes.Repeat([]byte("hello lz4 compression! "), 10)
	dst := make([]byte, lz4.CompressBlockBound(len(data)))

	allocs := testing.AllocsPerRun(10, func() {
		_, _ = lz4.CompressBlock(data, dst)
	})

	if allocs > 0 {
		t.Errorf("CompressBlock allocated %v times", allocs)
	}
}
