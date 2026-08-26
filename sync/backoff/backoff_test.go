// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package backoff_test

import (
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/testkit/assert"

	"github.com/lemon4ksan/foundation/sync/backoff"
)

func TestExponentialBackoff_Deterministic(t *testing.T) {
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

func TestExponentialBackoff_DefaultsAndEdgeCases(t *testing.T) {
	t.Parallel()

	// Testing zero/negative parameters for default fallback
	bo := backoff.NewExponential(0, 0, 0)
	assert.Equal(t, 100*time.Millisecond, bo.Initial)
	assert.Equal(t, 30*time.Second, bo.Max)
	assert.Equal(t, 2.0, bo.Factor)

	// Testing max < initial
	bo2 := backoff.NewExponential(5*time.Second, 1*time.Second, 1.0)
	assert.Equal(t, 30*time.Second, bo2.Max)
	assert.Equal(t, 2.0, bo2.Factor)
}

func TestExponentialBackoff_FullJitter(t *testing.T) {
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

func TestExponentialBackoff_EqualJitter(t *testing.T) {
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

func TestExponentialBackoff_DecorrelatedJitter(t *testing.T) {
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

func TestLinearBackoff(t *testing.T) {
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

func TestConstantBackoff(t *testing.T) {
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

func BenchmarkExponentialBackoff_ZeroAlloc(b *testing.B) {
	bo := backoff.NewExponential(100*time.Millisecond, 10*time.Second, 2.0).WithFullJitter()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = bo.NextDelay((i % 10) + 1)
	}
}
