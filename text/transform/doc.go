// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package transform provides reader and writer wrappers that transform the
// bytes passing through as well as various transformations.
//
// This is a port of golang.org/x/text/transform. Example transformations
// provided by other packages include normalization and conversion between character sets.
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: stream transformation pipeline
//
// Rejected compromise: The standard library io package operates on raw bytes without an extensible pipeline.
//
// Accepted cost: Additional complexity for stream transformations.
//
// Allocations: zero
package transform
