// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package internal_test

import (
	"fmt"
	"regexp"
	"testing"
	"unicode/utf8"

	"github.com/lemon4ksan/foundation/text/encoding"
	"github.com/lemon4ksan/foundation/text/encoding/internal"
	"github.com/lemon4ksan/foundation/text/encoding/internal/identifier"
	"github.com/lemon4ksan/foundation/text/transform"
)

// TestAdversarialBoundaryRunes tests handling of extreme unicode and boundary runes.
func TestAdversarialBoundaryRunes(t *testing.T) {
	boundaryRunes := []rune{
		0,              // NUL
		1,              // SOH
		0x7F,           // DEL
		0x80,           // First non-ASCII
		0x07FF,         // 2-byte boundary
		0x0800,         // 3-byte boundary
		0xD7FF,         // Just before surrogate pairs
		0xD800,         // Surrogate start
		0xDBFF,         // High surrogate end
		0xDC00,         // Low surrogate start
		0xDFFF,         // Surrogate end
		0xE000,         // Private use area start
		0xFFFF,         // BMP limit
		0x10000,        // First supplementary plane
		0x10FFFF,       // Maximum valid Unicode rune
		0x110000,       // Invalid rune beyond Unicode
		utf8.RuneError, // Unicode replacement char (0xFFFD)
		-1,             // Negative rune
	}

	for _, r := range boundaryRunes {
		t.Run(fmt.Sprintf("Rune_%x", r), func(t *testing.T) {
			buf := make([]byte, 4)
			n := utf8.EncodeRune(buf, r)
			valid := utf8.ValidRune(r)

			enc := &internal.Encoding{
				Encoding: encoding.Nop,
				Name:     fmt.Sprintf("enc-rune-%x", r),
				MIB:      identifier.MIB(100),
			}

			dec := enc.NewDecoder()
			res, err := dec.Bytes(buf[:n])
			if err != nil {
				t.Fatalf("unexpected error decoding rune %x: %v", r, err)
			}
			if len(res) != n {
				t.Fatalf("length mismatch: got %d, want %d", len(res), n)
			}

			// Validate RepertoireError with boundary byte
			repErr := internal.RepertoireError(byte(r & 0xFF))
			if repErr.Error() != "encoding: rune not supported by encoding." {
				t.Fatalf("unexpected error string: %s", repErr.Error())
			}
			if repErr.Replacement() != byte(r&0xFF) {
				t.Fatalf("replacement byte mismatch: got %x, want %x", repErr.Replacement(), byte(r&0xFF))
			}
			_ = valid
		})
	}
}

// TestAdversarialRepertoireErrorAllBytes exhaustively tests all 256 byte values for RepertoireError.
func TestAdversarialRepertoireErrorAllBytes(t *testing.T) {
	for b := 0; b < 256; b++ {
		err := internal.RepertoireError(byte(b))
		if err.Replacement() != byte(b) {
			t.Fatalf("byte %d: Replacement() = %d, want %d", b, err.Replacement(), b)
		}
		if err.Error() != "encoding: rune not supported by encoding." {
			t.Fatalf("byte %d: unexpected Error() string: %q", b, err.Error())
		}
	}

	if internal.ErrASCIIReplacement.Replacement() != encoding.ASCIISub {
		t.Fatalf("ErrASCIIReplacement = %x, want %x", internal.ErrASCIIReplacement.Replacement(), encoding.ASCIISub)
	}
}

// validOtherRegex conforms to identifier.Interface documentation:
// "The other string may only contain the characters a-z, A-Z, 0-9, - and _."
var validOtherRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

type mockID struct {
	mib   identifier.MIB
	other string
}

func (m mockID) ID() (identifier.MIB, string) {
	return m.mib, m.other
}

// TestAdversarialMockIdentifiers tests invalid and conforming mock identifier contracts.
func TestAdversarialMockIdentifiers(t *testing.T) {
	testCases := []struct {
		name          string
		mib           identifier.MIB
		other         string
		validContract bool
	}{
		{"ValidMIBOnly", identifier.MIB(106), "", true},
		{"ValidOtherOnly", identifier.MIB(0), "x-mac-dingbat", true},
		{"ValidOtherUnderscore", identifier.MIB(0), "x_custom-1", true},
		{"InvalidBothZero", identifier.MIB(0), "", false},
		{"InvalidBothNonZero", identifier.MIB(106), "utf-8", false},
		{"InvalidOtherSpaces", identifier.MIB(0), "x mac dingbat", false},
		{"InvalidOtherNull", identifier.MIB(0), "x-mac\x00dingbat", false},
		{"InvalidOtherSymbols", identifier.MIB(0), "x$mac@dingbat!", false},
		{"InvalidOtherSlash", identifier.MIB(0), "iso/8859/1", false},
		{"UnofficialRange", identifier.Unofficial, "", true},
		{"ReplacementEncoding", identifier.Replacement, "", true},
		{"MaxMIB", identifier.MIB(0xFFFF), "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := mockID{mib: tc.mib, other: tc.other}
			mib, other := m.ID()

			// Check contract: "Exactly one of the mib and other values should be non-zero."
			hasMIB := mib != 0
			hasOther := other != ""
			exactlyOneNonZero := (hasMIB || hasOther) && !(hasMIB && hasOther)

			otherCharsValid := true
			if hasOther && !validOtherRegex.MatchString(other) {
				otherCharsValid = false
			}

			isValid := exactlyOneNonZero && otherCharsValid
			if isValid != tc.validContract {
				t.Fatalf("%s: contract validity mismatch: got %v, want %v (mib=%v, other=%q)",
					tc.name, isValid, tc.validContract, mib, other)
			}

			// Wrap in internal.Encoding and verify safety
			enc := &internal.Encoding{
				Encoding: encoding.Nop,
				Name:     tc.name,
				MIB:      tc.mib,
			}
			if enc.String() != tc.name {
				t.Fatalf("enc.String() = %q, want %q", enc.String(), tc.name)
			}
			dec := enc.NewDecoder()
			if dec == nil {
				t.Fatalf("NewDecoder was nil")
			}
		})
	}
}

// TestAdversarialTransformersWithShortBuffers verifies transform bounds.
func TestAdversarialTransformersWithShortBuffers(t *testing.T) {
	trans := &internal.SimpleEncoding{
		Decoder: transform.Nop,
		Encoder: transform.Nop,
	}

	dec := trans.NewDecoder()
	dst := make([]byte, 2)
	src := []byte("hello")
	nDst, nSrc, err := dec.Transform(dst, src, false)
	if err != transform.ErrShortDst {
		t.Fatalf("expected ErrShortDst, got: %v (nDst=%d, nSrc=%d)", err, nDst, nSrc)
	}
}
