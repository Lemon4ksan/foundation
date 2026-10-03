// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package backoff provides mathematical backoff calculators with jitter distributions.
//
// The package prevents the thundering herd problem in high-throughput resilience pipelines
// by calculating backoff delays when operations fail. It includes stateful and stateless
// generators using constant, linear, and exponential algorithms, as well as multiple jitter modes
// such as equal, full, and decorrelated. These generators can be securely embedded across
// retry loops and circuit breakers to provide stable delay progression.
//
// # Compared to the standard library
//
// Stdlib counterpart: none
//
// Rejected compromise: relying on simple time.Sleep with rand.Intn which is not
// concurrently safe nor uniformly distributed without bias.
//
// Accepted cost: the decorrelated jitter generator must maintain state and requires
// external synchronization if shared across multiple concurrent operations.
//
// Allocations: zero
package backoff
