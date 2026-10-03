// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package randkit provides lock-free fast pseudo-random generators and zero-allocation UUID v4/v7 builders.
//
// Standard pseudo-random number generators utilize global mutexes that bottleneck under
// multi-core concurrency, and standard UUID generation libraries produce heap allocations
// on every identifier generated. Lock-free, thread-local algorithms provide identifier
// generation with zero memory allocation.
//
// # Compared to the standard library
//
// Stdlib counterpart: math/rand
//
// Rejected compromise: Global mutexes that bottleneck under concurrency and heap allocations per generated identifier.
//
// Accepted cost: Pseudo-random sequences are not cryptographically secure, and manual buffer management is required for UUID generation.
//
// Allocations: zero
package randkit
