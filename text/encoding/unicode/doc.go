// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package unicode provides Unicode encoding support.
//
// This is a port of golang.org/x/text/encoding/unicode.
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: legacy character set encodings
//
// Rejected compromise: The standard library only natively supports UTF-8.
//
// Accepted cost: Additional code size for mapping tables.
//
// Allocations: unconstrained
package unicode
