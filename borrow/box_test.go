//go:build !race

package borrow_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/borrow"
)

func TestAllocations(t *testing.T) {
	allocs := testing.AllocsPerRun(100, func() {
		borrow.Scoped(func(s *borrow.Scope) (string, error) {
			b := s.AllocBytes(10)
			slice := b.AsSlice()
			slice[0] = 'a'
			return "", nil
		})
	})
	if allocs > 0 {
		t.Errorf("expected 0 allocations, got %v", allocs)
	}
}
