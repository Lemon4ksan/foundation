// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shamir_test

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/lemon4ksan/foundation/cryptokit/shamir"
)

func TestGF256_Properties(t *testing.T) {
	// 1. Check all non-zero elements are invertible
	for a := 1; a < 256; a++ {
		inv := shamir.Div(1, byte(a))
		prod := shamir.Mul(byte(a), inv)
		if prod != 1 {
			t.Fatalf("GF(2^8) inversion failed for %d: %d * %d = %d", a, a, inv, prod)
		}
	}

	// 2. Identity and zero
	for a := 0; a < 256; a++ {
		if shamir.Mul(byte(a), 0) != 0 || shamir.Mul(0, byte(a)) != 0 {
			t.Fatalf("multiplication by zero failed for %d", a)
		}
		if shamir.Mul(byte(a), 1) != byte(a) || shamir.Mul(1, byte(a)) != byte(a) {
			t.Fatalf("multiplication by identity failed for %d", a)
		}
		if shamir.Add(byte(a), byte(a)) != 0 {
			t.Fatalf("self addition not zero for %d", a)
		}
	}
}

func TestShamir_SplitAndCombine_3of5(t *testing.T) {
	secret := make([]byte, 32)
	_, _ = rand.Read(secret)

	threshold := 3
	total := 5

	shares, err := shamir.Split(secret, threshold, total)
	if err != nil {
		t.Fatalf("Split failed: %v", err)
	}

	if len(shares) != total {
		t.Fatalf("expected %d shares, got %d", total, len(shares))
	}

	// Test all 10 combinations of 3-of-5:
	// (0,1,2), (0,1,3), (0,1,4), (0,2,3), (0,2,4), (0,3,4), (1,2,3), (1,2,4), (1,3,4), (2,3,4)
	combos := [][]int{
		{0, 1, 2},
		{0, 1, 3},
		{0, 1, 4},
		{0, 2, 3},
		{0, 2, 4},
		{0, 3, 4},
		{1, 2, 3},
		{1, 2, 4},
		{1, 3, 4},
		{2, 3, 4},
	}

	for _, c := range combos {
		subset := []shamir.Share{shares[c[0]], shares[c[1]], shares[c[2]]}
		recovered, err := shamir.Combine(subset)
		if err != nil {
			t.Fatalf("Combine combo %v failed: %v", c, err)
		}
		if !bytes.Equal(recovered, secret) {
			t.Fatalf("Combine combo %v produced wrong secret", c)
		}
	}

	// Test 4 shares
	subset4 := []shamir.Share{shares[0], shares[1], shares[3], shares[4]}
	recovered4, err := shamir.Combine(subset4)
	if err != nil || !bytes.Equal(recovered4, secret) {
		t.Fatalf("Combine 4 shares failed")
	}

	// Test all 5 shares
	recovered5, err := shamir.Combine(shares)
	if err != nil || !bytes.Equal(recovered5, secret) {
		t.Fatalf("Combine all shares failed")
	}

	// Test insufficient shares (2 shares when threshold is 3)
	subset2 := []shamir.Share{shares[0], shares[1]}
	recovered2, err := shamir.Combine(subset2)
	if err == nil && bytes.Equal(recovered2, secret) {
		t.Fatalf("2 shares should NOT reconstruct the secret for threshold 3!")
	}
}

func TestShamir_FormatAndParse(t *testing.T) {
	secret := []byte("AES-256-GCM MASTER DEK PAYLOAD!!")
	shares, err := shamir.Split(secret, 3, 5)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}

	var formatted []string
	for _, s := range shares {
		str := shamir.FormatShare(s, 3, 5)
		formatted = append(formatted, str)

		parsed, th, tot, err := shamir.ParseShare(str)
		if err != nil {
			t.Fatalf("ParseShare(%s) failed: %v", str, err)
		}
		if th != 3 || tot != 5 {
			t.Fatalf("expected 3-of-5, got %d-of-%d", th, tot)
		}
		if parsed.Index != s.Index || !bytes.Equal(parsed.Data, s.Data) {
			t.Fatalf("parsed share does not match original")
		}
	}

	// Combine from formatted strings (taking shares 1, 3, 4)
	p1, _, _, _ := shamir.ParseShare(formatted[0])
	p3, _, _, _ := shamir.ParseShare(formatted[2])
	p4, _, _, _ := shamir.ParseShare(formatted[3])

	rec, err := shamir.Combine([]shamir.Share{p1, p3, p4})
	if err != nil || !bytes.Equal(rec, secret) {
		t.Fatalf("reconstruction from parsed strings failed: %v", err)
	}

	// Test tampering detection
	tampered := formatted[0][:len(formatted[0])-1] + "0"
	if _, _, _, err := shamir.ParseShare(tampered); err == nil {
		t.Fatalf("expected tamper detection to fail on modified share")
	}
}
