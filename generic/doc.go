// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package generic provides a type-safe functional utility toolkit for Go.
//
// The package leverages Go generics to eliminate repetitive boilerplate code,
// manual type assertions, and unsafe reflection on hot paths. It provides slice and map
// operations, lazy sequences powered by Go iterators, thread-safe collections (like TTL caches),
// advanced concurrency controls (like ParallelMap and SingleFlight), and monadic wrappers.
// The [Optional] and [Result] types are the canonical monads for expressing partial states
// or batch operation outcomes. Note that this package is exclusively reserved for external consumers;
// internal foundation packages never import it, and low-level packages use comma-ok patterns instead.
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: type-safe functional utilities and concurrency primitives
//
// Rejected compromise: using type assertions (interface{}) and repetitive boilerplate for data manipulation
//
// Accepted cost: introducing custom wrapper types that diverge from standard Go idioms for specific architectural layers
//
// Allocations: amortized
package generic
