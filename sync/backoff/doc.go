// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package backoff provides zero-allocation mathematical backoff calculators with full, equal,
// and decorrelated jitter distributions for high-throughput resilience pipelines.
//
// # Architecture
//
// The package provides stateful and stateless backoff strategies implementing the [Strategy] interface:
//   - [ExponentialBackoff]: Multiplicative backoff growing by factor (default 2.0) capped at Max.
//   - [LinearBackoff]: Additive backoff growing by a constant step on each retry attempt.
//   - [ConstantBackoff]: Fixed delay generator for static rate-limiting or polling intervals.
//
// # Jitter Algorithms
//
// To prevent the "thundering herd" problem where multiple competing clients retry failed network
// operations simultaneously in synchronized waves, the package implements four jitter algorithms:
//   - [JitterNone]: Deterministic delay without randomization.
//   - [JitterFull]: Uniformly random delay in the interval [0, baseDelay] (AWS standard recommendation).
//   - [JitterEqual]: Combines deterministic base delay with uniform jitter in [baseDelay/2, baseDelay].
//   - [JitterDecorrelated]: Stateful jitter dynamically updating: next = min(max, rand(initial, prev * 3)).
//
// # Design Rationale
//
// In high-concurrency systems, retry loops can overwhelm struggling dependencies if retry intervals
// synchronize. [backoff] guarantees:
//   - Strictly zero heap allocations during delay calculation.
//   - Efficient pseudo-random generation using [randkit.Jitter].
//   - Clean interface abstraction for dependency injection across retry loops and circuit breakers.
//
// # Concurrency Guarantees
//
// [LinearBackoff] and [ConstantBackoff] are stateless and safe for concurrent invocation across goroutines.
// [ExponentialBackoff] with [JitterNone], [JitterFull], or [JitterEqual] is stateless and safe for concurrent use.
// [ExponentialBackoff] configured with [JitterDecorrelated] maintains state ([lastSleep]) and must either be
// dedicated to a single retry loop or protected by external synchronization.
//
// # Example
//
//	package main
//
//	import (
//		"context"
//		"fmt"
//		"time"
//
//		"github.com/lemon4ksan/foundation/sync/backoff"
//	)
//
//	func main() {
//		bo := backoff.NewExponential(100*time.Millisecond, 5*time.Second, 2.0).WithFullJitter()
//		ctx := context.Background()
//
//		for attempt := 1; attempt <= 5; attempt++ {
//			delay := bo.NextDelay(attempt)
//			fmt.Printf("Attempt %d delay: %v\n", attempt, delay)
//
//			select {
//			case <-time.After(delay):
//			case <-ctx.Done():
//				return
//			}
//		}
//	}
package backoff
