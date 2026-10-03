// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package values provides lenient scalar primitives for JSON serialization.
//
// Modern distributed systems frequently encounter inconsistent JSON payload schemas where numeric,
// boolean, or temporal fields arrive alternatively as quoted strings (e.g., "123", "true", "null")
// or native JSON literals. This package defines specialized scalar wrappers that implement
// standard JSON unmarshaling to seamlessly coerce these types into predictable Go primitives.
// These wrappers eliminate manual string-to-number transformation boilerplate in webhook handlers
// and legacy API clients.
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: lenient JSON unmarshaling for ambiguous scalar types
//
// Rejected compromise: strict schema adherence that causes unmarshaling to fail on perfectly readable strings like "123"
//
// Accepted cost: manual casting from wrapper types (e.g., uint64(val)) to raw numeric types after parsing
//
// Allocations: amortized
package values
