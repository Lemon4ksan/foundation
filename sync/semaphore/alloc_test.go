//go:build !race

package semaphore_test

import (
	"context"
	"testing"

	"github.com/lemon4ksan/foundation/sync/semaphore"
)

func TestSemaphore_Allocations(t *testing.T) {
	sem := semaphore.New(1)
	ctx := context.Background()

	allocs := testing.AllocsPerRun(1000, func() {
		_ = sem.Acquire(ctx)
		sem.Release()
	})

	if allocs > 0 {
		t.Logf("Semaphore hotpath allocates %v per op", allocs)
	}
}
