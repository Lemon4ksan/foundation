package limiter_test

import (
	"context"
	"time"

	"github.com/lemon4ksan/foundation/async/rate"
	"github.com/lemon4ksan/foundation/sync/limiter"
)

func ExampleAdaptiveLimiter() {
	adLim := limiter.NewAdaptiveLimiter(10.0)
	ctx := context.Background()

	// Acquire concurrency slot.
	if err := adLim.Acquire(ctx); err != nil {
		return
	}

	start := time.Now()
	// simulate work
	time.Sleep(10 * time.Millisecond)

	// Release slot and adjust capacity based on latency
	adLim.Release(time.Since(start))
}

func ExampleKeyedLimiter() {
	// 10 req/sec, burst 20, TTL 5 minutes of inactivity
	kl := limiter.NewKeyedLimiter[string](rate.Limit(10), 20, 5*time.Minute)
	defer kl.Close()

	ctx := context.Background()
	ip := "192.168.1.1"

	if err := kl.Wait(ctx, ip); err != nil {
		return
	}
}
