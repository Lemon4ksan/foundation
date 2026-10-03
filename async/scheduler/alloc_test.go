//go:build !race

package scheduler_test

import (
	"context"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/async/scheduler"
	"github.com/lemon4ksan/foundation/testing/assert"
)

var noop = func(ctx context.Context) error { return nil }

func TestAllocations(t *testing.T) {
	s := scheduler.New()

	task := s.AcquireTask()
	task.NextRun = time.Now().Add(time.Hour)
	task.Execute = noop

	// We do 1000 schedules and then 1000 releases (simulating heap behavior without Start)
	allocs := testing.AllocsPerRun(1000, func() {
		s.Schedule(task)
	})

	assert.Equal(t, float64(0), allocs)
}
