// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kdf_test

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/lemon4ksan/foundation/cryptokit/kdf"
)

func decodeHex(s string) []byte {
	b, _ := hex.DecodeString(s)
	return b
}

func TestHKDF_RFC5869TestCase1(t *testing.T) {
	// RFC 5869 Test Case 1 with SHA-256
	ikm := decodeHex("0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b")
	salt := decodeHex("000102030405060708090a0b0c")
	info := decodeHex("f0f1f2f3f4f5f6f7f8f9")
	l := 42

	expectedPRK := decodeHex("077709362c2e32df0ddc3f0dc47bba6390b6c73bb50f9c3122ec844ad7c2b3e5")
	expectedOKM := decodeHex("3cb25f25faacd57a90434f64d0362f2a2d2d0a90cf1a5a4c5db02d56ecc4c5bf34007208d5b887185865")

	prk := kdf.Extract(sha256.New, ikm, salt)
	if !bytes.Equal(prk, expectedPRK) {
		t.Fatalf("PRK mismatch:\nexpected %x\ngot      %x", expectedPRK, prk)
	}

	okm, err := kdf.Expand(sha256.New, prk, info, l)
	if err != nil {
		t.Fatalf("Expand failed: %v", err)
	}
	if !bytes.Equal(okm, expectedOKM) {
		t.Fatalf("OKM mismatch:\nexpected %x\ngot      %x", expectedOKM, okm)
	}

	derived, err := kdf.DeriveKey(sha256.New, ikm, salt, info, l)
	if err != nil || !bytes.Equal(derived, expectedOKM) {
		t.Fatalf("DeriveKey mismatch")
	}
}

func TestPBKDF2_RFC6070TestCase1(t *testing.T) {
	// RFC 6070 Test Case 1: password="password", salt="salt", c=1, dkLen=20, SHA-1
	// Expected: 0c60c80f961f0e71f3a9b524af6012062fe037a6
	password := []byte("password")
	salt := []byte("salt")
	iter := 1
	keyLen := 20
	expected := decodeHex("0c60c80f961f0e71f3a9b524af6012062fe037a6")

	dk := kdf.PBKDF2(sha1.New, password, salt, iter, keyLen)
	if !bytes.Equal(dk, expected) {
		t.Fatalf("PBKDF2 mismatch:\nexpected %x\ngot      %x", expected, dk)
	}
}

func TestArgon2id_FastProfile(t *testing.T) {
	pass := []byte("test-vault-password")
	salt := []byte("1234567890123456")
	k1 := kdf.Argon2id(pass, salt, kdf.ProfileFast, 32)
	k2 := kdf.Argon2id(pass, salt, kdf.ProfileFast, 32)
	if len(k1) != 32 || !bytes.Equal(k1, k2) {
		t.Fatal("Argon2id deterministic output mismatch or invalid length")
	}
}

func TestDeriveSubkeyAndNonce(t *testing.T) {
	master := []byte("01234567890123456789012345678901")
	sub1 := kdf.DeriveSubkey(master, "domain-a", 32)
	sub2 := kdf.DeriveSubkey(master, "domain-b", 32)
	if bytes.Equal(sub1, sub2) {
		t.Fatal("subkeys for different domains must not collide")
	}

	iv := []byte("master-iv-012345")
	nonce0 := kdf.DeriveNonce(iv, "chunk", 0, 12)
	nonce1 := kdf.DeriveNonce(iv, "chunk", 1, 12)
	if bytes.Equal(nonce0, nonce1) {
		t.Fatal("nonces for different counters must not collide")
	}
}
