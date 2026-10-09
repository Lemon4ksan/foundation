// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"slices"
	"strings"
	"testing"
)

func TestReturnType_Properties(t *testing.T) {
	if TypeFloat32.Kind() != RetFloat32 || TypeFloat32.Size() != 4 || TypeFloat32.GoType() != "float32" {
		t.Errorf("unexpected TypeFloat32 properties")
	}
	if TypeFloat64.Kind() != RetFloat64 || TypeFloat64.Size() != 8 || TypeFloat64.GoType() != "float64" {
		t.Errorf("unexpected TypeFloat64 properties")
	}
	if TypeVector128.Kind() != RetVector128 || TypeVector128.Size() != 16 || TypeVector128.GoType() != "[16]byte" {
		t.Errorf("unexpected TypeVector128 properties")
	}
	if TypeVector256.Kind() != RetVector256 || TypeVector256.Size() != 32 || TypeVector256.GoType() != "[32]byte" {
		t.Errorf("unexpected TypeVector256 properties")
	}
	if TypeInt.Kind() != RetInt64 || TypeInt.Size() != 8 || TypeInt.GoType() != "int" {
		t.Errorf("unexpected TypeInt properties")
	}

	sigs := []FuncSignature{
		{Name: "fn_f32", GoName: "FnF32", ParamCount: 1, HasReturn: true, ReturnType: TypeFloat32},
		{Name: "fn_v256", GoName: "FnV256", ParamCount: 2, HasReturn: true, ReturnType: TypeVector256},
	}
	stubs := string(EmitGoStub("pkg", sigs, "amd64"))
	if !strings.Contains(stubs, "func FnF32(arg0 uint64) float32") {
		t.Errorf("missing float32 stub")
	}
	if !strings.Contains(stubs, "func FnV256(arg0 uint64, arg1 uint64) [32]byte") {
		t.Errorf("missing vector256 stub")
	}
}

func TestBuildClangArgs_TargetTriple(t *testing.T) {
	optsAmd64 := CompileOptions{SourceFile: "foo.c", TargetArch: "amd64", ExtraFlags: []string{"-DFOO"}}
	argsAmd64 := BuildClangArgs(optsAmd64, "foo.o")
	if !slices.Contains(argsAmd64, "-target") || !slices.Contains(argsAmd64, "x86_64-pc-linux-gnu") {
		t.Errorf("amd64 args missing x86_64-pc-linux-gnu: %v", argsAmd64)
	}
	if !slices.Contains(argsAmd64, "-mavx2") || !slices.Contains(argsAmd64, "-DFOO") {
		t.Errorf("amd64 args missing flags: %v", argsAmd64)
	}

	optsArm64 := CompileOptions{SourceFile: "bar.c", TargetArch: "arm64"}
	argsArm64 := BuildClangArgs(optsArm64, "bar.o")
	if !slices.Contains(argsArm64, "-target") || !slices.Contains(argsArm64, "aarch64-linux-gnu") {
		t.Errorf("arm64 args missing aarch64-linux-gnu: %v", argsArm64)
	}
}

func TestDisassembleAMD64_StackAlignment(t *testing.T) {
	// Machine bytes for: ANDQ $-32, SP (48 83 e4 e0) followed by RET (c3)
	code := []byte{0x48, 0x83, 0xe4, 0xe0, 0xc3}
	res, err := DisassembleAMD64(code, 0, nil)
	if err != nil {
		t.Fatalf("DisassembleAMD64 error: %v", err)
	}
	if res.FrameSize != 32 {
		t.Errorf("expected FrameSize >= 32, got %d", res.FrameSize)
	}
	foundAlign := false
	for _, inst := range res.Insts {
		if strings.Contains(inst.Text, "ANDQ $-32, SP") || strings.Contains(inst.Text, "AND") {
			foundAlign = true
			break
		}
	}
	if !foundAlign {
		t.Errorf("expected stack alignment instruction preserved, got %v", res.Insts)
	}
}
