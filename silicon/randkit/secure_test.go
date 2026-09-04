// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package randkit_test

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/lemon4ksan/foundation/silicon/randkit"
)

func TestSecureBytesAndTokens(t *testing.T) {
	b1, err := randkit.SecureBytes(32)
	if err != nil || len(b1) != 32 {
		t.Fatalf("SecureBytes(32) failed: %v, len=%d", err, len(b1))
	}
	b2 := randkit.MustSecureBytes(32)
	if randkit.ConstantTimeCompare(b1, b2) {
		t.Fatal("two secure random slices are identical, impossible with secure CSPRNG")
	}

	tokHex, err := randkit.TokenHex(16)
	if err != nil || len(tokHex) != 32 {
		t.Fatalf("TokenHex(16) failed: len=%d, tok=%s", len(tokHex), tokHex)
	}
	if _, err := hex.DecodeString(tokHex); err != nil {
		t.Fatalf("TokenHex produced invalid hex: %v", err)
	}

	tokB64, err := randkit.TokenBase64(24)
	if err != nil || len(tokB64) == 0 {
		t.Fatalf("TokenBase64(24) failed: %v", err)
	}

	tokURL, err := randkit.TokenURLSafe(32)
	if err != nil || len(tokURL) == 0 {
		t.Fatalf("TokenURLSafe(32) failed: %v", err)
	}
	if strings.ContainsAny(tokURL, "+/=") {
		t.Fatalf("TokenURLSafe contains forbidden characters (+/=): %s", tokURL)
	}
}

func TestConstantTime(t *testing.T) {
	s1 := "super-secret-password"
	s2 := "super-secret-password"
	s3 := "super-secret-passwore"

	if !randkit.ConstantTimeEqual(s1, s2) {
		t.Fatal("expected ConstantTimeEqual(s1, s2) == true")
	}
	if randkit.ConstantTimeEqual(s1, s3) {
		t.Fatal("expected ConstantTimeEqual(s1, s3) == false")
	}
	if randkit.ConstantTimeEqual(s1, s1+"x") {
		t.Fatal("expected ConstantTimeEqual with different lengths == false")
	}
}
