// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sign_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/lemon4ksan/foundation/cryptokit/sign"
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
