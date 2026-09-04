// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package randkit

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
)

// SecureBytes reads n cryptographically secure random bytes from the operating system's CSPRNG.
func SecureBytes(n int) ([]byte, error) {
	if n < 0 {
		return nil, fmt.Errorf("randkit: negative length %d", n)
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return nil, fmt.Errorf("randkit: failed to read secure entropy: %w", err)
	}
	return b, nil
}

// MustSecureBytes reads n cryptographically secure random bytes, panicking if the CSPRNG fails.
func MustSecureBytes(n int) []byte {
	b, err := SecureBytes(n)
	if err != nil {
		panic(err)
	}
	return b
}

// TokenHex generates an n-byte cryptographic random token formatted as a 2n-character lowercase hexadecimal string.
func TokenHex(n int) (string, error) {
	b, err := SecureBytes(n)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// TokenBase64 generates an n-byte cryptographic random token formatted as a standard base64 string.
func TokenBase64(n int) (string, error) {
	b, err := SecureBytes(n)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// TokenURLSafe generates an n-byte cryptographic random token formatted as an unpadded, URL-safe base64 string.
func TokenURLSafe(n int) (string, error) {
	b, err := SecureBytes(n)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// ConstantTimeCompare returns true if slices a and b have equal length and contents,
// executed in constant time proportional to slice length to mitigate timing attacks.
func ConstantTimeCompare(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}

// ConstantTimeEqual returns true if strings a and b have equal length and contents,
// executed in constant time to mitigate timing attacks.
func ConstantTimeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
