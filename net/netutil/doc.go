// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package netutil provides network address normalization and host sanitization utilities.
//
// The package offers functions to safely normalize host strings and split network addresses.
// It strips IPv6 Zone IDs per RFC 6874 and converts Internationalized Domain Names (IDN) to
// ASCII Punycode, making the output suitable for network resolution, HTTP headers, and TLS SNI.
//
// # Compared to the standard library
//
// Stdlib counterpart: net
//
// Rejected compromise: net.SplitHostPort relies on generic string manipulation and does not sanitize IDNs or strip Zone IDs.
//
// Accepted cost: IDN conversion and string allocations happen on every call.
//
// Allocations: unconstrained
package netutil
