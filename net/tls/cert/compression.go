// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cert

import (
	"errors"
	"fmt"
	"strings"
)

// ErrUnknownCompressionAlgo indicates an unrecognized TLS certificate compression algorithm name.
var ErrUnknownCompressionAlgo = errors.New("cert: unknown certificate compression algorithm")

// RFC 8879 IANA Registry Constants for TLS Certificate Compression.
const (
	// ExtensionCompressCertificate is the TLS 1.3 extension type code point (27 / 0x001B) defined in RFC 8879 §7.1.
	ExtensionCompressCertificate uint16 = 27

	// HandshakeCompressedCertificate is the TLS 1.3 handshake message type (25 / 0x19) defined in RFC 8879 §7.2.
	HandshakeCompressedCertificate uint8 = 25
)

// CompressionAlgorithm specifies a certificate compression algorithm defined in RFC 8879 §7.3.
type CompressionAlgorithm uint16

const (
	// CertCompressionReserved is reserved in RFC 8879 §7.3 (value 0).
	CertCompressionReserved CompressionAlgorithm = 0
	// CertCompressionZlib specifies the zlib certificate compression algorithm (RFC 8879 §7.3 & RFC 1950).
	CertCompressionZlib CompressionAlgorithm = 1
	// CompressionZlib is an alias for [CertCompressionZlib].
	CompressionZlib = CertCompressionZlib

	// CertCompressionBrotli specifies the Brotli certificate compression algorithm (RFC 8879 §7.3 & RFC 7932).
	CertCompressionBrotli CompressionAlgorithm = 2
	// CompressionBrotli is an alias for [CertCompressionBrotli].
	CompressionBrotli = CertCompressionBrotli

	// CertCompressionZstd specifies the Zstandard certificate compression algorithm (RFC 8879 §7.3 & RFC 8478).
	CertCompressionZstd CompressionAlgorithm = 3
	// CompressionZstd is an alias for [CertCompressionZstd].
	CompressionZstd = CertCompressionZstd
)

// String returns the standard name of the certificate compression algorithm (RFC 8879 §7.3).
func (a CompressionAlgorithm) String() string {
	switch a {
	case CertCompressionZlib:
		return "zlib"
	case CertCompressionBrotli:
		return "brotli"
	case CertCompressionZstd:
		return "zstd"
	default:
		return fmt.Sprintf("unknown(%d)", uint16(a))
	}
}

// IsValid reports whether the compression algorithm is a standard RFC 8879 algorithm.
func (a CompressionAlgorithm) IsValid() bool {
	return a == CertCompressionZlib || a == CertCompressionBrotli || a == CertCompressionZstd
}

// ParseCompressionAlgorithm parses a string identifier ("zlib", "brotli", "zstd") into [CompressionAlgorithm].
func ParseCompressionAlgorithm(name string) (CompressionAlgorithm, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "zlib":
		return CertCompressionZlib, nil
	case "brotli", "br":
		return CertCompressionBrotli, nil
	case "zstd":
		return CertCompressionZstd, nil
	default:
		return 0, fmt.Errorf("%w: %q (RFC 8879)", ErrUnknownCompressionAlgo, name)
	}
}
