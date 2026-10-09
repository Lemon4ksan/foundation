// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package internal_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sync/atomic"
	"testing"
	"unicode/utf8"

	"github.com/lemon4ksan/foundation/text/encoding"
	"github.com/lemon4ksan/foundation/text/encoding/internal"
	"github.com/lemon4ksan/foundation/text/encoding/internal/identifier"
	"github.com/lemon4ksan/foundation/text/transform"
)

func TestEncoding(t *testing.T) {
	enc := &internal.Encoding{
		Encoding: encoding.Nop,
		Name:     "test-enc",
		MIB:      identifier.MIB(1234),
	}

	// Verify identifier.Interface implementation.
	var _ identifier.Interface = enc
	var _ encoding.Encoding = enc
	var _ fmt.Stringer = enc

	if enc.String() != "test-enc" {
		t.Fatalf("enc.String() = %q, want test-enc", enc.String())
	}

	mib, other := enc.ID()
	if mib != 1234 || other != "" {
		t.Fatalf("enc.ID() = (%v, %q), want (1234, \"\")", mib, other)
	}

	// Verify embedded encoding.Encoding delegation.
	dec := enc.NewDecoder()
	if dec == nil || dec.Transformer == nil {
		t.Fatalf("enc.NewDecoder() returned invalid decoder")
	}
	s, err := dec.String("hello")
	if err != nil || s != "hello" {
		t.Fatalf("dec.String(\"hello\") = (%q, %v), want (\"hello\", nil)", s, err)
	}

	encoder := enc.NewEncoder()
	if encoder == nil || encoder.Transformer == nil {
		t.Fatalf("enc.NewEncoder() returned invalid encoder")
	}
	s, err = encoder.String("world")
	if err != nil || s != "world" {
		t.Fatalf("encoder.String(\"world\") = (%q, %v), want (\"world\", nil)", s, err)
	}
}

func TestEncodingZeroAndUnofficial(t *testing.T) {
	enc := &internal.Encoding{
		Name: "",
		MIB:  identifier.Replacement,
	}
	if enc.String() != "" {
		t.Fatalf("enc.String() = %q, want empty", enc.String())
	}
	mib, other := enc.ID()
	if mib != identifier.Replacement || other != "" {
		t.Fatalf("enc.ID() = (%v, %q), want (%v, \"\")", mib, other, identifier.Replacement)
	}
}

// shiftTransformer is a simple test transformer that shifts ASCII bytes by delta.
type shiftTransformer struct {
	delta byte
}

func (s shiftTransformer) Reset() {}

func (s shiftTransformer) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	for nSrc < len(src) {
		if nDst >= len(dst) {
			return nDst, nSrc, transform.ErrShortDst
		}
		dst[nDst] = src[nSrc] + s.delta
		nDst++
		nSrc++
	}
	return nDst, nSrc, nil
}

func TestSimpleEncoding(t *testing.T) {
	// Nop transformers.
	simpleNop := &internal.SimpleEncoding{
		Decoder: transform.Nop,
		Encoder: transform.Nop,
	}

	dec := simpleNop.NewDecoder()
	if dec.Transformer != transform.Nop {
		t.Fatalf("expected transform.Nop decoder")
	}

	enc := simpleNop.NewEncoder()
	if enc.Transformer != transform.Nop {
		t.Fatalf("expected transform.Nop encoder")
	}

	// Custom shift transformers: Encoder adds 1, Decoder subtracts 1.
	simpleShift := &internal.SimpleEncoding{
		Decoder: shiftTransformer{delta: 255}, // -1 mod 256
		Encoder: shiftTransformer{delta: 1},   // +1 mod 256
	}

	raw := "Hello, World!"
	encoded, err := simpleShift.NewEncoder().String(raw)
	if err != nil {
		t.Fatalf("NewEncoder().String failed: %v", err)
	}
	decoded, err := simpleShift.NewDecoder().String(encoded)
	if err != nil {
		t.Fatalf("NewDecoder().String failed: %v", err)
	}
	if decoded != raw {
		t.Fatalf("roundtrip mismatch: got %q, want %q", decoded, raw)
	}

	// Test short destination buffer.
	dst := make([]byte, 2)
	nDst, nSrc, err := simpleShift.NewEncoder().Transform(dst, []byte("Hello"), true)
	if !errors.Is(err, transform.ErrShortDst) {
		t.Fatalf("expected transform.ErrShortDst, got %v", err)
	}
	if nDst != 2 || nSrc != 2 {
		t.Fatalf("unexpected nDst=%d, nSrc=%d", nDst, nSrc)
	}

	// Test streaming Reader and Writer.
	var buf bytes.Buffer
	w := simpleShift.NewEncoder().Writer(&buf)
	if _, err := io.WriteString(w, raw); err != nil {
		t.Fatalf("Writer.WriteString failed: %v", err)
	}
	r := simpleShift.NewDecoder().Reader(&buf)
	roundtrip, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("Reader.ReadAll failed: %v", err)
	}
	if string(roundtrip) != raw {
		t.Fatalf("streaming roundtrip mismatch: got %q, want %q", string(roundtrip), raw)
	}
}

func TestFuncEncoding(t *testing.T) {
	var decCalls, encCalls int64

	fe := internal.FuncEncoding{
		Decoder: func() transform.Transformer {
			atomic.AddInt64(&decCalls, 1)
			return shiftTransformer{delta: 255}
		},
		Encoder: func() transform.Transformer {
			atomic.AddInt64(&encCalls, 1)
			return shiftTransformer{delta: 1}
		},
	}

	// Verify factory function invocation.
	dec1 := fe.NewDecoder()
	if decCalls != 1 {
		t.Fatalf("NewDecoder() call count = %d, want 1", decCalls)
	}
	dec2 := fe.NewDecoder()
	if decCalls != 2 {
		t.Fatalf("NewDecoder() call count = %d, want 2", decCalls)
	}
	if dec1 == dec2 {
		t.Fatalf("expected separate decoder instances")
	}

	enc1 := fe.NewEncoder()
	if encCalls != 1 {
		t.Fatalf("NewEncoder() call count = %d, want 1", encCalls)
	}
	enc2 := fe.NewEncoder()
	if encCalls != 2 {
		t.Fatalf("NewEncoder() call count = %d, want 2", encCalls)
	}
	if enc1 == enc2 {
		t.Fatalf("expected separate encoder instances")
	}

	// Verify transform execution and roundtrip.
	input := "Foundation Go Framework"
	encodedBytes, err := enc1.Bytes([]byte(input))
	if err != nil {
		t.Fatalf("enc1.Bytes failed: %v", err)
	}
	decodedBytes, err := dec1.Bytes(encodedBytes)
	if err != nil {
		t.Fatalf("dec1.Bytes failed: %v", err)
	}
	if string(decodedBytes) != input {
		t.Fatalf("FuncEncoding roundtrip mismatch: got %q, want %q", string(decodedBytes), input)
	}
}

func TestRepertoireError(t *testing.T) {
	tests := []struct {
		name        string
		err         internal.RepertoireError
		wantRepl    byte
		wantErrText string
	}{
		{
			name:        "ASCII Sub",
			err:         internal.RepertoireError(encoding.ASCIISub),
			wantRepl:    0x1a,
			wantErrText: "encoding: rune not supported by encoding.",
		},
		{
			name:        "Question mark replacement",
			err:         internal.RepertoireError('?'),
			wantRepl:    '?',
			wantErrText: "encoding: rune not supported by encoding.",
		},
		{
			name:        "Zero replacement",
			err:         internal.RepertoireError(0),
			wantRepl:    0,
			wantErrText: "encoding: rune not supported by encoding.",
		},
		{
			name:        "0xFF replacement",
			err:         internal.RepertoireError(0xff),
			wantRepl:    0xff,
			wantErrText: "encoding: rune not supported by encoding.",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var err error = tc.err
			if err.Error() != tc.wantErrText {
				t.Fatalf("err.Error() = %q, want %q", err.Error(), tc.wantErrText)
			}
			if tc.err.Replacement() != tc.wantRepl {
				t.Fatalf("err.Replacement() = %v, want %v", tc.err.Replacement(), tc.wantRepl)
			}
		})
	}

	// Test ErrASCIIReplacement package variable.
	if internal.ErrASCIIReplacement != internal.RepertoireError(encoding.ASCIISub) {
		t.Fatalf("ErrASCIIReplacement = %v, want %v", internal.ErrASCIIReplacement, encoding.ASCIISub)
	}
	if internal.ErrASCIIReplacement.Replacement() != 0x1a {
		t.Fatalf("ErrASCIIReplacement.Replacement() = %#x, want 0x1a", internal.ErrASCIIReplacement.Replacement())
	}
	if internal.ErrASCIIReplacement.Error() != "encoding: rune not supported by encoding." {
		t.Fatalf("unexpected ErrASCIIReplacement.Error(): %q", internal.ErrASCIIReplacement.Error())
	}
}

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
	for b := range 256 {
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
