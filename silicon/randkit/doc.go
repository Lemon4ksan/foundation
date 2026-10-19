// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package randkit provides fast pseudo-random number generators, time-ordered UUIDv7
// builders, and CSPRNG cryptographic token helpers.
//
// Standard pseudo-random number generators using global mutexes bottleneck under
// multi-core concurrency, and standard UUID generation libraries produce multiple heap
// allocations per identifier. Package randkit combines lock-free math/rand/v2 and
// per-goroutine Wyrand ([RNG]) primitives with zero-allocation [AppendUUIDv7] formatting
// and CSPRNG token generators ([SecureBytes], [TokenHex], [TokenBase64], [TokenURLSafe]).
//
// # Compared to the standard library
//
// Stdlib counterpart: math/rand, crypto/rand
//
// Rejected compromise: Global mutexes that bottleneck under concurrency and multiple heap allocations per generated identifier.
//
// Accepted cost: [RNG] is not safe for concurrent use without external synchronization and is not cryptographically secure (use [SecureBytes] or Token* helpers for cryptographic entropy).
//
// Allocations: amortized
package randkit
