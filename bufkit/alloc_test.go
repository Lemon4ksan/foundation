//go:build !race

package bufkit_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/bufkit"
)

func TestAllocations(t *testing.T) {
	allocs := testing.AllocsPerRun(100, func() {
		chain := bufkit.NewChain()
		chain.WriteString("hello world")
		chain.Release()
	})
	if allocs > 0 {
		t.Errorf("expected 0 allocations, got %v", allocs)
	}
}
