// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package svcb implements Service Binding (SVCB) and HTTPS Resource Records strictly
// conforming to RFC 9460.
//
// It provides zero-allocation wire-format parsing and serialization for SVCB (RR 64)
// and HTTPS (RR 65) resource records, supporting ALPN negotiation, IP hints, Encrypted
// ClientHello (ECH), and endpoint priorities.
//
// # Compared to the standard library
//
// Stdlib counterpart: net (unexported / missing RFC 9460 support)
//
// Rejected compromise: relying on external DNS dependencies or missing SVCB/HTTPS
// discovery for HTTP/3 and ECH
//
// Accepted cost: manual SvcParam decoding and validation
//
// Allocations: bounded(1/op)
package svcb
