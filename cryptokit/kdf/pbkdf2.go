// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kdf

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"hash"
)

// PBKDF2 derives a cryptographic key from a password and salt using PBKDF2 (RFC 2898 / PKCS #5 v2.0).
func PBKDF2(h func() hash.Hash, password, salt []byte, iter, keyLen int) []byte {
	prf := hmac.New(h, password)
	hashLen := prf.Size()
	numBlocks := (keyLen + hashLen - 1) / hashLen

	var (
		buf [4]byte
		dk  = make([]byte, 0, numBlocks*hashLen)
		u   = make([]byte, hashLen)
		t   = make([]byte, hashLen)
	)

	for block := 1; block <= numBlocks; block++ {
		// Initial iteration: U_1 = PRF(Password, Salt || INT_32_BE(block))
		prf.Reset()
		prf.Write(salt)
		binary.BigEndian.PutUint32(buf[:], uint32(block))
		prf.Write(buf[:])
		u = prf.Sum(u[:0])
		copy(t, u)

		// Subsequent iterations: U_c = PRF(Password, U_{c-1})
		for c := 2; c <= iter; c++ {
			prf.Reset()
			prf.Write(u)
			u = prf.Sum(u[:0])
			for x := 0; x < hashLen; x++ {
				t[x] ^= u[x]
			}
		}

		dk = append(dk, t...)
	}

	return dk[:keyLen]
}

// PBKDF2SHA256 is a convenience wrapper for PBKDF2 using SHA-256.
func PBKDF2SHA256(password, salt []byte, iter, keyLen int) []byte {
	return PBKDF2(sha256.New, password, salt, iter, keyLen)
}
