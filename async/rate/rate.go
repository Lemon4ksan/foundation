// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rate

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

// Limit defines the maximum frequency of some events.
//
// Limit is represented as number of events per second.
// A zero Limit allows no events.
type Limit float64

// Inf is the infinite rate limit; it allows all events.
const Inf = Limit(math.MaxFloat64)

// Every converts a minimum time interval between events to a Limit.
//
// Executes in O(1) time and is zero-allocation.
func Every(interval time.Duration) Limit {
	if interval <= 0 {
		return Inf
	}
	return 1 / Limit(interval.Seconds())
}

// Limiter controls how frequently events are allowed to happen.
//
// The zero value is valid but rejects all events.
// Read-only operations are safe for concurrent use; mutations require external synchronization unless stated otherwise.
// Note that methods on [Limiter] manage their own internal locks.
type Limiter struct {
	mu     sync.Mutex
	limit  Limit
	burst  int
	tokens float64
	// last is the last time the limiter's tokens field was updated
	last time.Time
	// lastEvent is the latest time of a rate-limited event (past or future)
	lastEvent time.Time
}

// Limit returns the maximum overall event rate.
//
// Executes in O(1) time and is zero-allocation.
func (l *Limiter) Limit() Limit {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.limit
}

// Burst returns the maximum burst size.
//
// Executes in O(1) time and is zero-allocation.
func (l *Limiter) Burst() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.burst
}

// TokensAt returns the number of tokens available at time t.
//
// Executes in O(1) time and is zero-allocation.
func (l *Limiter) TokensAt(t time.Time) float64 {
	l.mu.Lock()
	tokens := l.advance(t) // does not mutate l
	l.mu.Unlock()
	return tokens
}

// Tokens returns the number of tokens available now.
//
// Executes in O(1) time and is zero-allocation.
func (l *Limiter) Tokens() float64 {
	return l.TokensAt(time.Now())
}

// NewLimiter creates a new [Limiter] that allows events up to rate r and permits bursts of at most b tokens.
//
// Executes in O(1) time and allocates memory.
func NewLimiter(r Limit, b int) *Limiter {
	return &Limiter{
		limit:  r,
		burst:  b,
		tokens: float64(b),
	}
}

// Allow reports whether an event may happen now.
//
// Executes in O(1) time and is zero-allocation.
func (l *Limiter) Allow() bool {
	return l.AllowN(time.Now(), 1)
}

// AllowN reports whether n events may happen at time t.
//
// Executes in O(1) time and is zero-allocation.
func (l *Limiter) AllowN(t time.Time, n int) bool {
	return l.reserveN(t, n, 0).ok
}

// Reservation holds information about events that are permitted by a Limiter to happen after a delay.
//
// The zero value is empty and ready to use.
type Reservation struct {
	ok        bool
	lim       *Limiter
	tokens    int
	timeToAct time.Time
	// This is the Limit at reservation time, it can change later.
	limit Limit
}

// OK reports whether the limiter can provide the requested number of tokens within the maximum wait time.
//
// Executes in O(1) time and is zero-allocation.
func (r *Reservation) OK() bool {
	return r.ok
}

// Delay returns the wait duration required before the reservation can be acted upon.
//
// Executes in O(1) time and is zero-allocation.
func (r *Reservation) Delay() time.Duration {
	return r.DelayFrom(time.Now())
}

// InfDuration is the duration returned by Delay when a Reservation is not OK.
const InfDuration = time.Duration(math.MaxInt64)

// DelayFrom returns the duration for which the reservation holder must wait before taking the reserved action.
//
// Executes in O(1) time and is zero-allocation.
func (r *Reservation) DelayFrom(t time.Time) time.Duration {
	if !r.ok {
		return InfDuration
	}
	delay := r.timeToAct.Sub(t)
	if delay < 0 {
		return 0
	}
	return delay
}

// Cancel indicates that the reservation holder will not perform the reserved action.
//
// Executes in O(1) time and is zero-allocation.
func (r *Reservation) Cancel() {
	r.CancelAt(time.Now())
}

// CancelAt indicates that the reservation holder will not perform the reserved action at time t.
//
// Executes in O(1) time and is zero-allocation.
func (r *Reservation) CancelAt(t time.Time) {
	if !r.ok {
		return
	}

	r.lim.mu.Lock()
	defer r.lim.mu.Unlock()

	if r.lim.limit == Inf || r.tokens == 0 || r.timeToAct.Before(t) {
		return
	}

	// calculate tokens to restore
	// The duration between lim.lastEvent and r.timeToAct tells us how many tokens were reserved
	// after r was obtained. These tokens should not be restored.
	restoreTokens := float64(r.tokens) - r.limit.tokensFromDuration(r.lim.lastEvent.Sub(r.timeToAct))
	if restoreTokens <= 0 {
		return
	}
	// advance time to now
	tokens := r.lim.advance(t)
	// calculate new number of tokens
	tokens += restoreTokens
	if burst := float64(r.lim.burst); tokens > burst {
		tokens = burst
	}
	// update state
	r.lim.last = t
	r.lim.tokens = tokens
	if r.timeToAct.Equal(r.lim.lastEvent) {
		prevEvent := r.timeToAct.Add(r.limit.durationFromTokens(float64(-r.tokens)))
		if !prevEvent.Before(t) {
			r.lim.lastEvent = prevEvent
		}
	}
}

// Reserve returns a [Reservation] indicating how long to wait before an event can happen.
//
// Executes in O(1) time and allocates memory for the Reservation.
func (l *Limiter) Reserve() *Reservation {
	return l.ReserveN(time.Now(), 1)
}

// ReserveN returns a [Reservation] indicating how long to wait before n events can happen.
//
// Executes in O(1) time and allocates memory for the Reservation.
func (l *Limiter) ReserveN(t time.Time, n int) *Reservation {
	r := l.reserveN(t, n, InfDuration)
	return &r
}

// Wait blocks until the limiter permits an event to happen.
//
// Executes in amortized O(1) time excluding wait duration and is zero-allocation.
func (l *Limiter) Wait(ctx context.Context) (err error) {
	return l.WaitN(ctx, 1)
}

// WaitN blocks until the limiter permits n events to happen.
//
// Executes in amortized O(1) time excluding wait duration and is zero-allocation.
func (l *Limiter) WaitN(ctx context.Context, n int) (err error) {
	// The test code calls lim.wait with a fake timer generator.
	// This is the real timer generator.
	newTimer := func(d time.Duration) (<-chan time.Time, func() bool, func()) {
		timer := time.NewTimer(d)
		return timer.C, timer.Stop, func() {}
	}

	return l.wait(ctx, n, time.Now(), newTimer)
}

// wait is the internal implementation of WaitN.
func (l *Limiter) wait(
	ctx context.Context,
	n int,
	t time.Time,
	newTimer func(d time.Duration) (<-chan time.Time, func() bool, func()),
) error {
	l.mu.Lock()
	burst := l.burst
	limit := l.limit
	l.mu.Unlock()

	if n > burst && limit != Inf {
		return fmt.Errorf("rate: Wait(n=%d) exceeds limiter's burst %d", n, burst)
	}
	// Check if ctx is already cancelled
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	// Determine wait limit
	waitLimit := InfDuration
	if deadline, ok := ctx.Deadline(); ok {
		waitLimit = deadline.Sub(t)
	}
	// Reserve
	r := l.reserveN(t, n, waitLimit)
	if !r.ok {
		return fmt.Errorf("rate: Wait(n=%d) would exceed context deadline", n)
	}
	// Wait if necessary
	delay := r.DelayFrom(t)
	if delay == 0 {
		return nil
	}
	ch, stop, advance := newTimer(delay)
	defer stop()
	advance() // only has an effect when testing
	select {
	case <-ch:
		// We can proceed.
		return nil
	case <-ctx.Done():
		// Context was canceled before we could proceed.  Cancel the
		// reservation, which may permit other events to proceed sooner.
		r.Cancel()
		return ctx.Err()
	}
}

// SetLimit sets a new rate limit for the limiter.
//
// Executes in O(1) time and is zero-allocation.
func (l *Limiter) SetLimit(newLimit Limit) {
	l.SetLimitAt(time.Now(), newLimit)
}

// SetLimitAt sets a new rate limit for the limiter at a specific time.
//
// Executes in O(1) time and is zero-allocation.
func (l *Limiter) SetLimitAt(t time.Time, newLimit Limit) {
	l.mu.Lock()
	defer l.mu.Unlock()

	tokens := l.advance(t)

	l.last = t
	l.tokens = tokens
	l.limit = newLimit
}

// SetBurst sets a new burst size for the limiter.
//
// Executes in O(1) time and is zero-allocation.
func (l *Limiter) SetBurst(newBurst int) {
	l.SetBurstAt(time.Now(), newBurst)
}

// SetBurstAt sets a new burst size for the limiter at a specific time.
//
// Executes in O(1) time and is zero-allocation.
func (l *Limiter) SetBurstAt(t time.Time, newBurst int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	tokens := l.advance(t)

	l.last = t
	l.tokens = tokens
	l.burst = newBurst
}

// reserveN is a helper method for AllowN, ReserveN, and WaitN.
// maxFutureReserve specifies the maximum reservation wait duration allowed.
// reserveN returns Reservation, not *Reservation, to avoid allocation in AllowN and WaitN.
func (l *Limiter) reserveN(t time.Time, n int, maxFutureReserve time.Duration) Reservation {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.limit == Inf {
		return Reservation{
			ok:        true,
			lim:       l,
			tokens:    n,
			timeToAct: t,
		}
	}

	tokens := l.advance(t)

	// Calculate the remaining number of tokens resulting from the request.
	tokens -= float64(n)

	// Calculate the wait duration
	var waitDuration time.Duration
	if tokens < 0 {
		waitDuration = l.limit.durationFromTokens(-tokens)
	}

	// Decide result
	ok := n <= l.burst && waitDuration <= maxFutureReserve

	// Prepare reservation
	r := Reservation{
		ok:    ok,
		lim:   l,
		limit: l.limit,
	}
	if ok {
		r.tokens = n
		r.timeToAct = t.Add(waitDuration)

		// Update state
		l.last = t
		l.tokens = tokens
		l.lastEvent = r.timeToAct
	}

	return r
}

// advance calculates and returns an updated number of tokens for lim
// resulting from the passage of time.
// lim is not changed.
// advance requires that l.mu is held.
func (l *Limiter) advance(t time.Time) (newTokens float64) {
	last := l.last
	if t.Before(last) {
		last = t
	}

	// Calculate the new number of tokens, due to time that passed.
	elapsed := t.Sub(last)
	delta := l.limit.tokensFromDuration(elapsed)
	tokens := l.tokens + delta
	if burst := float64(l.burst); tokens > burst {
		tokens = burst
	}
	return tokens
}

// durationFromTokens is a unit conversion function from the number of tokens to the duration
// of time it takes to accumulate them at a rate of limit tokens per second.
func (l Limit) durationFromTokens(tokens float64) time.Duration {
	if l <= 0 {
		return InfDuration
	}

	duration := (tokens / float64(l)) * float64(time.Second)

	// Cap the duration to the maximum representable int64 value, to avoid overflow.
	if duration > float64(math.MaxInt64) {
		return InfDuration
	}

	return time.Duration(duration)
}

// tokensFromDuration is a unit conversion function from a time duration to the number of tokens
// which could be accumulated during that duration at a rate of limit tokens per second.
func (l Limit) tokensFromDuration(d time.Duration) float64 {
	if l <= 0 {
		return 0
	}
	return d.Seconds() * float64(l)
}
