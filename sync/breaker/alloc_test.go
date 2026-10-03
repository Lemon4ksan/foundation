//go:build !race

package breaker_test

import (
	"context"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/sync/breaker"
)

func TestBreakerAllocations(t *testing.T) {
	cb := breaker.New[int](breaker.Config{
		MinRequests: 10000,
		Window:      time.Hour,
	})

	ctx := context.Background()

	fn := func(ctx context.Context) (int, error) {
		return 1, nil
	}

	// Pre-warm
	for i := 0; i < 1024; i++ {
		_, _ = cb.Do(ctx, fn)
	}

	allocs := testing.AllocsPerRun(100, func() {
		_, _ = cb.Do(ctx, fn)
	})

	if allocs > 0 {
		t.Errorf("expected 0 amortized allocations, got %v", allocs)
	}
}
