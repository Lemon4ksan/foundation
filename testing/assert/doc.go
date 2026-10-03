// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package assert provides non-terminating testing assertions with detailed failure diffs.
//
// The package offers a comprehensive set of testing primitives (e.g., [Equal], [True], [Nil])
// that report failures without halting the current test execution. This allows a single
// test case to collect multiple assertion failures, providing a complete picture of the
// test state. For assertions that must halt execution on failure, use the sibling
// [github.com/lemon4ksan/foundation/testing/require] package.
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: rich, readable test assertions
//
// Rejected compromise: relying purely on manual if-statements and boilerplate for value comparisons
//
// Accepted cost: adopting a custom API surface for assertions
//
// Allocations: unconstrained
package assert
