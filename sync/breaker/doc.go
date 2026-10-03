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
// # Compared to the standard library
//
// Stdlib counterpart: none
//
// Rejected compromise: static retry loops without backpressure and coordination.
//
// Accepted cost: Mutex locking on every request to track sliding window metrics.
//
// Allocations: amortized
package breaker
