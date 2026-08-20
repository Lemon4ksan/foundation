// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package breaker

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrCircuitOpen is returned immediately when [CircuitBreaker.Do] is invoked
// while the circuit breaker is in the StateOpen or blocked StateHalfOpen state.
var ErrCircuitOpen = errors.New("circuit breaker is open")

// State represents the operational state of a [CircuitBreaker].
type State int

const (
	// StateClosed allows requests to flow normally.
	StateClosed State = iota
	// StateOpen fails requests immediately with [ErrCircuitOpen].
	StateOpen
	// StateHalfOpen allows a single trial request to probe downstream health.
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "Closed"
	case StateOpen:
		return "Open"
	case StateHalfOpen:
		return "Half-Open"
	default:
		return "Unknown"
	}
}

// record represents a single request outcome tracked in the sliding window.
type record struct {
	time     time.Time
	isFailed bool
}

// Config defines the operational boundaries and thresholds for a [CircuitBreaker].
type Config struct {
	// FailureThreshold is the ratio of failures (0.0 to 1.0) over the sliding
	// window that triggers a transition from [StateClosed] to [StateOpen].
	// Defaults to 0.5 (50% failure rate).
	FailureThreshold float64

	// Cooldown is the duration spent in [StateOpen] before transitioning
	// to [StateHalfOpen] to attempt recovery. Defaults to 5 seconds.
	Cooldown time.Duration

	// MinRequests is the minimum number of tracked requests in a Window required
	// before the failure threshold ratio is actively checked. Defaults to 5.
	MinRequests int

	// Window is the sliding time duration over which failures are tracked.
	// Defaults to 10 seconds.
	Window time.Duration

	// OnStateChange is an optional callback executed asynchronously on state transitions.
	OnStateChange func(from, to State)
}

// ResolveDefaults validates the configuration and applies safe default fallback boundaries.
func (c *Config) ResolveDefaults() {
	if c.FailureThreshold <= 0 || c.FailureThreshold > 1.0 {
		c.FailureThreshold = 0.5
	}

	if c.Cooldown <= 0 {
		c.Cooldown = 5 * time.Second
	}

	if c.MinRequests <= 0 {
		c.MinRequests = 5
	}

	if c.Window <= 0 {
		c.Window = 10 * time.Second
	}
}

// CircuitBreaker wraps synchronous operations with a fail-fast state machine
// to protect upstream services from cascading failures.
//
// A CircuitBreaker is safe for concurrent use by multiple goroutines.
type CircuitBreaker[T any] struct {
	mu       sync.Mutex
	cfg      Config
	state    State
	openTime time.Time
	records  []record

	halfOpenExecuting bool
}

// New creates and returns a new [CircuitBreaker] instance with the given [Config].
func New[T any](cfg Config) *CircuitBreaker[T] {
	cfg.ResolveDefaults()

	return &CircuitBreaker[T]{
		cfg:   cfg,
		state: StateClosed,
	}
}

// State returns the current operational [State] of the CircuitBreaker.
//
// It is safe for concurrent use and dynamically evaluates cooldown expirations.
func (cb *CircuitBreaker[T]) State() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.checkState(time.Now())

	return cb.state
}

// Do wraps the execution of the provided function fn, monitoring its outcome.
//
// If the breaker is in [StateOpen], or if the breaker is in [StateHalfOpen] and
// another trial request is already executing, Do returns [ErrCircuitOpen]
// immediately without running fn.
//
// If the execution of fn returns an error, it is recorded as a failure.
// Otherwise, it is recorded as a success.
func (cb *CircuitBreaker[T]) Do(ctx context.Context, fn func(ctx context.Context) (T, error)) (T, error) {
	cb.mu.Lock()

	now := time.Now()
	cb.checkState(now)

	if cb.state == StateOpen {
		cb.mu.Unlock()

		var zero T

		return zero, ErrCircuitOpen
	}

	// Concurrency barrier for StateHalfOpen: only allow a single trial request.
	if cb.state == StateHalfOpen {
		if cb.halfOpenExecuting {
			cb.mu.Unlock()

			var zero T

			return zero, ErrCircuitOpen
		}

		cb.halfOpenExecuting = true
	}

	isHalfOpen := cb.state == StateHalfOpen
	cb.mu.Unlock()

	// Execute the payload outside the lock to prevent thread blocking.
	val, err := fn(ctx)

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if isHalfOpen {
		cb.halfOpenExecuting = false

		// Resolve StateHalfOpen based on the single trial request outcome.
		if err != nil {
			cb.transitionTo(StateOpen, time.Now())
		} else {
			cb.transitionTo(StateClosed, time.Time{})
		}

		return val, err
	}

	// If the state transitioned away from [StateClosed] while we were executing,
	// do not record this metric to prevent dirtying the new state's window.
	if cb.state != StateClosed {
		return val, err
	}

	cb.records = append(cb.records, record{time: time.Now(), isFailed: err != nil})
	cb.pruneRecords(time.Now())

	// Evaluate failure threshold if we have accumulated enough requests.
	if len(cb.records) >= cb.cfg.MinRequests {
		failures := 0
		for _, r := range cb.records {
			if r.isFailed {
				failures++
			}
		}

		ratio := float64(failures) / float64(len(cb.records))
		if ratio >= cb.cfg.FailureThreshold {
			cb.transitionTo(StateOpen, time.Now())
		}
	}

	return val, err
}

// checkState evaluates if the cooldown timer has expired and transitions
// the state from [StateOpen] to [StateHalfOpen].
//
// The caller must hold cb.mu.
func (cb *CircuitBreaker[T]) checkState(now time.Time) {
	if cb.state == StateOpen && now.Sub(cb.openTime) > cb.cfg.Cooldown {
		cb.transitionTo(StateHalfOpen, time.Time{})
	}
}

// transitionTo executes the state transition, resets metric windows,
// and triggers transition callbacks.
//
// The caller must hold cb.mu.
func (cb *CircuitBreaker[T]) transitionTo(target State, openTime time.Time) {
	if cb.state == target {
		return
	}

	from := cb.state
	cb.state = target
	cb.openTime = openTime

	// Clear sliding metrics on boundary state transitions.
	if target == StateClosed || target == StateOpen {
		cb.records = nil
	}

	// Trigger callbacks in an isolated background goroutine to prevent
	// potential user-callback deadlocks on cb.mu.
	if cb.cfg.OnStateChange != nil {
		go cb.cfg.OnStateChange(from, target)
	}
}

// pruneRecords discards any metrics that have slid out of the configured window.
//
// The caller must hold cb.mu.
func (cb *CircuitBreaker[T]) pruneRecords(now time.Time) {
	cutoff := now.Add(-cb.cfg.Window)
	i := 0

	for i < len(cb.records) && cb.records[i].time.Before(cutoff) {
		i++
	}

	if i > 0 {
		cb.records = cb.records[i:]
	}
}
