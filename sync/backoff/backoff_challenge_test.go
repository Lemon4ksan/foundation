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
)

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
