//go:build !race

package generic_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/generic"
)

func TestGeneric_Allocations(t *testing.T) {
	items := []string{"a", "b", "c", "d"}

	allocs := testing.AllocsPerRun(100, func() {
		_ = generic.Chunked(items, 2)
	})

	// Chunked shares the backing slice, making only 1 allocation for the outer slice of slices.
	if allocs > 1 {
		t.Errorf("expected 1 allocation for Chunked, got %v", allocs)
	}
}
