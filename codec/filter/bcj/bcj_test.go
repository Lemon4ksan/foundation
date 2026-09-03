// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bcj

import (
	"bytes"
	"io"
	"testing"
)

func TestBCJ_Roundtrip(t *testing.T) {
	archs := []Architecture{X86, ARM, ARMT, ARM64, PPC, IA64, SPARC}

	// Create test payload with synthetic call/jump patterns
	original := make([]byte, 1024)
	for i := range original {
		original[i] = byte((i * 37) & 0xFF)
	}
	// Inject x86 E8/E9 CALL/JMP opcodes
	original[10] = 0xE8
	original[11] = 0x12
	original[12] = 0x34
	original[13] = 0x56
	original[14] = 0x00

	original[100] = 0xE9
	original[101] = 0xFE
	original[102] = 0xDC
	original[103] = 0xBA
	original[104] = 0x00

	for _, arch := range archs {
		t.Run(t.Name(), func(t *testing.T) {
			buf := make([]byte, len(original))
			copy(buf, original)

			// Encode
			Filter(arch, buf, 0, true)

			// Decode
			Filter(arch, buf, 0, false)

			if !bytes.Equal(buf, original) {
				t.Fatalf("arch %v: roundtrip mismatch", arch)
			}
		})
	}
}

func TestBCJ_Streaming(t *testing.T) {
	original := make([]byte, 2048)
	for i := range original {
		original[i] = byte((i*53 + 7) & 0xFF)
	}

	var encoded bytes.Buffer
	writer := NewWriter(&encoded, X86)
	n, err := writer.Write(original)
	if err != nil || n != len(original) {
		t.Fatalf("Write failed: n=%d, err=%v", n, err)
	}

	reader := NewReader(&encoded, X86)
	decoded, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if !bytes.Equal(decoded, original) {
		t.Fatalf("Streaming roundtrip mismatch")
	}
}
