//go:build !race

package bufkit_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/bufkit"
)

func TestAllocations(t *testing.T) {
	chain := bufkit.NewChain()
	defer chain.Release()

	allocs := testing.AllocsPerRun(100, func() {
		chain.WriteString("hello world")
		chain.Reset()
	})
	if allocs > 0 {
		t.Errorf("expected 0 allocations, got %v", allocs)
	}
}
