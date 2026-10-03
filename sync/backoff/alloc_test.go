//go:build !race

package backoff_test

import (
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/sync/backoff"
)

func TestExponentialBackoff_Allocations(t *testing.T) {
	bo := backoff.NewExponential(100*time.Millisecond, 5*time.Second, 2.0).WithFullJitter()
	allocs := testing.AllocsPerRun(1000, func() {
		_ = bo.NextDelay(5)
	})
	if allocs > 0 {
		t.Errorf("Expected 0 allocations, got %v", allocs)
	}
}
