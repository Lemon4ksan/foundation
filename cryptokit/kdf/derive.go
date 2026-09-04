// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kdf

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

// DeriveSubkey expands a master key into a domain-separated subkey using HKDF-SHA256.
func DeriveSubkey(masterKey []byte, domainTag string, length int) []byte {
	k, err := Expand(sha256.New, masterKey, []byte(domainTag), length)
	if err != nil {
		panic(fmt.Sprintf("kdf: derive subkey failed: %v", err))
	}
	return k
}

// DeriveNonce derives a deterministic, collision-resistant nonce from a master IV,
// domain tag, and sequential counter using HKDF-SHA256.
func DeriveNonce(masterIV []byte, domainTag string, counter uint64, length int) []byte {
	info := make([]byte, len(domainTag)+8)
	copy(info, domainTag)
	binary.BigEndian.PutUint64(info[len(domainTag):], counter)

	nonce, err := Expand(sha256.New, masterIV, info, length)
	if err != nil {
		panic(fmt.Sprintf("kdf: derive nonce failed: %v", err))
	}
	return nonce
}
