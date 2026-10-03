// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package limiter provides dynamic concurrency rate limiters and key-based limiters.
//
// The package prevents overload in distributed systems by dynamically adjusting the allowed
// concurrent request limit based on observed response times using a Vegas-style congestion
// control algorithm, or by managing rate limiters per client identifier with automatic
// cleanup for inactive users to prevent memory exhaustion.
//
// # Compared to the standard library
//
// Stdlib counterpart: golang.org/x/time/rate
//
// Rejected compromise: static limits that either underutilize bandwidth during healthy periods
// or overwhelm degraded downstream services, as well as leaking map entries when rate limiting
// by connection IP address.
//
// Accepted cost: Vegas congestion control requires active measurement of downstream latency
// on every request to accurately calculate the exponential moving average, and keyed limiters
// allocate dynamically when unseen keys are processed.
//
// Allocations: amortized
package limiter
