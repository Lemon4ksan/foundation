// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package rate implements a token-bucket rate limiter and execution sampling filters.
//
// The package provides two primary rate-shaping primitives. The [Limiter] models traffic
// flow using a token bucket characterized by an emission rate (tokens replenished per second)
// and a maximum burst capacity. The [Sometimes] sampler controls execution frequency across
// hotpath loops or telemetry emission using interval or event-count decimation.
//
// The token-bucket algorithm naturally accommodates legitimate traffic bursts while
// guaranteeing an upper bound on sustained frequency. Both primitives are safe for
// concurrent invocation across multiple goroutines. Fast paths, such as checking
// [Limiter.Allow] or waiting on an available token via [Limiter.Wait] without blocking,
// are allocation-free. If [Limiter.Wait] is forced to block for a future token, it
// allocates an internal timer.
//
// # Compared to the standard library
//
// Stdlib counterpart: golang.org/x/time/rate
//
// Rejected compromise: none - this is a direct port of golang.org/x/time/rate to avoid adding a third-party module dependency for a single package.
//
// Accepted cost: the maintenance burden of keeping the fork updated with upstream bug fixes.
//
// Allocations: zero
package rate
