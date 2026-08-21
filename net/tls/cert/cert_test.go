// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cert_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lemon4ksan/foundation/net/tls/cert"
)

func generateTestCert(t *testing.T) *x509.Certificate {
	t.Helper()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "example.com",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	require.NoError(t, err)

	parsed, err := x509.ParseCertificate(derBytes)
	require.NoError(t, err)

	return parsed
}

func TestSPKIFingerprint(t *testing.T) {
	t.Parallel()

	c := generateTestCert(t)

	fp, err := cert.SPKIFingerprint(c)
	require.NoError(t, err)
	assert.NotEqual(t, [32]byte{}, fp)

	b64, err := cert.SPKIFingerprintBase64(c)
	require.NoError(t, err)
	assert.NotEmpty(t, b64)

	err = cert.ValidateChainSPKI([]*x509.Certificate{c}, [][32]byte{fp})
	assert.NoError(t, err)

	err = cert.ValidateChainSPKI([]*x509.Certificate{c}, [][32]byte{[32]byte{0xFF}})
	assert.ErrorIs(t, err, cert.ErrNoMatchingPin)
}

func TestCompressionAlgorithm_Parsing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected cert.CompressionAlgorithm
		str      string
	}{
		{"zlib", cert.CertCompressionZlib, "zlib"},
		{"brotli", cert.CertCompressionBrotli, "brotli"},
		{"br", cert.CertCompressionBrotli, "brotli"},
		{"zstd", cert.CertCompressionZstd, "zstd"},
	}

	for _, tt := range tests {
		algo, err := cert.ParseCompressionAlgorithm(tt.input)
		require.NoError(t, err)
		assert.Equal(t, tt.expected, algo)
		assert.Equal(t, tt.str, algo.String())
		assert.True(t, algo.IsValid())
	}

	_, err := cert.ParseCompressionAlgorithm("invalid")
	assert.ErrorIs(t, err, cert.ErrUnknownCompressionAlgo)
}
