// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package kdf provides standard Key Derivation Functions (HKDF RFC 5869, PBKDF2 RFC 2898,
// Argon2id profiles, and deterministic domain-separated subkey and nonce derivation).
package kdf

import (
	"crypto/hmac"
	"errors"
	"fmt"
	"hash"
)

var (
	// ErrBadLength indicates the requested derived key length exceeds the maximum allowed by HKDF.
	ErrBadLength = errors.New("hkdf: requested key length exceeds maximum limit")
)

// Extract implements HKDF-Extract (RFC 5869 Section 2.2).
// It extracts a pseudorandom key (PRK) of hash length from the input keying material (secret) and salt.
// If salt is nil or empty, a string of HashLen zeros is used.
func Extract(h func() hash.Hash, secret, salt []byte) []byte {
	hasher := h()
	if len(salt) == 0 {
		salt = make([]byte, hasher.Size())
	}
	mac := hmac.New(h, salt)
	mac.Write(secret)
	return mac.Sum(nil)
}

// Expand implements HKDF-Expand (RFC 5869 Section 2.3).
// It expands a pseudorandom key (PRK) with domain context info into an output key of the specified length.
func Expand(h func() hash.Hash, prk, info []byte, length int) ([]byte, error) {
	if length < 0 {
		return nil, ErrBadLength
	}
	if length == 0 {
		return []byte{}, nil
	}

	hashLen := h().Size()
	limit := 255 * hashLen
	if length > limit {
		return nil, fmt.Errorf("%w: %d > %d", ErrBadLength, length, limit)
	}

	n := (length + hashLen - 1) / hashLen
	out := make([]byte, length)

	var prev []byte
	offset := 0

	for i := 1; i <= n; i++ {
		mac := hmac.New(h, prk)
		if i > 1 {
			mac.Write(prev)
		}
		mac.Write(info)
		mac.Write([]byte{byte(i)})
		prev = mac.Sum(prev[:0])

		toCopy := hashLen
		if offset+toCopy > length {
			toCopy = length - offset
		}
		copy(out[offset:offset+toCopy], prev[:toCopy])
		offset += toCopy
	}

	return out, nil
}

// DeriveKey combines HKDF-Extract and HKDF-Expand into a single call.
func DeriveKey(h func() hash.Hash, secret, salt, info []byte, length int) ([]byte, error) {
	prk := Extract(h, secret, salt)
	return Expand(h, prk, info, length)
}
