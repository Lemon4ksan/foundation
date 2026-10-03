// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package testing provides a zero-dependency toolkit for unit testing and mocking.
//
// It replaces bulky external testing dependencies with a lightweight, high-performance
// alternative that integrates seamlessly with the standard library. The toolkit comprises
// subpackages for non-terminating assertions ([github.com/lemon4ksan/foundation/testing/assert]),
// immediate-terminating requirements ([github.com/lemon4ksan/foundation/testing/require]),
// and interface mocking primitives ([github.com/lemon4ksan/foundation/testing/mock]).
//
// # Compared to the standard library
//
// Stdlib counterpart: testing
//
// Rejected compromise: relying purely on manual if-statements and boilerplate for value comparisons
//
// Accepted cost: adopting a custom API surface for assertions and mocks
//
// Allocations: unconstrained
package testing
