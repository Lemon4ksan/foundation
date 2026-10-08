// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package backoff

import (
	"math"
	"time"

	"github.com/lemon4ksan/foundation/silicon/randkit"
)

// JitterMode defines the randomization algorithm applied to backoff intervals.
type JitterMode uint8

const (
	// JitterNone applies no randomization (deterministic delay).
	JitterNone JitterMode = iota
	// JitterFull randomizes delay uniformly in range [0, baseDelay].
	JitterFull
	// JitterEqual splits delay into half deterministic and half uniform randomized [baseDelay/2, baseDelay].
	JitterEqual
	// JitterDecorrelated updates state: next = min(max, rand(initial, prev * 3)).
	JitterDecorrelated
)

// Policy represents a stateful or stateless backoff delay generator.
type Policy interface {
	// NextDelay calculates the sleep duration for the given 1-based attempt index.
	NextDelay(attempt int) time.Duration
	// Reset resets any internal stateful state (e.g. for decorrelated jitter).
	Reset()
}

// Exponential computes backoff growing by factor with an optional jitter distribution.
type Exponential struct {
	Initial   time.Duration
	Max       time.Duration
	Factor    float64
	Jitter    JitterMode
	lastSleep time.Duration
}

// NewExponential creates an [Exponential] with reasonable defaults (factor=2.0).
func NewExponential(initial, max time.Duration, factor float64) *Exponential {
	if initial <= 0 {
		initial = 100 * time.Millisecond
	}

	if max <= 0 || max < initial {
		max = 30 * time.Second
	}

	if factor <= 1.0 {
		factor = 2.0
	}

	return &Exponential{
		Initial:   initial,
		Max:       max,
		Factor:    factor,
		Jitter:    JitterNone,
		lastSleep: initial,
	}
}

// WithFullJitter sets JitterMode to Full Jitter (AWS recommended standard).
func (b *Exponential) WithFullJitter() *Exponential {
	b.Jitter = JitterFull
	return b
}

// WithEqualJitter sets JitterMode to Equal Jitter.
func (b *Exponential) WithEqualJitter() *Exponential {
	b.Jitter = JitterEqual
	return b
}

// WithDecorrelatedJitter sets JitterMode to Decorrelated Jitter.
func (b *Exponential) WithDecorrelatedJitter() *Exponential {
	b.Jitter = JitterDecorrelated
	return b
}

// Reset resets the stateful sleep history.
func (b *Exponential) Reset() {
	b.lastSleep = b.Initial
}

// NextDelay calculates the duration for attempt (1-based).
func (b *Exponential) NextDelay(attempt int) time.Duration {
	if attempt <= 1 {
		return b.applyJitter(b.Initial)
	}

	if b.Jitter == JitterDecorrelated {
		// next = min(max, rand(initial, lastSleep * 3))
		maxRange := min(b.lastSleep*3, b.Max)
		delta := maxRange - b.Initial

		var sleep time.Duration
		if delta > 0 {
			sleep = b.Initial + randkit.Jitter(delta)
		} else {
			sleep = b.Initial
		}

		b.lastSleep = sleep

		return sleep
	}

	mult := math.Pow(b.Factor, float64(attempt-1))
	computed := float64(b.Initial) * mult

	var baseDelay time.Duration
	if computed >= float64(b.Max) || math.IsInf(computed, 1) {
		baseDelay = b.Max
	} else {
		baseDelay = time.Duration(computed)
	}

	return b.applyJitter(baseDelay)
}

func (b *Exponential) applyJitter(baseDelay time.Duration) time.Duration {
	switch b.Jitter {
	case JitterFull:
		if baseDelay <= 0 {
			return 0
		}

		return randkit.Jitter(baseDelay)

	case JitterEqual:
		if baseDelay <= 0 {
			return 0
		}

		half := baseDelay / 2

		return half + randkit.Jitter(half)

	default:
		return baseDelay
	}
}

// Linear computes backoff growing linearly by step on each attempt.
type Linear struct {
	Initial time.Duration
	Max     time.Duration
	Step    time.Duration
	Jitter  JitterMode
}

// NewLinear creates a [Linear] calculator.
func NewLinear(initial, max, step time.Duration) *Linear {
	if initial <= 0 {
		initial = 100 * time.Millisecond
	}

	if max <= 0 || max < initial {
		max = 10 * time.Second
	}

	if step <= 0 {
		step = initial
	}

	return &Linear{
		Initial: initial,
		Max:     max,
		Step:    step,
		Jitter:  JitterNone,
	}
}

// WithFullJitter enables Full Jitter on linear backoff.
func (b *Linear) WithFullJitter() *Linear {
	b.Jitter = JitterFull
	return b
}

// Reset is a no-op for stateless linear backoff.
func (b *Linear) Reset() {}

// NextDelay calculates the duration for attempt (1-based).
func (b *Linear) NextDelay(attempt int) time.Duration {
	if attempt <= 1 {
		return b.applyJitter(b.Initial)
	}

	if b.Step <= 0 {
		return b.applyJitter(min(b.Initial, b.Max))
	}

	if b.Max > b.Initial {
		maxSafeAttempts := (b.Max - b.Initial) / b.Step
		if time.Duration(attempt-1) > maxSafeAttempts {
			return b.applyJitter(b.Max)
		}
	} else {
		return b.applyJitter(b.Max)
	}

	return b.applyJitter(min(b.Initial+time.Duration(attempt-1)*b.Step, b.Max))
}

func (b *Linear) applyJitter(baseDelay time.Duration) time.Duration {
	if b.Jitter == JitterFull && baseDelay > 0 {
		return randkit.Jitter(baseDelay)
	}

	return baseDelay
}

// Constant computes a fixed static delay with optional jitter.
type Constant struct {
	Delay  time.Duration
	Jitter JitterMode
}

// NewConstant creates a fixed static delay backoff.
func NewConstant(delay time.Duration) *Constant {
	return &Constant{
		Delay:  delay,
		Jitter: JitterNone,
	}
}

// WithFullJitter enables Full Jitter on constant delay.
func (b *Constant) WithFullJitter() *Constant {
	b.Jitter = JitterFull
	return b
}

// Reset is a no-op for constant backoff.
func (b *Constant) Reset() {}

// NextDelay returns the constant delay.
func (b *Constant) NextDelay(attempt int) time.Duration {
	if b.Jitter == JitterFull && b.Delay > 0 {
		return randkit.Jitter(b.Delay)
	}

	return b.Delay
}
