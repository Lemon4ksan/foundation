// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package internal_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"testing"

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
