// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package shamir implements Shamir's Secret Sharing (M-of-N threshold scheme)
// over Galois Field GF(2^8).
package shamir

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"hash/crc32"
	"strconv"
	"strings"
)

var (
	ErrInvalidSecret    = errors.New("shamir: secret cannot be empty")
	ErrInvalidThreshold = errors.New("shamir: threshold must be >= 2 and <= total")
	ErrInvalidTotal     = errors.New("shamir: total shares must be <= 255")
	ErrInsufficient     = errors.New("shamir: at least 2 shares required to reconstruct")
	ErrLengthMismatch   = errors.New("shamir: all shares must have equal length")
	ErrDuplicateShare   = errors.New("shamir: duplicate share index detected")
	ErrInvalidShareStr  = errors.New("shamir: invalid share string format")
	ErrChecksumFailed   = errors.New("shamir: share checksum verification failed")
)

// Share represents a single Shamir polynomial share.
type Share struct {
	Index byte   // Polynomial x-coordinate (1..255)
	Data  []byte // Evaluated y-coordinates for each byte of the secret
}

// Split splits a secret into total shares with a threshold required for reconstruction.
func Split(secret []byte, threshold, total int) ([]Share, error) {
	if len(secret) == 0 {
		return nil, ErrInvalidSecret
	}
	if threshold < 2 || threshold > total {
		return nil, ErrInvalidThreshold
	}
	if total > 255 {
		return nil, ErrInvalidTotal
	}

	shares := make([]Share, total)
	for i := 0; i < total; i++ {
		shares[i] = Share{
			Index: byte(i + 1),
			Data:  make([]byte, len(secret)),
		}
	}

	coeffs := make([]byte, threshold)
	for byteIdx, secretByte := range secret {
		coeffs[0] = secretByte
		if _, err := rand.Read(coeffs[1:]); err != nil {
			return nil, fmt.Errorf("shamir: generate random coefficients: %w", err)
		}

		for i := 0; i < total; i++ {
			x := shares[i].Index
			shares[i].Data[byteIdx] = evalPoly(coeffs, x)
		}
	}

	return shares, nil
}

// Combine reconstructs the secret from a set of threshold shares.
func Combine(shares []Share) ([]byte, error) {
	if len(shares) < 2 {
		return nil, ErrInsufficient
	}

	secretLen := len(shares[0].Data)
	if secretLen == 0 {
		return nil, ErrInvalidSecret
	}

	xs := make([]byte, len(shares))
	seen := make(map[byte]bool)

	for i, s := range shares {
		if len(s.Data) != secretLen {
			return nil, ErrLengthMismatch
		}
		if s.Index == 0 {
			return nil, errors.New("shamir: share index cannot be zero")
		}
		if seen[s.Index] {
			return nil, ErrDuplicateShare
		}
		seen[s.Index] = true
		xs[i] = s.Index
	}

	secret := make([]byte, secretLen)
	ys := make([]byte, len(shares))

	for byteIdx := 0; byteIdx < secretLen; byteIdx++ {
		for i := range shares {
			ys[i] = shares[i].Data[byteIdx]
		}
		secret[byteIdx] = lagrange0(xs, ys)
	}

	return secret, nil
}

// FormatShare formats a share into a standardized, human-readable string with CRC32 checksum.
// Format: aeon-share-m{threshold}n{total}-{index:02x}-{hexData}-{crc32:08x}
func FormatShare(s Share, threshold, total int) string {
	hexData := hex.EncodeToString(s.Data)
	prefix := fmt.Sprintf("aeon-share-m%dn%d-%02x-%s", threshold, total, s.Index, hexData)
	crc := crc32.ChecksumIEEE([]byte(prefix))
	return fmt.Sprintf("%s-%08x", prefix, crc)
}

// ParseShare parses and verifies a formatted share string.
func ParseShare(str string) (s Share, threshold, total int, err error) {
	str = strings.TrimSpace(str)
	parts := strings.Split(str, "-")
	if len(parts) != 6 || parts[0] != "aeon" || parts[1] != "share" {
		return s, 0, 0, ErrInvalidShareStr
	}

	// Verify CRC32
	prefix := strings.Join(parts[:5], "-")
	expectedCRC, err := strconv.ParseUint(parts[5], 16, 32)
	if err != nil {
		return s, 0, 0, ErrInvalidShareStr
	}
	if crc32.ChecksumIEEE([]byte(prefix)) != uint32(expectedCRC) {
		return s, 0, 0, ErrChecksumFailed
	}

	// Parse m{threshold}n{total}
	mn := parts[2]
	if !strings.HasPrefix(mn, "m") || !strings.Contains(mn, "n") {
		return s, 0, 0, ErrInvalidShareStr
	}
	mnParts := strings.Split(strings.TrimPrefix(mn, "m"), "n")
	if len(mnParts) != 2 {
		return s, 0, 0, ErrInvalidShareStr
	}
	threshold, err = strconv.Atoi(mnParts[0])
	if err != nil {
		return s, 0, 0, ErrInvalidShareStr
	}
	total, err = strconv.Atoi(mnParts[1])
	if err != nil {
		return s, 0, 0, ErrInvalidShareStr
	}

	// Parse index
	idx64, err := strconv.ParseUint(parts[3], 16, 8)
	if err != nil || idx64 == 0 {
		return s, 0, 0, ErrInvalidShareStr
	}

	// Parse data
	data, err := hex.DecodeString(parts[4])
	if err != nil {
		return s, 0, 0, ErrInvalidShareStr
	}

	s = Share{
		Index: byte(idx64),
		Data:  data,
	}

	return s, threshold, total, nil
}
