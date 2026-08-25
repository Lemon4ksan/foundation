// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const sampleRODataCKernel = `
#include <stdint.h>
#include <stddef.h>

static const uint8_t hex_lookup_table[16] = {
    '0', '1', '2', '3', '4', '5', '6', '7',
    '8', '9', 'a', 'b', 'c', 'd', 'e', 'f'
};

uint64_t fast_hex_encode_byte(uint8_t val) {
    uint8_t hi = hex_lookup_table[(val >> 4) & 0x0F];
    uint8_t lo = hex_lookup_table[val & 0x0F];
    return ((uint64_t)hi << 8) | (uint64_t)lo;
}
`

func TestC2Plan9RODataEndToEnd(t *testing.T) {
	tmpDir := t.TempDir()
	cFile := filepath.Join(tmpDir, "rodata_kernel.c")
	if err := os.WriteFile(cFile, []byte(sampleRODataCKernel), 0o644); err != nil {
		t.Fatalf("failed to write C kernel: %v", err)
	}

	objPath, err := Compile(CompileOptions{
		ClangPath:  "clang",
		SourceFile: cFile,
		TargetArch: "amd64",
		TargetSysV: true,
	})
	if err != nil {
		t.Skipf("clang not available or failed: %v", err)
	}
	defer os.Remove(objPath)

	obj, err := ParseObject(objPath)
	if err != nil {
		t.Fatalf("failed to parse object: %v", err)
	}

	if len(obj.ROData) == 0 {
		t.Fatalf("expected .rodata in object file, got 0 bytes")
	}

	t.Logf("Extracted .rodata size: %d bytes, relocations: %d", len(obj.ROData), len(obj.Relocations))

	sigs := map[string]FuncSignature{
		"fast_hex_encode_byte": {
			Name:       "fast_hex_encode_byte",
			GoName:     "FastHexEncodeByte",
			ParamCount: 1,
			HasReturn:  true,
		},
	}

	asmBytes, err := EmitPlan9Assembly("hexlut", obj.Symbols, sigs, obj.ROData, obj.Relocations, "amd64")
	if err != nil {
		t.Fatalf("failed to emit Plan 9 assembly: %v", err)
	}

	asmStr := string(asmBytes)
	if !strings.Contains(asmStr, "DATA ·rodata<>") {
		t.Errorf("assembly missing DATA ·rodata<> declarations, got:\n%s", asmStr)
	}

	if !strings.Contains(asmStr, "GLOBL ·rodata<>(SB), RODATA") {
		t.Errorf("assembly missing GLOBL ·rodata<> declaration, got:\n%s", asmStr)
	}

	t.Logf("Generated Plan 9 Assembly with .rodata:\n%s", asmStr)

	// Verify that go test compiles and runs with the .rodata constants
	sFile := filepath.Join(tmpDir, "hex_amd64.s")
	if err := os.WriteFile(sFile, asmBytes, 0o644); err != nil {
		t.Fatalf("failed to write assembly file: %v", err)
	}

	stubCode := `//go:build amd64 && !purego

package main

//go:noescape
func FastHexEncodeByte(val uint64) uint64
`
	if err := os.WriteFile(filepath.Join(tmpDir, "hex_amd64.go"), []byte(stubCode), 0o644); err != nil {
		t.Fatalf("failed to write stub: %v", err)
	}

	testCode := `package main

import (
	"testing"
)

func TestHexLUTExecution(t *testing.T) {
	for i := 0; i < 256; i++ {
		res := FastHexEncodeByte(uint64(i))
		hi := byte(res >> 8)
		lo := byte(res & 0xFF)

		const hexChars = "0123456789abcdef"
		expectedHi := hexChars[(i>>4)&0x0F]
		expectedLo := hexChars[i&0x0F]

		if hi != expectedHi || lo != expectedLo {
			t.Fatalf("for byte 0x%02x: got '%c%c', want '%c%c'", i, hi, lo, expectedHi, expectedLo)
		}
	}
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "hex_test.go"), []byte(testCode), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	goMod := `module testhex

go 1.25
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	_ = os.Remove(cFile)

	cmd := exec.Command("go", "test", "-v", ".")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go test failed on generated .rodata Plan 9 assembly (%v):\n%s", err, string(out))
	}
	t.Logf("go test on generated .rodata kernel passed:\n%s", string(out))
}
