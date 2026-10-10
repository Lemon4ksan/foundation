// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sign

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
)

const (
	// PublicKeySize is the length in bytes of an Ed25519 public key (32 bytes).
	PublicKeySize = ed25519.PublicKeySize

	// PrivateKeySize is the length in bytes of an Ed25519 private key (64 bytes).
	PrivateKeySize = ed25519.PrivateKeySize

	// SignatureSize is the length in bytes of an Ed25519 signature (64 bytes).
	SignatureSize = ed25519.SignatureSize

	// SeedSize is the length in bytes of an Ed25519 private key seed (32 bytes).
	SeedSize = ed25519.SeedSize
)

var (
	// ErrInvalidKeyLength indicates that the key size does not match expected Ed25519 lengths.
	ErrInvalidKeyLength = errors.New("sign: invalid key length")

	// ErrInvalidSignatureLength indicates that the signature size is not 64 bytes.
	ErrInvalidSignatureLength = errors.New("sign: invalid signature length")

	// ErrInvalidPEM indicates that the input could not be decoded as a valid PEM block.
	ErrInvalidPEM = errors.New("sign: invalid PEM block")
)

// PublicKey is an Ed25519 public key (32 bytes).
type PublicKey = ed25519.PublicKey

// PrivateKey is an Ed25519 private key (64 bytes).
type PrivateKey = ed25519.PrivateKey

// GenerateKey generates a new Ed25519 key pair using cryptographic entropy.
func GenerateKey() (PublicKey, PrivateKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("sign: generate key: %w", err)
	}
	return pub, priv, nil
}

// NewKeyFromSeed derives an Ed25519 private key from a 32-byte seed.
func NewKeyFromSeed(seed []byte) (PublicKey, PrivateKey, error) {
	if len(seed) != SeedSize {
		return nil, nil, fmt.Errorf("%w: seed must be %d bytes, got %d", ErrInvalidKeyLength, SeedSize, len(seed))
	}
	priv := ed25519.NewKeyFromSeed(seed)
	pub := priv.Public().(ed25519.PublicKey)
	return pub, priv, nil
}

// Sign signs the given message using the private key and returns the 64-byte signature.
// It panics if len(privKey) is not PrivateKeySize (64 bytes).
func Sign(privKey PrivateKey, message []byte) []byte {
	if len(privKey) != PrivateKeySize {
		panic(fmt.Sprintf("sign: invalid private key length %d", len(privKey)))
	}
	return ed25519.Sign(privKey, message)
}

// orderL is the order of the Curve25519 subgroup: 2^252 + 27742317777372353535851937790883648493 in little-endian.
var orderL = [32]byte{
	0xed, 0xd3, 0xf5, 0x5c, 0x1a, 0x63, 0x12, 0x58,
	0xd6, 0x9c, 0xf7, 0xa2, 0xde, 0xf9, 0xde, 0x14,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10,
}

func isCanonicalS(s []byte) bool {
	for i := 31; i >= 0; i-- {
		if s[i] < orderL[i] {
			return true
		}
		if s[i] > orderL[i] {
			return false
		}
	}
	return false // S == L is invalid; S must be strictly < L
}

func isCanonicalPublicKey(pub []byte) bool {
	if len(pub) != PublicKeySize {
		return false
	}
	yHigh := pub[31] & 0x7F
	if yHigh < 0x7F {
		return true
	}
	// If yHigh == 0x7F, check if y >= 2^255 - 19 (non-canonical encoding)
	for i := 30; i >= 1; i-- {
		if pub[i] != 0xFF {
			return true
		}
	}
	return pub[0] < 0xED
}

// Verify verifies whether signature is a valid Ed25519 signature of message by pubKey.
// It strictly enforces non-malleability (S < L per RFC 8032) and canonical public key encodings.
func Verify(pubKey PublicKey, message, signature []byte) bool {
	if len(pubKey) != PublicKeySize || len(signature) != SignatureSize {
		return false
	}
	if !isCanonicalPublicKey(pubKey) || !isCanonicalS(signature[32:]) {
		return false
	}
	return ed25519.Verify(pubKey, message, signature)
}

// FingerprintSHA256 returns the standard OpenSSH-compatible SHA-256 fingerprint
// of the public key (e.g. "SHA256:...").
func FingerprintSHA256(pubKey PublicKey) string {
	if len(pubKey) != PublicKeySize {
		return ""
	}
	// OpenSSH wire format for Ed25519 public key:
	// string "ssh-ed25519" (4-byte length prefix 11 + 11 bytes)
	// string pubKey bytes (4-byte length prefix 32 + 32 bytes)
	var wire [51]byte
	wire[3] = 11
	copy(wire[4:15], "ssh-ed25519")
	wire[18] = 32
	copy(wire[19:], pubKey)

	h := sha256.Sum256(wire[:])
	return "SHA256:" + base64.RawStdEncoding.EncodeToString(h[:])
}

// FingerprintHex returns the colon-delimited lowercase hex SHA-256 fingerprint.
func FingerprintHex(pubKey PublicKey) string {
	if len(pubKey) != PublicKeySize {
		return ""
	}
	h := sha256.Sum256(pubKey)
	parts := make([]string, len(h))
	for i, b := range h {
		parts[i] = fmt.Sprintf("%02x", b)
	}
	return strings.Join(parts, ":")
}

// EncodePrivateKeyPEM serializes an Ed25519 private key into PKCS#8 PEM format.
func EncodePrivateKeyPEM(privKey PrivateKey) ([]byte, error) {
	pkcs8Bytes, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		return nil, fmt.Errorf("sign: marshal pkcs8: %w", err)
	}
	block := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: pkcs8Bytes,
	}
	return pem.EncodeToMemory(block), nil
}

// DecodePrivateKeyPEM deserializes an Ed25519 private key from PKCS#8 PEM format.
func DecodePrivateKeyPEM(pemData []byte) (PrivateKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil || block.Type != "PRIVATE KEY" {
		return nil, ErrInvalidPEM
	}

	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("sign: parse pkcs8: %w", err)
	}

	privKey, ok := parsed.(ed25519.PrivateKey)
	if !ok || len(privKey) != PrivateKeySize {
		return nil, errors.New("sign: key is not an Ed25519 private key")
	}

	return privKey, nil
}

// EncodePublicKeyPEM serializes an Ed25519 public key into PKIX PEM format.
func EncodePublicKeyPEM(pubKey PublicKey) ([]byte, error) {
	pkixBytes, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return nil, fmt.Errorf("sign: marshal pkix: %w", err)
	}
	block := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pkixBytes,
	}
	return pem.EncodeToMemory(block), nil
}

// DecodePublicKeyPEM deserializes an Ed25519 public key from PKIX PEM format.
func DecodePublicKeyPEM(pemData []byte) (PublicKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, ErrInvalidPEM
	}

	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("sign: parse pkix: %w", err)
	}

	pubKey, ok := parsed.(ed25519.PublicKey)
	if !ok || len(pubKey) != PublicKeySize {
		return nil, errors.New("sign: key is not an Ed25519 public key")
	}

	return pubKey, nil
}

// PublicKeyHex returns the hexadecimal string representation of the public key.
func PublicKeyHex(pubKey PublicKey) string {
	if len(pubKey) != PublicKeySize {
		return ""
	}
	return hex.EncodeToString(pubKey)
}

// ParsePublicKeyHex decodes a 64-character hex string into an Ed25519 public key.
func ParsePublicKeyHex(hexStr string) (PublicKey, error) {
	clean := strings.TrimSpace(hexStr)
	b, err := hex.DecodeString(clean)
	if err != nil {
		return nil, fmt.Errorf("sign: invalid hex string: %w", err)
	}
	if len(b) != PublicKeySize {
		return nil, fmt.Errorf("%w: expected %d bytes, got %d", ErrInvalidKeyLength, PublicKeySize, len(b))
	}
	return PublicKey(b), nil
}
