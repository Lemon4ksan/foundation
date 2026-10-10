// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sign_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"

	"github.com/lemon4ksan/foundation/crypto/sign"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

func TestSign_Roundtrip(t *testing.T) {
	pub, priv, err := sign.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	if len(pub) != sign.PublicKeySize {
		t.Fatalf("unexpected pub key size: %d", len(pub))
	}
	if len(priv) != sign.PrivateKeySize {
		t.Fatalf("unexpected priv key size: %d", len(priv))
	}

	message := []byte("foundation-ed25519-manifest-statement-2026")
	sig := sign.Sign(priv, message)

	if len(sig) != sign.SignatureSize {
		t.Fatalf("unexpected signature size: %d", len(sig))
	}

	// Valid verification
	if !sign.Verify(pub, message, sig) {
		t.Fatal("expected signature verification to succeed")
	}

	// Tampered message
	tamperedMsg := []byte("foundation-ed25519-manifest-statement-2027")
	if sign.Verify(pub, tamperedMsg, sig) {
		t.Fatal("expected verification to fail with tampered message")
	}

	// Tampered signature
	corruptSig := append([]byte(nil), sig...)
	corruptSig[0] ^= 0xFF
	if sign.Verify(pub, message, corruptSig) {
		t.Fatal("expected verification to fail with corrupt signature")
	}

	// Wrong public key
	pub2, _, _ := sign.GenerateKey()
	if sign.Verify(pub2, message, sig) {
		t.Fatal("expected verification to fail with different public key")
	}
}

func TestSign_FromSeed(t *testing.T) {
	seed := bytes.Repeat([]byte{0x42}, sign.SeedSize)
	pub1, priv1, err := sign.NewKeyFromSeed(seed)
	if err != nil {
		t.Fatalf("NewKeyFromSeed failed: %v", err)
	}

	pub2, priv2, err := sign.NewKeyFromSeed(seed)
	if err != nil {
		t.Fatalf("NewKeyFromSeed failed: %v", err)
	}

	if !bytes.Equal(pub1, pub2) || !bytes.Equal(priv1, priv2) {
		t.Fatal("expected identical keys derived from identical seed")
	}

	// Invalid seed size
	_, _, err = sign.NewKeyFromSeed(seed[:16])
	if err == nil {
		t.Fatal("expected error for invalid seed size")
	}
}

func TestSign_Fingerprints(t *testing.T) {
	pub, _, err := sign.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	fp := sign.FingerprintSHA256(pub)
	if !strings.HasPrefix(fp, "SHA256:") {
		t.Fatalf("expected SHA256: prefix, got: %s", fp)
	}

	fpHex := sign.FingerprintHex(pub)
	parts := strings.Split(fpHex, ":")
	if len(parts) != 32 {
		t.Fatalf("expected 32 colon-separated hex octets, got: %d (%s)", len(parts), fpHex)
	}
}

func TestRepro_OpenSSHFingerprintIncompatibility(t *testing.T) {
	pub, _, err := sign.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	sshKey, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}

	expected := ssh.FingerprintSHA256(sshKey)
	actual := sign.FingerprintSHA256(pub)

	if actual != expected {
		t.Fatalf("OpenSSH fingerprint mismatch!\nExpected: %s\nActual:   %s", expected, actual)
	}
}

func TestSign_PEMSerialization(t *testing.T) {
	pub, priv, err := sign.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	// 1. Private Key PEM
	privPEM, err := sign.EncodePrivateKeyPEM(priv)
	if err != nil {
		t.Fatalf("EncodePrivateKeyPEM failed: %v", err)
	}
	if !strings.Contains(string(privPEM), "-----BEGIN PRIVATE KEY-----") {
		t.Fatalf("invalid private key PEM header: %s", string(privPEM))
	}

	parsedPriv, err := sign.DecodePrivateKeyPEM(privPEM)
	if err != nil {
		t.Fatalf("DecodePrivateKeyPEM failed: %v", err)
	}
	if !bytes.Equal(parsedPriv, priv) {
		t.Fatal("decoded private key does not match original")
	}

	// 2. Public Key PEM
	pubPEM, err := sign.EncodePublicKeyPEM(pub)
	if err != nil {
		t.Fatalf("EncodePublicKeyPEM failed: %v", err)
	}
	if !strings.Contains(string(pubPEM), "-----BEGIN PUBLIC KEY-----") {
		t.Fatalf("invalid public key PEM header: %s", string(pubPEM))
	}

	parsedPub, err := sign.DecodePublicKeyPEM(pubPEM)
	if err != nil {
		t.Fatalf("DecodePublicKeyPEM failed: %v", err)
	}
	if !bytes.Equal(parsedPub, pub) {
		t.Fatal("decoded public key does not match original")
	}

	// 3. Verify signature using decoded keys
	msg := []byte("test-pem-roundtrip")
	sig := sign.Sign(parsedPriv, msg)
	if !sign.Verify(parsedPub, msg, sig) {
		t.Fatal("verification failed with PEM-roundtripped keys")
	}

	// 4. Invalid PEM input
	_, err = sign.DecodePrivateKeyPEM([]byte("garbage not pem"))
	if err == nil {
		t.Fatal("expected error on invalid PEM")
	}
}

func TestSign_Hex(t *testing.T) {
	pub, _, err := sign.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	hexStr := sign.PublicKeyHex(pub)
	if len(hexStr) != 64 {
		t.Fatalf("expected 64 hex characters, got %d", len(hexStr))
	}

	parsed, err := sign.ParsePublicKeyHex(hexStr)
	if err != nil {
		t.Fatalf("ParsePublicKeyHex failed: %v", err)
	}
	if !bytes.Equal(parsed, pub) {
		t.Fatal("hex decoded public key mismatch")
	}

	_, err = sign.ParsePublicKeyHex("invalid-hex-chars!")
	if err == nil {
		t.Fatal("expected error on invalid hex")
	}
}

func TestRepro_CorruptedKeyHandling(t *testing.T) {
	pub, priv, err := sign.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	// 1. Wrong PEM type in DecodePrivateKeyPEM
	pubPEM, err := sign.EncodePublicKeyPEM(pub)
	if err != nil {
		t.Fatal(err)
	}
	_, err = sign.DecodePrivateKeyPEM(pubPEM)
	if err == nil {
		t.Fatal("expected error when decoding public key PEM as private key")
	}

	// 2. Wrong PEM type in DecodePublicKeyPEM
	privPEM, err := sign.EncodePrivateKeyPEM(priv)
	if err != nil {
		t.Fatal(err)
	}
	_, err = sign.DecodePublicKeyPEM(privPEM)
	if err == nil {
		t.Fatal("expected error when decoding private key PEM as public key")
	}

	// 3. Corrupted key length handling
	if fp := sign.FingerprintSHA256([]byte("too-short")); fp != "" {
		t.Fatalf("expected empty fingerprint for invalid key size, got %s", fp)
	}
	if fpHex := sign.FingerprintHex([]byte("too-short")); fpHex != "" {
		t.Fatalf("expected empty hex fingerprint for invalid key size, got %s", fpHex)
	}
	if hexVal := sign.PublicKeyHex([]byte("too-short")); hexVal != "" {
		t.Fatalf("expected empty hex for invalid key size, got %s", hexVal)
	}

	// 4. Verify with corrupted public key
	corruptedPub := make([]byte, sign.PublicKeySize)
	copy(corruptedPub, pub)
	corruptedPub[0] ^= 0xFF
	msg := []byte("hello")
	sig := sign.Sign(priv, msg)
	if sign.Verify(corruptedPub, msg, sig) {
		t.Fatal("verification must fail with corrupted public key")
	}
}

func FuzzParsePublicKeyHex(f *testing.F) {
	pub, _, _ := sign.GenerateKey()
	f.Add(sign.PublicKeyHex(pub))
	f.Add("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	f.Add("")
	f.Add("zzzz")

	f.Fuzz(func(t *testing.T, hexStr string) {
		k, err := sign.ParsePublicKeyHex(hexStr)
		if err == nil {
			if len(k) != sign.PublicKeySize {
				t.Errorf("parsed key had unexpected length: %d", len(k))
			}
		}
	})
}

func TestRepro_Sign_SignatureMalleabilityAndTruncation(t *testing.T) {
	pub, priv, err := sign.GenerateKey()
	require.NoError(t, err)

	msg := []byte("foundation-ed25519-strict-verification")
	sig := sign.Sign(priv, msg)
	require.Equal(t, sign.SignatureSize, len(sig))

	// Valid verification
	assert.True(t, sign.Verify(pub, msg, sig))

	// Truncated signatures must return false immediately
	assert.False(t, sign.Verify(pub, msg, nil))
	assert.False(t, sign.Verify(pub, msg, []byte{}))
	assert.False(t, sign.Verify(pub, msg, sig[:10]))
	assert.False(t, sign.Verify(pub, msg, sig[:63]))
	assert.False(t, sign.Verify(pub, msg, append(sig, 0x00))) // 65 bytes

	// Truncated or invalid public keys must return false
	assert.False(t, sign.Verify(nil, msg, sig))
	assert.False(t, sign.Verify([]byte{}, msg, sig))
	assert.False(t, sign.Verify(pub[:31], msg, sig))

	// Malleable signature: S >= L
	// Order L = 2^252 + 27742317777372353535851937790883648493
	orderL := []byte{
		0xed, 0xd3, 0xf5, 0x5c, 0x1a, 0x63, 0x12, 0x58,
		0xd6, 0x9c, 0xf7, 0xa2, 0xde, 0xf9, 0xde, 0x14,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10,
	}

	malleableSig := append([]byte(nil), sig...)
	copy(malleableSig[32:], orderL) // S = L (invalid, must be S < L)
	assert.False(t, sign.Verify(pub, msg, malleableSig))

	// S > L
	malleableSig2 := append([]byte(nil), sig...)
	malleableSig2[63] = 0xFF // high byte exceeds 0x10
	assert.False(t, sign.Verify(pub, msg, malleableSig2))

	// Non-canonical public key: y >= 2^255 - 19
	nonCanonicalPub := make([]byte, 32)
	for i := 1; i <= 30; i++ {
		nonCanonicalPub[i] = 0xFF
	}
	nonCanonicalPub[31] = 0x7F
	nonCanonicalPub[0] = 0xEE // 2^255 - 18 >= 2^255 - 19
	assert.False(t, sign.Verify(nonCanonicalPub, msg, sig))
}

func TestRepro_Sign_SeedDerivationBoundaryConditions(t *testing.T) {
	// 1. Boundary sizes: 0, 31, 33, 64 bytes
	testCases := []int{0, 1, 16, 31, 33, 64, 128}
	for _, size := range testCases {
		seed := make([]byte, size)
		_, _, err := sign.NewKeyFromSeed(seed)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, sign.ErrInvalidKeyLength))
	}

	// 2. nil seed
	_, _, err := sign.NewKeyFromSeed(nil)
	assert.Error(t, err)

	// 3. Exactly 32 bytes with extreme patterns: all-zeros, all-ones
	allZero := make([]byte, sign.SeedSize)
	pub0, priv0, err := sign.NewKeyFromSeed(allZero)
	require.NoError(t, err)
	assert.Equal(t, sign.PublicKeySize, len(pub0))
	assert.Equal(t, sign.PrivateKeySize, len(priv0))

	allOnes := bytes.Repeat([]byte{0xFF}, sign.SeedSize)
	pubFF, privFF, err := sign.NewKeyFromSeed(allOnes)
	require.NoError(t, err)
	assert.Equal(t, sign.PublicKeySize, len(pubFF))
	assert.Equal(t, sign.PrivateKeySize, len(privFF))
	assert.False(t, bytes.Equal(pub0, pubFF))
}


