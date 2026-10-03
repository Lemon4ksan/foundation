// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package mock provides primitives for fluent method mocking and call verification.
//
// The package allows test authors to create mock objects that track method invocations,
// specify expected return values, and verify call counts. By embedding [Mock] in a custom type,
// you can intercept method calls and return pre-configured responses, enabling robust unit testing
// of components without their real dependencies.
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: method mocking and expectation tracking
//
// Rejected compromise: writing manual mock implementations with ad-hoc tracking logic for every interface
//
// Accepted cost: reflection overhead during test execution to match method signatures and track calls
//
// Allocations: unconstrained
package mock
