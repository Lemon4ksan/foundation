// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package gomock provides an interface mocking engine.
//
// Designed to interoperate with generated mock implementations, this package provides
// the core controllers and matchers required to verify complex interactions between
// components. It acts as the underlying execution engine for interface mocks.
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: automated interface mocking and behavior verification
//
// Rejected compromise: hand-rolling mock implementations and manual state verification for every interface
//
// Accepted cost: utilizing generated code and complex matching logic during tests
//
// Allocations: unconstrained
package gomock
