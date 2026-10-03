// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package diff implements semantic schema and API contract drift analysis.
//
// It compares local and remote API definitions to detect breaking changes,
// non-breaking modifications, and ghost endpoints (endpoints missing in one contract).
// The package produces a [DiffReport] summarizing the detected [DriftItem] changes.
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: API schema drift analysis
//
// Rejected compromise: The standard library does not provide API schema parsing or comparison.
//
// Accepted cost: Consumers must parse their API contracts into a format this package understands.
//
// Allocations: unconstrained
package diff
