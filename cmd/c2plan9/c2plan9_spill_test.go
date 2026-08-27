// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// sampleSpillCKernel uses a local stack buffer forcing Clang to allocate frame space (subq $N, %rsp).
const sampleSpillCKernel = `
#include <stdint.h>
#include <stddef.h>

uint64_t fast_stack_spill_kernel(uint64_t a, uint64_t b, uint64_t c, uint64_t d) {
    volatile uint64_t stack_buf[16];
    stack_buf[0] = a * 3;
    stack_buf[1] = b * 5;
    stack_buf[2] = c * 7;
    stack_buf[3] = d * 11;
    stack_buf[4] = stack_buf[0] ^ stack_buf[1];
    stack_buf[5] = stack_buf[2] ^ stack_buf[3];
    stack_buf[6] = stack_buf[4] + stack_buf[5];
    return stack_buf[6];
}
`

func TestC2Plan9StackSpillEndToEnd(t *testing.T) {
	tmpDir := t.TempDir()
	cFile := filepath.Join(tmpDir, "spill_kernel.c")
	if err := os.WriteFile(cFile, []byte(sampleSpillCKernel), 0o644); err != nil {
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

	sigs := map[string]FuncSignature{
		"fast_stack_spill_kernel": {
			Name:       "fast_stack_spill_kernel",
			GoName:     "FastStackSpillKernel",
			ParamCount: 4,
			HasReturn:  true,
		},
	}

	asmBytes, err := EmitPlan9Assembly("testspill", obj.Symbols, sigs, obj.ROData, obj.Relocations, "amd64")
	if err != nil {
		t.Fatalf("failed to emit Plan 9 assembly: %v", err)
	}

	asmStr := string(asmBytes)
	t.Logf("Generated Plan 9 Assembly with stack frame:\n%s", asmStr)

	// Verify that SUBQ/ADDQ SP was stripped and frame size in TEXT is $88-40
	if strings.Contains(asmStr, "SUBQ $") && strings.Contains(asmStr, ", SP") {
		t.Errorf("assembly should have stripped prologue SUBQ ..., SP, got:\n%s", asmStr)
	}

	if strings.Contains(asmStr, "ADDQ $") && strings.Contains(asmStr, ", SP") {
		t.Errorf("assembly should have stripped epilogue ADDQ ..., SP, got:\n%s", asmStr)
	}

	if !strings.Contains(asmStr, "$88-40") {
		t.Errorf("assembly TEXT declaration should specify $88-40 frame, got:\n%s", asmStr)
	}

	sFile := filepath.Join(tmpDir, "spill_amd64.s")
	if err := os.WriteFile(sFile, asmBytes, 0o644); err != nil {
		t.Fatalf("failed to write assembly file: %v", err)
	}

	stubCode := `//go:build amd64 && !purego

package testspill

//go:noescape
func FastStackSpillKernel(a, b, c, d uint64) uint64
`
	if err := os.WriteFile(filepath.Join(tmpDir, "spill_amd64.go"), []byte(stubCode), 0o644); err != nil {
		t.Fatalf("failed to write stub: %v", err)
	}

	testCode := `//go:build amd64 && !purego

package testspill

import (
	"testing"
)

func TestSpillKernelExecution(t *testing.T) {
	a, b, c, d := uint64(10), uint64(20), uint64(30), uint64(40)
	
	s0 := a * 3
	s1 := b * 5
	s2 := c * 7
	s3 := d * 11
	s4 := s0 ^ s1
	s5 := s2 ^ s3
	expected := s4 + s5

	res := FastStackSpillKernel(a, b, c, d)
	if res != expected {
		t.Fatalf("expected %d, got %d", expected, res)
	}
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "spill_test.go"), []byte(testCode), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	goMod := `module testspill

go 1.25
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	_ = os.Remove(cFile)

	var cmd *exec.Cmd
	if runtime.GOARCH == "amd64" {
		cmd = exec.Command("go", "test", "-v", ".")
	} else {
		cmd = exec.Command("go", "build", "-o", os.DevNull, ".")
		cmd.Env = append(os.Environ(), "GOARCH=amd64")
	}
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go test failed on generated stack spill Plan 9 assembly (%v):\n%s", err, string(out))
	}
	t.Logf("go test on generated stack spill kernel passed:\n%s", string(out))
}
