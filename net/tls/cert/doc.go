// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package cert provides TLS certificate utilities for SPKI fingerprinting and compression.
//
// This package implements Subject Public Key Info (SPKI) SHA-256 fingerprinting (RFC 7469)
// for certificate pinning, and defines constants for TLS Certificate Compression (RFC 8879).
// It verifies public keys directly from ASN.1 DER data to skip parsing full X.509 chains
// when possible.
//
// # Compared to the standard library
//
// Stdlib counterpart: crypto/tls and crypto/x509
//
// Rejected compromise: crypto/x509 requires parsing the entire certificate to extract the public key.
//
// Accepted cost: callers must manually manage expected SPKI pin sets.
//
// Allocations: bounded(1/op)
package cert
