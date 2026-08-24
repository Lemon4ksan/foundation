// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build arm64 && !purego

#include "textflag.h"

// func procYield(cycles uint32)
TEXT ·procYield(SB), NOSPLIT, $0-4
	MOVW cycles+0(FP), R0
loop:
	YIELD
	SUB $1, R0, R0
	CBNZ R0, loop
	RET
