// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package rate implements a high-precision token-bucket rate limiter and execution sampling filters.
//
// # Architecture
//
// The package provides two primary rate-shaping primitives:
//   - Token-Bucket Rate Limiter: [Limiter] models traffic flow using a token bucket characterized by
//     an emission rate [Limit] (tokens replenished per second) and maximum burst capacity ([Limiter.Burst]).
//     It supports three consumption paradigms:
//     1. Non-blocking checks: [Limiter.Allow] and [Limiter.AllowN] report immediate token availability.
//     2. Blocking context-aware waiting: [Limiter.Wait] and [Limiter.WaitN] suspend the calling goroutine
//     until tokens are replenished or the context deadline expires.
//     3. Reservation: [Limiter.Reserve] and [Limiter.ReserveN] return a [Reservation] enabling callers to delay
//     actions or inspect future replenishment delays.
//   - Execution Sampler: [Sometimes] controls execution frequency across hotpath loops, logging statements,
//     and telemetry emission using interval or event-count decimation.
//
// # Design Rationale
//
// Modern microservices require strict throughput bounds to guard against resource exhaustion, downstream
// cascade failures, and Denial of Service. Unlike naive sleep-based limiters, the token-bucket algorithm
// naturally accommodates legitimate traffic bursts while guaranteeing an upper bound on sustained frequency.
//
// # Concurrency Guarantees
//
//   - [Limiter] is fully thread-safe. All methods ([Limiter.Allow], [Limiter.Wait], [Limiter.Reserve],
//     [Limiter.SetLimit], [Limiter.SetBurst]) are synchronized internally via a [sync.Mutex].
//   - [Sometimes] instances utilize atomic operations or internal synchronization, making them safe
//     for concurrent invocation across multiple worker goroutines.
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
//		"github.com/lemon4ksan/foundation/async/rate"
//	)
//
//	func main() {
//		// Limit to 10 operations per second with burst capability of 3
//		limiter := rate.NewLimiter(rate.Limit(10), 3)
//		ctx := context.Background()
//
//		for i := range 5 {
//			if err := limiter.Wait(ctx); err != nil {
//				panic(err)
//			}
//			fmt.Printf("Processed request %d at %s\n", i, time.Now().Format("15:04:05.000"))
//		}
//	}
package rate
