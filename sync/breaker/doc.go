// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package breaker implements a thread-safe, generic circuit breaker with sliding
// window metrics and automatic state transition coordination.
//
// The circuit breaker pattern prevents cascading failures in distributed systems
// by failing fast when downstream services are unhealthy, giving them time
// to recover before accepting new requests.
//
// # State Machine
//
// A [CircuitBreaker] operates in one of three states:
//   - StateClosed: Requests flow normally. Failures and successes are recorded
//     over a sliding time window. If the failure ratio exceeds the configured
//     threshold, the breaker transitions to StateOpen.
//   - StateOpen: Requests are rejected immediately without execution, returning
//     [ErrCircuitOpen]. After a configured Cooldown duration, the breaker
//     automatically transitions to StateHalfOpen.
//   - StateHalfOpen: Allows a single concurrent "trial" request to execute and probe
//     downstream health. If this trial request succeeds, the breaker transitions
//     back to StateClosed. If it fails, the breaker immediately returns to StateOpen
//     with a fresh Cooldown timer.
//
// # Concurrency and Thread Safety
//
// All operations are synchronized using a [sync.Mutex] guarding internal states.
// While a wrapped task is executing, the mutex is unlocked to prevent blocking
// other concurrent goroutines. In StateHalfOpen, a dedicated concurrency barrier
// (halfOpenExecuting) ensures that strictly one concurrent probe request is allowed
// to execute at any given time.
package breaker
