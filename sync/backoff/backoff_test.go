// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package backoff_test

import (
	"math"
	"sync"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/sync/backoff"
	"github.com/lemon4ksan/foundation/testing/assert"
)

func TestExponential_Deterministic(t *testing.T) {
	t.Parallel()

	bo := backoff.NewExponential(100*time.Millisecond, 1*time.Second, 2.0)

	assert.Equal(t, 100*time.Millisecond, bo.NextDelay(1))
	assert.Equal(t, 200*time.Millisecond, bo.NextDelay(2))
	assert.Equal(t, 400*time.Millisecond, bo.NextDelay(3))
	assert.Equal(t, 800*time.Millisecond, bo.NextDelay(4))
	assert.Equal(t, 1000*time.Millisecond, bo.NextDelay(5)) // capped at max
	assert.Equal(t, 1000*time.Millisecond, bo.NextDelay(10))
	assert.Equal(t, 1000*time.Millisecond, bo.NextDelay(1000)) // overflow cap
}

func TestExponential_DefaultsAndEdgeCases(t *testing.T) {
	t.Parallel()

	// Testing zero/negative parameters for default fallback
	bo := backoff.NewExponential(0, 0, 0)
	assert.Equal(t, 100*time.Millisecond, bo.Initial)
	assert.Equal(t, 30*time.Second, bo.Max)
	assert.Equal(t, 2.0, bo.Factor)

	// Testing max < initial
	bo2 := backoff.NewExponential(5*time.Second, 1*time.Second, 1.0)
	assert.Equal(t, 5*time.Second, bo2.Max)
	assert.Equal(t, 2.0, bo2.Factor)
}

func TestExponential_FullJitter(t *testing.T) {
	t.Parallel()

	bo := backoff.NewExponential(100*time.Millisecond, 1*time.Second, 2.0).WithFullJitter()

	for attempt := 1; attempt <= 5; attempt++ {
		delay := bo.NextDelay(attempt)
		assert.GreaterOrEqual(t, delay, time.Duration(0))
		assert.LessOrEqual(t, delay, 1*time.Second)
	}

	// Zero delay jitter
	zeroBo := backoff.NewExponential(1, 1, 2.0).WithFullJitter()
	zeroBo.Initial = 0
	assert.Equal(t, time.Duration(0), zeroBo.NextDelay(1))
}

func TestExponential_EqualJitter(t *testing.T) {
	t.Parallel()

	bo := backoff.NewExponential(100*time.Millisecond, 1*time.Second, 2.0).WithEqualJitter()

	for attempt := 1; attempt <= 5; attempt++ {
		delay := bo.NextDelay(attempt)
		assert.GreaterOrEqual(t, delay, 50*time.Millisecond)
		assert.LessOrEqual(t, delay, 1*time.Second)
	}

	// Zero delay equal jitter
	zeroBo := backoff.NewExponential(1, 1, 2.0).WithEqualJitter()
	zeroBo.Initial = 0
	assert.Equal(t, time.Duration(0), zeroBo.NextDelay(1))
}

func TestExponential_DecorrelatedJitter(t *testing.T) {
	t.Parallel()

	bo := backoff.NewExponential(50*time.Millisecond, 500*time.Millisecond, 2.0).WithDecorrelatedJitter()

	for attempt := 1; attempt <= 10; attempt++ {
		delay := bo.NextDelay(attempt)
		assert.GreaterOrEqual(t, delay, 50*time.Millisecond)
		assert.LessOrEqual(t, delay, 500*time.Millisecond)
	}

	bo.Reset()
	assert.Equal(t, bo.Initial, bo.NextDelay(1))

	// Test delta <= 0 in decorrelated
	boFlat := backoff.NewExponential(100*time.Millisecond, 100*time.Millisecond, 2.0).WithDecorrelatedJitter()
	d := boFlat.NextDelay(2)
	assert.Equal(t, 100*time.Millisecond, d)
}

func TestLinear(t *testing.T) {
	t.Parallel()

	bo := backoff.NewLinear(100*time.Millisecond, 500*time.Millisecond, 100*time.Millisecond)

	assert.Equal(t, 100*time.Millisecond, bo.NextDelay(1))
	assert.Equal(t, 200*time.Millisecond, bo.NextDelay(2))
	assert.Equal(t, 300*time.Millisecond, bo.NextDelay(3))
	assert.Equal(t, 400*time.Millisecond, bo.NextDelay(4))
	assert.Equal(t, 500*time.Millisecond, bo.NextDelay(5))
	assert.Equal(t, 500*time.Millisecond, bo.NextDelay(6)) // capped at max

	bo.Reset()

	// Defaults and edge cases
	boDef := backoff.NewLinear(0, 0, 0)
	assert.Equal(t, 100*time.Millisecond, boDef.Initial)
	assert.Equal(t, 10*time.Second, boDef.Max)
	assert.Equal(t, 100*time.Millisecond, boDef.Step)

	// Jitter
	boJitter := backoff.NewLinear(100*time.Millisecond, 500*time.Millisecond, 100*time.Millisecond).WithFullJitter()
	for i := 1; i <= 5; i++ {
		d := boJitter.NextDelay(i)
		assert.GreaterOrEqual(t, d, time.Duration(0))
		assert.LessOrEqual(t, d, 500*time.Millisecond)
	}
}

func TestConstant(t *testing.T) {
	t.Parallel()

	bo := backoff.NewConstant(250 * time.Millisecond)

	assert.Equal(t, 250*time.Millisecond, bo.NextDelay(1))
	assert.Equal(t, 250*time.Millisecond, bo.NextDelay(5))

	bo.Reset()

	boJitter := backoff.NewConstant(250 * time.Millisecond).WithFullJitter()
	for i := 1; i <= 5; i++ {
		d := boJitter.NextDelay(i)
		assert.GreaterOrEqual(t, d, time.Duration(0))
		assert.LessOrEqual(t, d, 250*time.Millisecond)
	}
}

func BenchmarkExponential_ZeroAlloc(b *testing.B) {
	bo := backoff.NewExponential(100*time.Millisecond, 10*time.Second, 2.0).WithFullJitter()

	b.ReportAllocs()
	b.ResetTimer()

	var i int
	for b.Loop() {
		_ = bo.NextDelay((i % 10) + 1)
		i++
	}
}

// TestBackoff_StatelessConcurrentSafety tests that Linear and Constant backoff policies
// are safely callable concurrently from multiple goroutines without data races.
func TestBackoff_StatelessConcurrentSafety(t *testing.T) {
	linear := backoff.NewLinear(10*time.Millisecond, 1*time.Second, 10*time.Millisecond)
	constant := backoff.NewConstant(50 * time.Millisecond).WithFullJitter()

	const (
		numWorkers = 64
		iterations = 1000
	)

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for range numWorkers {
		go func() {
			defer wg.Done()
			for i := 1; i <= iterations; i++ {
				d1 := linear.NextDelay(i)
				if d1 < 0 || d1 > 1*time.Second {
					t.Errorf("linear delay out of bounds: %v", d1)
				}

				d2 := constant.NextDelay(i)
				if d2 < 0 || d2 > 50*time.Millisecond {
					t.Errorf("constant delay out of bounds: %v", d2)
				}
			}
		}()
	}

	wg.Wait()
}

// TestBackoff_Exponential_OverflowSafety tests that huge attempt numbers do not overflow
// or cause negative / NaN / infinite durations.
func TestBackoff_Exponential_OverflowSafety(t *testing.T) {
	exp := backoff.NewExponential(100*time.Millisecond, 30*time.Second, 2.0)

	attempts := []int{
		math.MinInt,
		-100,
		-1,
		0,
		1,
		2,
		63,
		64,
		100,
		1000,
		1000000,
		math.MaxInt,
	}

	for _, att := range attempts {
		d := exp.NextDelay(att)
		if d < 0 {
			t.Errorf("attempt %d resulted in negative delay: %v", att, d)
		}
		if d > 30*time.Second {
			t.Errorf("attempt %d exceeded max delay: %v", att, d)
		}
	}
}

// TestBackoff_Linear_LargeAttemptOverflow tests whether extremely large attempt indices
// in Linear backoff cause integer multiplication overflow leading to negative delays.
func TestBackoff_Linear_LargeAttemptOverflow(t *testing.T) {
	lin := backoff.NewLinear(100*time.Millisecond, 10*time.Second, 1*time.Second)

	attempts := []int{
		10_000_000_000, // 10 billion attempts
		math.MaxInt32,
		math.MaxInt64 / int(time.Second),
		math.MaxInt64,
	}

	for _, att := range attempts {
		d := lin.NextDelay(att)
		if d < 0 {
			t.Errorf("attempt %d caused integer overflow resulting in negative delay: %v", att, d)
		}
		if d > 10*time.Second {
			t.Errorf("attempt %d exceeded max: %v", att, d)
		}
	}
}

func TestExponential_Allocations(t *testing.T) {
	bo := backoff.NewExponential(100*time.Millisecond, 5*time.Second, 2.0).WithFullJitter()
	allocs := testing.AllocsPerRun(1000, func() {
		_ = bo.NextDelay(5)
	})
	if allocs > 0 {
		t.Errorf("Expected 0 allocations, got %v", allocs)
	}
}

func TestReproduce_ExponentialBackoffDecreaseBug(t *testing.T) {
	// If max is 0, it defaults to 30s. If initial is larger than the default max (e.g., 60s),
	// the fallback logic in NewExponential fails to clamp initial or bump max to match initial.
	// As a result, Initial=60s and Max=30s.
	bo := backoff.NewExponential(60*time.Second, 0, 2.0)

	delay1 := bo.NextDelay(1)
	delay2 := bo.NextDelay(2)

	// A backoff delay should never decrease as attempt increases.
	// However, due to this bug, delay1 is 60s and delay2 gets clamped to Max (30s).
	if delay2 < delay1 {
		t.Fatalf("Bug triggered: delay decreased from %v to %v!", delay1, delay2)
	}
}
