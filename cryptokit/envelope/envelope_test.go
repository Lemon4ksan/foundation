// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package envelope_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/lemon4ksan/foundation/cryptokit/aead"
	"github.com/lemon4ksan/foundation/cryptokit/envelope"
	"github.com/lemon4ksan/foundation/cryptokit/kdf"
	"github.com/lemon4ksan/foundation/cryptokit/kms"
	"github.com/lemon4ksan/foundation/silicon/randkit"
)

func TestEnvelope_PasswordUnlock(t *testing.T) {
	secret := []byte("secret-dek-for-container-encryption")
	env, err := envelope.NewEnvelope(secret, aead.AES256GCM)
	if err != nil {
		t.Fatalf("NewEnvelope failed: %v", err)
	}

	password := "CorrectHorseBatteryStaple2026!"
	err = env.AddPasswordSlot(password, kdf.ProfileFast, "user-master-password")
	if err != nil {
		t.Fatalf("AddPasswordSlot failed: %v", err)
	}

	// Unlock with correct password
	unlocked, err := env.UnlockWithPassword(password)
	if err != nil {
		t.Fatalf("UnlockWithPassword failed: %v", err)
	}
	if !bytes.Equal(unlocked, secret) {
		t.Fatalf("unlocked secret mismatch: %s != %s", unlocked, secret)
	}

	// Unlock with wrong password
	_, err = env.UnlockWithPassword("wrong-password-guess")
	if !errors.Is(err, envelope.ErrNoMatchingSlot) {
		t.Fatalf("expected ErrNoMatchingSlot, got %v", err)
	}
}

func TestEnvelope_MultiSlotDualControl(t *testing.T) {
	ctx := context.Background()
	secret := randkit.MustSecureBytes(32)

	env, err := envelope.NewEnvelope(secret, aead.ChaCha20Poly1305)
	if err != nil {
		t.Fatalf("NewEnvelope failed: %v", err)
	}

	// Slot 0: Alice
	alicePass := "AliceSecret123!"
	if err := env.AddPasswordSlot(alicePass, kdf.ProfileFast, "Alice"); err != nil {
		t.Fatalf("add alice: %v", err)
	}

	// Slot 1: Bob
	bobPass := "BobSecret456!"
	if err := env.AddPasswordSlot(bobPass, kdf.ProfileFast, "Bob"); err != nil {
		t.Fatalf("add bob: %v", err)
	}

	// Slot 2: KMS Mock
	kmsClient := kms.NewMockClient()
	keyURI := "mock://company-escrow-key"
	if err := env.AddKMSSlot(ctx, kmsClient, keyURI, "Cloud Escrow"); err != nil {
		t.Fatalf("add kms: %v", err)
	}

	// Slot 3: Raw KEK
	rawKEK := randkit.MustSecureBytes(32)
	if err := env.AddKEKSlot(rawKEK, "Recovery Hardware Key"); err != nil {
		t.Fatalf("add kek: %v", err)
	}

	if len(env.Slots) != 4 {
		t.Fatalf("expected 4 slots, got %d", len(env.Slots))
	}

	// Serialize to binary format
	var buf bytes.Buffer
	if err := env.WriteBinary(&buf); err != nil {
		t.Fatalf("WriteBinary failed: %v", err)
	}

	// Deserialize from binary format
	parsedEnv, err := envelope.ReadBinary(&buf)
	if err != nil {
		t.Fatalf("ReadBinary failed: %v", err)
	}

	// 1. Unlock with Alice
	secAlice, err := parsedEnv.UnlockWithPassword(alicePass)
	if err != nil || !bytes.Equal(secAlice, secret) {
		t.Fatalf("alice unlock failed: %v", err)
	}

	// 2. Unlock with Bob
	secBob, err := parsedEnv.UnlockWithPassword(bobPass)
	if err != nil || !bytes.Equal(secBob, secret) {
		t.Fatalf("bob unlock failed: %v", err)
	}

	// 3. Unlock with KMS
	secKMS, err := parsedEnv.UnlockWithKMS(ctx, kmsClient)
	if err != nil || !bytes.Equal(secKMS, secret) {
		t.Fatalf("kms unlock failed: %v", err)
	}

	// 4. Unlock with raw KEK
	secKEK, err := parsedEnv.UnlockWithKEK(rawKEK)
	if err != nil || !bytes.Equal(secKEK, secret) {
		t.Fatalf("kek unlock failed: %v", err)
	}

	// 5. Wrong KMS client should fail
	wrongKMS := kms.NewMockClient()
	_, err = parsedEnv.UnlockWithKMS(ctx, wrongKMS)
	if !errors.Is(err, envelope.ErrNoMatchingSlot) {
		t.Fatalf("expected ErrNoMatchingSlot for wrong KMS, got %v", err)
	}

	// 6. Wrong KEK should fail
	wrongKEK := randkit.MustSecureBytes(32)
	_, err = parsedEnv.UnlockWithKEK(wrongKEK)
	if !errors.Is(err, envelope.ErrNoMatchingSlot) {
		t.Fatalf("expected ErrNoMatchingSlot for wrong KEK, got %v", err)
	}
}

func TestEnvelope_TamperedBinary(t *testing.T) {
	env, _ := envelope.NewEnvelope(nil)
	_ = env.AddPasswordSlot("pass", kdf.ProfileFast)

	var buf bytes.Buffer
	_ = env.WriteBinary(&buf)
	data := buf.Bytes()

	// Corrupt checksum at the end
	corruptChecksum := append([]byte(nil), data...)
	corruptChecksum[len(corruptChecksum)-1] ^= 0xAA
	_, err := envelope.ReadBinary(bytes.NewReader(corruptChecksum))
	if !errors.Is(err, envelope.ErrInvalidEnvelope) {
		t.Fatalf("expected ErrInvalidEnvelope on corrupt checksum, got %v", err)
	}

	// Corrupt magic header
	corruptMagic := append([]byte(nil), data...)
	corruptMagic[0] = 'X'
	_, err = envelope.ReadBinary(bytes.NewReader(corruptMagic))
	if !errors.Is(err, envelope.ErrInvalidEnvelope) {
		t.Fatalf("expected ErrInvalidEnvelope on corrupt magic, got %v", err)
	}
}
