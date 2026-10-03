// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package casing provides high-performance string case conversion algorithms
// (snake_case, camelCase, PascalCase, kebab-case, SCREAMING_SNAKE_CASE).
//
// # Compared to the standard library
//
// Stdlib counterpart: strings
//
// Rejected compromise: The strings package does not have specialized case converters for variable naming formats.
//
// Accepted cost: This package adds surface area outside the standard strings package.
//
// Allocations: bounded(1/op)
package casing
