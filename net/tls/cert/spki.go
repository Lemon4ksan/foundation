// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package cert provides TLS certificate utilities including Subject Public Key Info (SPKI)
// SHA-256 fingerprinting (RFC 7469) and TLS Certificate Compression (RFC 8879).
package cert

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"errors"
)

// Standard errors for SPKI verification.
var (
	ErrNoCertificates = errors.New("cert: empty certificate chain")
	ErrNoMatchingPin  = errors.New("cert: no certificate in chain matches pinned SPKI")
	ErrInvalidSPKI    = errors.New("cert: invalid or empty SubjectPublicKeyInfo")
)

// SPKIFingerprint calculates the 32-byte SHA-256 digest of a certificate's Subject Public Key Info (RFC 7469 §2.4).
func SPKIFingerprint(cert *x509.Certificate) ([32]byte, error) {
	if cert == nil || len(cert.RawSubjectPublicKeyInfo) == 0 {
		return [32]byte{}, ErrInvalidSPKI
	}

	return sha256.Sum256(cert.RawSubjectPublicKeyInfo), nil
}

// SPKIFingerprintFromDER calculates the 32-byte SHA-256 digest of raw DER-encoded Subject Public Key Info.
func SPKIFingerprintFromDER(rawSPKI []byte) [32]byte {
	return sha256.Sum256(rawSPKI)
}

// SPKIFingerprintBase64 returns the standard base64-encoded SHA-256 SPKI fingerprint (RFC 7469 §2.1.1).
func SPKIFingerprintBase64(cert *x509.Certificate) (string, error) {
	fp, err := SPKIFingerprint(cert)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(fp[:]), nil
}

// ValidateChainSPKI validates that at least one certificate in the provided X.509 chain matches
// one of the expected 32-byte SPKI SHA-256 hashes (RFC 7469 §2.6).
func ValidateChainSPKI(chain []*x509.Certificate, expectedPins [][32]byte) error {
	if len(chain) == 0 {
		return ErrNoCertificates
	}

	if len(expectedPins) == 0 {
		return nil
	}

	for _, cert := range chain {
		if cert == nil || len(cert.RawSubjectPublicKeyInfo) == 0 {
			continue
		}

		spkiHash := sha256.Sum256(cert.RawSubjectPublicKeyInfo)
		for _, pin := range expectedPins {
			if spkiHash == pin {
				return nil
			}
		}
	}

	return ErrNoMatchingPin
}
