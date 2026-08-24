// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build amd64 && !purego

#include "textflag.h"

// func procYield(cycles uint32)
TEXT ·procYield(SB), NOSPLIT, $0-4
	MOVL cycles+0(FP), AX
loop:
	PAUSE
	SUBL $1, AX
	JNZ loop
	RET
