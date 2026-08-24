// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package gomock provides drop-in GoMock compatibility by re-exporting
// the mock package under the gomock package name.
package gomock

import (
	"github.com/lemon4ksan/foundation/testkit/mock"
)

// Re-exported types and functions for drop-in GoMock compatibility.
type (
	// Controller is the mock controller.
	Controller = mock.Controller
	// Call represents an expected mock method invocation.
	Call = mock.Call
	// Matcher represents a call argument matcher.
	Matcher = mock.Matcher
	// TestReporter is the interface for error reporting in tests.
	TestReporter = mock.TestReporter
	// TestHelper is the interface for helper annotation.
	TestHelper = mock.TestHelper
)

var (
	// NewController creates a new mock controller.
	NewController = mock.NewController
	// InOrder declares that the given calls must be matched in order.
	InOrder = mock.InOrder
	// Any returns a matcher matching anything.
	Any = mock.Any
	// Eq returns a matcher matching by deep equality.
	Eq = mock.Eq
	// Nil returns a matcher matching nil pointers, slices, maps, or interfaces.
	Nil = mock.Nil
	// Not inverts the inner matcher.
	Not = mock.Not
	// AssignableToTypeOf matches if arg is assignable to the target type.
	AssignableToTypeOf = mock.AssignableToTypeOf
	// Len matches if arg has length n.
	Len = mock.Len
)
