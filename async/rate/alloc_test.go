//go:build !race

package rate_test

import (
	"context"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/async/rate"
	"github.com/lemon4ksan/foundation/testing/assert"
)

func TestAllocations(t *testing.T) {
	limiter := rate.NewLimiter(rate.Inf, 1)

	allocs := testing.AllocsPerRun(1000, func() {
		limiter.Allow()
	})
	assert.Equal(t, float64(0), allocs)

	ctx := context.Background()
	allocsWait := testing.AllocsPerRun(1000, func() {
		_ = limiter.Wait(ctx)
	})
	assert.Equal(t, float64(0), allocsWait)

	sometimes := &rate.Sometimes{First: 1, Interval: time.Hour}
	allocsSometimes := testing.AllocsPerRun(1000, func() {
		sometimes.Do(func() {})
	})
	assert.Equal(t, float64(0), allocsSometimes)
}
