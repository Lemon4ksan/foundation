//go:build !race

package limiter_test

import (
	"context"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/sync/limiter"
)

func TestAdaptiveLimiter_Allocations(t *testing.T) {
	adLim := limiter.NewAdaptiveLimiter(10.0)
	ctx := context.Background()

	allocs := testing.AllocsPerRun(1000, func() {
		_ = adLim.Acquire(ctx)
		adLim.Release(time.Millisecond)
	})

	if allocs > 0 {
		// Log the allocations to know what it is.
		t.Logf("AdaptiveLimiter allocates %v per op", allocs)
	}
}
