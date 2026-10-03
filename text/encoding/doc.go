// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package encoding defines an interface for character encodings, such as Shift JIS and Windows 1252.
//
// This is a port of golang.org/x/text/encoding. It provides character set encodings that can be
// transformed to and from UTF-8. Implementations for specific encodings are provided in sub-packages.
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: legacy character set encodings
//
// Rejected compromise: The standard library only natively supports UTF-8.
//
// Accepted cost: Additional code size and complexity for legacy encodings.
//
// Allocations: unconstrained
package encoding
