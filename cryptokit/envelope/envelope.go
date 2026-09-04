// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package envelope

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/lemon4ksan/foundation/cryptokit/aead"
	"github.com/lemon4ksan/foundation/cryptokit/kdf"
	"github.com/lemon4ksan/foundation/cryptokit/kms"
	"github.com/lemon4ksan/foundation/silicon/randkit"
)

// Magic header for binary envelope serialization: 'FENV'
var MagicEnvelope = [4]byte{'F', 'E', 'N', 'V'}

var (
	// ErrNoMatchingSlot is returned when no slot could be decrypted with the provided credentials.
	ErrNoMatchingSlot = errors.New("envelope: no matching slot could be decrypted")

	// ErrInvalidEnvelope indicates corrupt or invalid envelope data.
	ErrInvalidEnvelope = errors.New("envelope: invalid or corrupted envelope header")

	// ErrMaxSlotsExceeded is returned when attempting to add more than 32 slots.
	ErrMaxSlotsExceeded = errors.New("envelope: maximum slots (32) exceeded")
)

// SlotType represents the mechanism used to wrap the secret in a slot.
type SlotType uint8

const (
	// SlotTypeArgon2id wraps using a password-derived key via Argon2id.
	SlotTypeArgon2id SlotType = 1

	// SlotTypePBKDF2 wraps using a password-derived key via PBKDF2-SHA256.
	SlotTypePBKDF2 SlotType = 2

	// SlotTypeKMS wraps using an envelope KEK managed by a KMS service.
	SlotTypeKMS SlotType = 3

	// SlotTypeRawKEK wraps using a pre-shared 256-bit symmetric key.
	SlotTypeRawKEK SlotType = 4
)

// Slot contains authentication parameters and encrypted secret payload for one key holder.
type Slot struct {
	Index          uint8    `json:"index"`
	Type           SlotType `json:"type"`
	Label          string   `json:"label,omitempty"`
	Salt           []byte   `json:"salt,omitempty"`
	KDFMemoryKiB   uint32   `json:"kdf_memory_kib,omitempty"`
	KDFIterations  uint32   `json:"kdf_iterations,omitempty"`
	KDFParallelism uint8    `json:"kdf_parallelism,omitempty"`
	KMSKeyURI      string   `json:"kms_key_uri,omitempty"`
	KMSCiphertext  []byte   `json:"kms_ciphertext,omitempty"`
	WrappedSecret  []byte   `json:"wrapped_secret"`
}

// Envelope manages multi-slot secret wrapping, unlocking, and persistence.
type Envelope struct {
	CipherAlgo aead.Algorithm `json:"cipher_algo"`
	MasterIV   [16]byte       `json:"master_iv"`
	Slots      []Slot         `json:"slots"`
	secret     []byte         // cached in memory during creation or after unlocking
}

// NewEnvelope initializes an envelope to protect secret.
// If secret is nil or empty, a secure random 32-byte Data Encryption Key (DEK) is generated.
func NewEnvelope(secret []byte, algo ...aead.Algorithm) (*Envelope, error) {
	cipherAlgo := aead.AES256GCM
	if len(algo) > 0 {
		cipherAlgo = algo[0]
	}

	sec := secret
	if len(sec) == 0 {
		sec = randkit.MustSecureBytes(32)
	}

	var masterIV [16]byte
	ivBytes := randkit.MustSecureBytes(16)
	copy(masterIV[:], ivBytes)

	return &Envelope{
		CipherAlgo: cipherAlgo,
		MasterIV:   masterIV,
		secret:     append([]byte(nil), sec...),
	}, nil
}

// Secret returns the unencrypted secret protected by this envelope, if available.
func (e *Envelope) Secret() []byte {
	return append([]byte(nil), e.secret...)
}

func (e *Envelope) deriveSlotNonce(slotIdx uint8) []byte {
	nonceSize, _ := e.CipherAlgo.NonceSize()
	return kdf.DeriveNonce(e.MasterIV[:], "envelope-slot-nonce", uint64(slotIdx), nonceSize)
}

func slotAAD(slotIdx uint8) []byte {
	return []byte(fmt.Sprintf("foundation-envelope-v1-slot-%d", slotIdx))
}

// AddPasswordSlot wraps the envelope's secret with a passphrase using Argon2id.
func (e *Envelope) AddPasswordSlot(password string, profile kdf.Argon2idProfile, label ...string) error {
	if len(e.secret) == 0 {
		return errors.New("envelope: no secret set to wrap")
	}
	if len(e.Slots) >= 32 {
		return ErrMaxSlotsExceeded
	}

	salt := randkit.MustSecureBytes(32)
	kek := kdf.Argon2id([]byte(password), salt, profile, 32)

	slotIdx := uint8(len(e.Slots))
	nonce := e.deriveSlotNonce(slotIdx)
	aad := slotAAD(slotIdx)

	wrapped, err := aead.Seal(e.CipherAlgo, kek, nonce, e.secret, aad)
	if err != nil {
		return fmt.Errorf("envelope: seal slot: %w", err)
	}

	lbl := ""
	if len(label) > 0 {
		lbl = label[0]
	}

	e.Slots = append(e.Slots, Slot{
		Index:          slotIdx,
		Type:           SlotTypeArgon2id,
		Label:          lbl,
		Salt:           salt,
		KDFMemoryKiB:   profile.MemoryKiB,
		KDFIterations:  profile.Iterations,
		KDFParallelism: profile.Parallelism,
		WrappedSecret:  wrapped,
	})

	return nil
}

// AddKMSSlot wraps the envelope's secret using an envelope KEK encrypted via the provided KMS client.
func (e *Envelope) AddKMSSlot(ctx context.Context, client kms.Client, keyURI string, label ...string) error {
	if len(e.secret) == 0 {
		return errors.New("envelope: no secret set to wrap")
	}
	if len(e.Slots) >= 32 {
		return ErrMaxSlotsExceeded
	}
	if client == nil {
		return errors.New("envelope: nil KMS client")
	}

	kek := randkit.MustSecureBytes(32)
	kmsCT, err := client.Encrypt(ctx, keyURI, kek)
	if err != nil {
		return fmt.Errorf("envelope: kms encrypt kek: %w", err)
	}

	slotIdx := uint8(len(e.Slots))
	nonce := e.deriveSlotNonce(slotIdx)
	aad := slotAAD(slotIdx)

	wrapped, err := aead.Seal(e.CipherAlgo, kek, nonce, e.secret, aad)
	if err != nil {
		return fmt.Errorf("envelope: seal slot: %w", err)
	}

	lbl := ""
	if len(label) > 0 {
		lbl = label[0]
	}

	e.Slots = append(e.Slots, Slot{
		Index:         slotIdx,
		Type:          SlotTypeKMS,
		Label:         lbl,
		KMSKeyURI:     keyURI,
		KMSCiphertext: kmsCT,
		WrappedSecret: wrapped,
	})

	return nil
}

// AddKEKSlot wraps the envelope's secret using a pre-shared 32-byte symmetric key.
func (e *Envelope) AddKEKSlot(kek []byte, label ...string) error {
	if len(e.secret) == 0 {
		return errors.New("envelope: no secret set to wrap")
	}
	if len(e.Slots) >= 32 {
		return ErrMaxSlotsExceeded
	}
	if len(kek) != 32 {
		return aead.ErrInvalidKeyLength
	}

	slotIdx := uint8(len(e.Slots))
	nonce := e.deriveSlotNonce(slotIdx)
	aad := slotAAD(slotIdx)

	wrapped, err := aead.Seal(e.CipherAlgo, kek, nonce, e.secret, aad)
	if err != nil {
		return fmt.Errorf("envelope: seal slot: %w", err)
	}

	lbl := ""
	if len(label) > 0 {
		lbl = label[0]
	}

	e.Slots = append(e.Slots, Slot{
		Index:         slotIdx,
		Type:          SlotTypeRawKEK,
		Label:         lbl,
		WrappedSecret: wrapped,
	})

	return nil
}

// UnlockWithPassword iterates over all password slots and decrypts the envelope secret.
func (e *Envelope) UnlockWithPassword(password string) ([]byte, error) {
	passBytes := []byte(password)

	for _, slot := range e.Slots {
		var kek []byte
		var err error

		switch slot.Type {
		case SlotTypeArgon2id:
			profile := kdf.Argon2idProfile{
				MemoryKiB:   slot.KDFMemoryKiB,
				Iterations:  slot.KDFIterations,
				Parallelism: slot.KDFParallelism,
			}
			kek = kdf.Argon2id(passBytes, slot.Salt, profile, 32)
		case SlotTypePBKDF2:
			kek = kdf.PBKDF2SHA256(passBytes, slot.Salt, int(slot.KDFIterations), 32)
		default:
			continue
		}

		if err != nil {
			continue
		}

		nonce := e.deriveSlotNonce(slot.Index)
		aad := slotAAD(slot.Index)
		pt, err := aead.Open(e.CipherAlgo, kek, nonce, slot.WrappedSecret, aad)
		if err == nil {
			e.secret = append([]byte(nil), pt...)
			return e.Secret(), nil
		}
	}

	return nil, ErrNoMatchingSlot
}

// UnlockWithKMS attempts to decrypt any KMS slot using the provided KMS client.
func (e *Envelope) UnlockWithKMS(ctx context.Context, client kms.Client) ([]byte, error) {
	if client == nil {
		return nil, errors.New("envelope: nil KMS client")
	}

	for _, slot := range e.Slots {
		if slot.Type != SlotTypeKMS || len(slot.KMSCiphertext) == 0 {
			continue
		}

		kek, err := client.Decrypt(ctx, slot.KMSKeyURI, slot.KMSCiphertext)
		if err != nil {
			continue
		}

		nonce := e.deriveSlotNonce(slot.Index)
		aad := slotAAD(slot.Index)
		pt, err := aead.Open(e.CipherAlgo, kek, nonce, slot.WrappedSecret, aad)
		if err == nil {
			e.secret = append([]byte(nil), pt...)
			return e.Secret(), nil
		}
	}

	return nil, ErrNoMatchingSlot
}

// UnlockWithKEK attempts to decrypt any raw KEK slot using the supplied 32-byte key.
func (e *Envelope) UnlockWithKEK(kek []byte) ([]byte, error) {
	if len(kek) != 32 {
		return nil, aead.ErrInvalidKeyLength
	}

	for _, slot := range e.Slots {
		if slot.Type != SlotTypeRawKEK {
			continue
		}

		nonce := e.deriveSlotNonce(slot.Index)
		aad := slotAAD(slot.Index)
		pt, err := aead.Open(e.CipherAlgo, kek, nonce, slot.WrappedSecret, aad)
		if err == nil {
			e.secret = append([]byte(nil), pt...)
			return e.Secret(), nil
		}
	}

	return nil, ErrNoMatchingSlot
}

// WriteBinary writes the compact binary representation of the Envelope to w.
func (e *Envelope) WriteBinary(w io.Writer) error {
	jsonData, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("envelope: marshal json: %w", err)
	}

	var hdr [8]byte
	copy(hdr[0:4], MagicEnvelope[:])
	binary.BigEndian.PutUint32(hdr[4:8], uint32(len(jsonData)))

	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}
	if _, err := w.Write(jsonData); err != nil {
		return err
	}

	// 32-byte checksum of envelope header payload
	checksum := sha256.Sum256(jsonData)
	if _, err := w.Write(checksum[:]); err != nil {
		return err
	}

	return nil
}

// ReadBinary reads and verifies an Envelope from r.
func ReadBinary(r io.Reader) (*Envelope, error) {
	var hdr [8]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, err
	}

	if !bytes.Equal(hdr[0:4], MagicEnvelope[:]) {
		return nil, ErrInvalidEnvelope
	}

	length := binary.BigEndian.Uint32(hdr[4:8])
	if length > 1024*1024 { // 1 MiB max
		return nil, errors.New("envelope: payload too large")
	}

	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}

	var expectedChecksum [32]byte
	if _, err := io.ReadFull(r, expectedChecksum[:]); err != nil {
		return nil, err
	}

	actualChecksum := sha256.Sum256(buf)
	if !bytes.Equal(actualChecksum[:], expectedChecksum[:]) {
		return nil, fmt.Errorf("%w: checksum mismatch", ErrInvalidEnvelope)
	}

	var env Envelope
	if err := json.Unmarshal(buf, &env); err != nil {
		return nil, fmt.Errorf("envelope: parse json: %w", err)
	}

	return &env, nil
}
