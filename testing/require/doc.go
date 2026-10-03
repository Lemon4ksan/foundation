// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package require provides immediate-terminating testing assertions.
//
// The package mirrors the API of [github.com/lemon4ksan/foundation/testing/assert] but calls
// t.FailNow() upon failure, immediately halting the current test's execution.
// This is essential for validating preconditions (such as checking if an error is nil)
// where subsequent test logic would panic or behave unpredictably if the condition is not met.
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: immediate-terminating test assertions
//
// Rejected compromise: relying purely on manual if-statements and t.Fatalf for value comparisons
//
// Accepted cost: adopting a custom API surface for assertions
//
// Allocations: unconstrained
package require
