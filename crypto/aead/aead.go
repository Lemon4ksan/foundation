// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package aead

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
)

// KeySize is the expected key length (32 bytes / 256 bits) for all supported AEAD ciphers.
const KeySize = 32

// Algorithm identifies a supported authenticated encryption with associated data (AEAD) cipher.
type Algorithm uint8

const (
	// AES256GCM represents AES-256 in Galois/Counter Mode (RFC 5116).
	// Standard nonce size: 12 bytes.
	AES256GCM Algorithm = 1

	// ChaCha20Poly1305 represents ChaCha20-Poly1305 AEAD (RFC 8439).
	// Standard nonce size: 12 bytes.
	ChaCha20Poly1305 Algorithm = 2

	// XChaCha20Poly1305 represents extended-nonce ChaCha20-Poly1305.
	// Standard nonce size: 24 bytes (safe for random nonce generation).
	XChaCha20Poly1305 Algorithm = 3
)

var (
	// ErrInvalidKeyLength indicates that the supplied key does not match KeySize (32 bytes).
	ErrInvalidKeyLength = errors.New("aead: invalid key length; must be 32 bytes")

	// ErrUnsupportedAlgorithm indicates that the requested AEAD algorithm ID is unrecognized.
	ErrUnsupportedAlgorithm = errors.New("aead: unsupported algorithm")

	// ErrInvalidNonceLength indicates that the nonce does not match the cipher's required nonce size.
	ErrInvalidNonceLength = errors.New("aead: invalid nonce length")
)

// String returns a human-readable name of the AEAD algorithm.
func (a Algorithm) String() string {
	switch a {
	case AES256GCM:
		return "AES-256-GCM"
	case ChaCha20Poly1305:
		return "ChaCha20-Poly1305"
	case XChaCha20Poly1305:
		return "XChaCha20-Poly1305"
	default:
		return fmt.Sprintf("Unknown-AEAD(0x%02X)", uint8(a))
	}
}

// NonceSize returns the standard nonce length in bytes for the specified algorithm.
func (a Algorithm) NonceSize() (int, error) {
	switch a {
	case AES256GCM:
		return 12, nil
	case ChaCha20Poly1305:
		return 12, nil
	case XChaCha20Poly1305:
		return chacha20poly1305.NonceSizeX, nil // 24 bytes
	default:
		return 0, fmt.Errorf("%w: %d", ErrUnsupportedAlgorithm, a)
	}
}

// New instantiates a cipher.AEAD for the specified algorithm and 256-bit (32-byte) key.
func New(algo Algorithm, key []byte) (cipher.AEAD, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("%w: got %d bytes, expected %d", ErrInvalidKeyLength, len(key), KeySize)
	}

	switch algo {
	case AES256GCM:
		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, fmt.Errorf("aead: aes cipher creation failed: %w", err)
		}
		return cipher.NewGCM(block)

	case ChaCha20Poly1305:
		return chacha20poly1305.New(key)

	case XChaCha20Poly1305:
		return chacha20poly1305.NewX(key)

	default:
		return nil, fmt.Errorf("%w: %d", ErrUnsupportedAlgorithm, algo)
	}
}

// Seal encrypts and authenticates plaintext along with additionalData using the specified algorithm and key.
func Seal(algo Algorithm, key, nonce, plaintext, additionalData []byte) ([]byte, error) {
	aead, err := New(algo, key)
	if err != nil {
		return nil, err
	}
	if len(nonce) != aead.NonceSize() {
		return nil, fmt.Errorf("%w: expected %d bytes, got %d", ErrInvalidNonceLength, aead.NonceSize(), len(nonce))
	}
	return aead.Seal(nil, nonce, plaintext, additionalData), nil
}

// Open authenticates and decrypts ciphertext along with additionalData using the specified algorithm and key.
func Open(algo Algorithm, key, nonce, ciphertext, additionalData []byte) ([]byte, error) {
	aead, err := New(algo, key)
	if err != nil {
		return nil, err
	}
	if len(nonce) != aead.NonceSize() {
		return nil, fmt.Errorf("%w: expected %d bytes, got %d", ErrInvalidNonceLength, aead.NonceSize(), len(nonce))
	}
	return aead.Open(nil, nonce, ciphertext, additionalData)
}
