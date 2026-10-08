//go:build !race

package limiter_test

import (
	"context"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/sync/limiter"
)

func TestAdaptive_Allocations(t *testing.T) {
	adLim := limiter.NewAdaptive(10.0)
	ctx := context.Background()

	allocs := testing.AllocsPerRun(1000, func() {
		_ = adLim.Acquire(ctx)
		adLim.Release(time.Millisecond)
	})

	if allocs > 0 {
		// Log the allocations to know what it is.
		t.Logf("Adaptive allocates %v per op", allocs)
	}
}
