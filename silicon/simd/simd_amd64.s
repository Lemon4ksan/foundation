// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build amd64 && !purego

#include "textflag.h"

// func applyFastMaskAVX2(b []byte, mask uint32)
TEXT ·applyFastMaskAVX2(SB), NOSPLIT, $0-28
	MOVQ b_base+0(FP), AX
	MOVQ b_len+8(FP), BX

	CMPQ BX, $32
	JL mask_fallback

	// Broadcast byte constants across 256-bit YMM registers:
	// Y0: '\r' (0x0D)
	// Y1: '\n' (0x0A)
	// Y2: ' '  (0x20)
	MOVL $0x0D, CX
	MOVD CX, X0
	VPBROADCASTB X0, Y0

	MOVL $0x0A, CX
	MOVD CX, X1
	VPBROADCASTB X1, Y1

	MOVL $0x20, CX
	MOVD CX, X2
	VPBROADCASTB X2, Y2

mask_loop32:
	VMOVDQU (AX), Y3
	VPCMPEQB Y0, Y3, Y4
	VPCMPEQB Y1, Y3, Y5
	VPOR Y4, Y5, Y4
	VPBLENDVB Y4, Y2, Y3, Y3
	VMOVDQU Y3, (AX)

	ADDQ $32, AX
	SUBQ $32, BX
	CMPQ BX, $32
	JGE mask_loop32

mask_fallback:
	VZEROUPPER
	RET

// func pext64(val, mask uint64) uint64
TEXT ·pext64(SB), NOSPLIT, $0-24
	MOVQ val+0(FP), AX
	MOVQ mask+8(FP), CX
	PEXTQ CX, AX, DX
	MOVQ DX, ret+16(FP)
	RET

// func pdep64(val, mask uint64) uint64
TEXT ·pdep64(SB), NOSPLIT, $0-24
	MOVQ val+0(FP), AX
	MOVQ mask+8(FP), CX
	PDEPQ CX, AX, DX
	MOVQ DX, ret+16(FP)
	RET

// func prefetchL1(ptr unsafe.Pointer)
TEXT ·prefetchL1(SB), NOSPLIT, $0-8
	MOVQ ptr+0(FP), AX
	PREFETCHT0 (AX)
	RET

// func streamCopy256(dst, src []byte)
TEXT ·streamCopy256(SB), NOSPLIT, $0-48
	MOVQ dst_base+0(FP), AX
	MOVQ src_base+24(FP), BX
	MOVQ src_len+32(FP), CX

	CMPQ CX, $32
	JL stream_fallback

stream_loop32:
	VMOVDQU (BX), Y0
	VMOVNTDQ Y0, (AX)

	ADDQ $32, AX
	ADDQ $32, BX
	SUBQ $32, CX
	CMPQ CX, $32
	JGE stream_loop32

	SFENCE

stream_fallback:
	VZEROUPPER
	RET
