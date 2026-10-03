//go:build !race

package fse_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/codec/compress/fse"
)

func TestCompressAllocs(t *testing.T) {
	data := []byte("aaabbbbccccccdddddddd")
	var s fse.Scratch

	// Compress once to initialize the scratch
	_, _ = fse.Compress(data, &s)

	allocs := testing.AllocsPerRun(10, func() {
		_, _ = fse.Compress(data, &s)
	})

	if allocs > 0 {
		t.Errorf("Compress allocated %v times", allocs)
	}
}
