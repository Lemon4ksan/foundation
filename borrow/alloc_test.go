//go:build !race

package borrow_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/borrow"
)

func TestAllocations(t *testing.T) {
	allocs := testing.AllocsPerRun(100, func() {
		borrow.Scoped(func(s *borrow.Scope) (string, error) {
			box := borrow.Alloc[int](s)
			mut := box.BorrowMut()
			mut.Write(42)
			return "", nil
		})
	})
	if allocs > 0 {
		t.Errorf("expected 0 allocations, got %v", allocs)
	}
}
