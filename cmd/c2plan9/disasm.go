// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"strings"

	"golang.org/x/arch/x86/x86asm"
)

// DisassembledInst represents a single disassembled machine instruction.
type DisassembledInst struct {
	Offset   uint64
	Length   int
	RawBytes []byte
	Text     string
	Label    string // non-empty if a label should be placed before this instruction
	IsRet    bool
}

// DisassembleAMD64 disassembles raw x86-64 machine code into Plan 9 assembly lines.
func DisassembleAMD64(code []byte, baseOffset uint64) ([]DisassembledInst, error) {
	var (
		insts       []DisassembledInst
		jumpTargets = make(map[uint64]string)
		offset      = 0
		n           = len(code)
	)

	// Pass 1: Discover all branch targets and assign clean label names
	for offset < n {
		inst, err := x86asm.Decode(code[offset:], 64)
		if err != nil {
			// Single unrecognized byte
			offset++
			continue
		}

		if offset+3 <= n && code[offset] == 0xC5 && code[offset+1] == 0xF8 && code[offset+2] == 0x77 {
			inst.Len = 3
		}

		pc := uint64(offset)
		for _, arg := range inst.Args {
			if rel, ok := arg.(x86asm.Rel); ok {
				target := uint64(int64(pc) + int64(inst.Len) + int64(rel))
				if target <= uint64(n) && target > 0 {
					if _, exists := jumpTargets[target]; !exists {
						jumpTargets[target] = fmt.Sprintf("L_%04x", target)
					}
				}
			}
		}

		offset += inst.Len
	}

	// SymLookup callback for Plan9Syntax to resolve local labels
	symLookup := func(addr uint64) (string, uint64) {
		if label, ok := jumpTargets[addr]; ok {
			return label, addr
		}
		return "", 0
	}

	// Pass 2: Generate Plan 9 syntax for each instruction
	offset = 0
	for offset < n {
		pc := uint64(offset)
		inst, err := x86asm.Decode(code[offset:], 64)
		if err != nil {
			// Emit as raw byte
			raw := code[offset : offset+1]
			insts = append(insts, DisassembledInst{
				Offset:   pc,
				Length:   1,
				RawBytes: raw,
				Text:     fmt.Sprintf("BYTE $0x%02x", raw[0]),
				Label:    jumpTargets[pc],
			})
			offset++
			continue
		}

		// Fix x86asm decoder bug: VZEROUPPER (C5 F8 77) has no ModR/M byte (length is always 3)
		if offset+3 <= n && code[offset] == 0xC5 && code[offset+1] == 0xF8 && code[offset+2] == 0x77 {
			inst.Len = 3
		}

		raw := code[offset : offset+inst.Len]
		text := x86asm.GoSyntax(inst, pc, symLookup)
		text = cleanPlan9Syntax(text)

		insts = append(insts, DisassembledInst{
			Offset:   pc,
			Length:   inst.Len,
			RawBytes: raw,
			Text:     text,
			Label:    jumpTargets[pc],
			IsRet:    inst.Op == x86asm.RET,
		})

		offset += inst.Len
	}

	return insts, nil
}

// cleanPlan9Syntax standardizes register and instruction syntax for Go's assembler.
func cleanPlan9Syntax(s string) string {
	s = strings.TrimSpace(s)

	// Replace known register names if necessary
	replacements := []struct {
		old string
		new string
	}{
		{"%rax", "AX"}, {"%rbx", "BX"}, {"%rcx", "CX"}, {"%rdx", "DX"},
		{"%rsi", "SI"}, {"%rdi", "DI"}, {"%rbp", "BP"}, {"%rsp", "SP"},
		{"%r8", "R8"}, {"%r9", "R9"}, {"%r10", "R10"}, {"%r11", "R11"},
		{"%r12", "R12"}, {"%r13", "R13"}, {"%r14", "R14"}, {"%r15", "R15"},
	}

	for _, r := range replacements {
		s = strings.ReplaceAll(s, r.old, r.new)
	}

	// Strip (SB) from local jump targets (e.g. JE L_0014(SB) -> JE L_0014)
	if strings.Contains(s, "(SB)") && strings.Contains(s, "L_") {
		s = strings.ReplaceAll(s, "(SB)", "")
	}

	// Replace CS NOP / prefix NOP with standard NOP
	if strings.Contains(s, "NOP") {
		s = "NOP"
	}

	// Fix 8-bit register suffixes (e.g. TESTL $12, DL -> TESTB $12, DL)
	eightBitRegs := []string{"AL", "BL", "CL", "DL", "SIL", "DIL", "BPL", "SPL", "R8B", "R9B", "R10B", "R11B", "R12B", "R13B", "R14B", "R15B"}
	for _, reg := range eightBitRegs {
		if strings.HasSuffix(s, ", "+reg) || strings.HasSuffix(s, " "+reg) {
			if strings.HasPrefix(s, "TESTL ") {
				s = "TESTB " + s[6:]
			} else if strings.HasPrefix(s, "CMPL ") {
				s = "CMPB " + s[5:]
			} else if strings.HasPrefix(s, "MOVL ") {
				s = "MOVB " + s[5:]
			} else if strings.HasPrefix(s, "ANDL ") {
				s = "ANDB " + s[5:]
			} else if strings.HasPrefix(s, "ORL ") {
				s = "ORB " + s[4:]
			} else if strings.HasPrefix(s, "XORL ") {
				s = "XORB " + s[5:]
			}
		}
	}

	// Mnemonic translations for Go's Plan 9 assembler
	if strings.HasPrefix(s, "MOVZX ") {
		s = "MOVBLZX " + s[6:]
	} else if strings.HasPrefix(s, "MOVSX ") {
		s = "MOVBLSX " + s[6:]
	} else if strings.HasPrefix(s, "MOVSXD ") {
		s = "MOVLQSX " + s[7:]
	} else if strings.HasPrefix(s, "CMOVAE ") {
		s = "CMOVLCC " + s[7:]
	} else if strings.HasPrefix(s, "CMOVB ") {
		s = "CMOVLCS " + s[6:]
	} else if strings.HasPrefix(s, "CMOVE ") {
		s = "CMOVLEQ " + s[6:]
	} else if strings.HasPrefix(s, "CMOVNE ") {
		s = "CMOVLNE " + s[7:]
	} else if strings.HasPrefix(s, "CMOVG ") {
		s = "CMOVLGT " + s[6:]
	} else if strings.HasPrefix(s, "CMOVGE ") {
		s = "CMOVLGE " + s[7:]
	} else if strings.HasPrefix(s, "CMOVL ") {
		s = "CMOVLLT " + s[6:]
	} else if strings.HasPrefix(s, "CMOVLE ") {
		s = "CMOVLLE " + s[7:]
	} else if strings.HasPrefix(s, "TZCNT ") {
		s = "TZCNTL " + s[6:]
	} else if strings.HasPrefix(s, "BSF ") {
		s = "BSFL " + s[4:]
	} else if strings.HasPrefix(s, "BSR ") {
		s = "BSRL " + s[4:]
	} else if strings.HasPrefix(s, "LZCNT ") {
		s = "LZCNTL " + s[6:]
	} else if strings.HasPrefix(s, "POPCNT ") {
		s = "POPCNTL " + s[7:]
	}

	return s
}
