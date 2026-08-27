// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"testing"
)

func FuzzDisassembleARM64Raw(f *testing.F) {
	// Sample arm64 instructions: MOV, ADD, SUB, RET
	f.Add([]byte("\x00\x00\x80\xd2\x20\x00\x00\x8b\xc0\x03\x5f\xd6"))
	f.Add([]byte(""))
	f.Add([]byte("\xff\xff\xff\xff\x00\x00\x00\x00"))

	f.Fuzz(func(t *testing.T, code []byte) {
		_, _ = DisassembleARM64(code, 0, nil)
	})
}

func FuzzCleanPlan9ARM64Syntax(f *testing.F) {
	f.Add("LDURBW -1(R11), R13")
	f.Add("STURBW R12, -1(R10)")
	f.Add("VMAXV V0.B16, B1")
	f.Add("ADD $16, SP, SP")
	f.Add("ISB $15")
	f.Add("RET")
	f.Add("")

	f.Fuzz(func(t *testing.T, line string) {
		_ = cleanPlan9ARM64Syntax(line)
	})
}
