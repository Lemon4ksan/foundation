// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package pathkit provides an immutable, unified path abstraction.
//
// It bridges remote network URLs, cloud storage URIs, local OS filepaths, and RFC 8089
// file:// URIs with cross-platform consistency. It supports smart joining without
// duplicate separators, standardizes separator slashes, provides zero-allocation Go
// 1.27 push iterators (Segments, Ancestors), and caller-supplied destination buffer
// methods (Append*).
//
// # Compared to the standard library
//
// Stdlib counterpart: path, path/filepath, net/url
//
// Rejected compromise: The stdlib splits path manipulation across multiple packages
// based on the path type and incurs allocations on joining, cleaning, and iteration.
//
// Accepted cost: Path is a custom type that must be converted to strings for standard
// library calls.
//
// Allocations: bounded(1/op)
package pathkit
