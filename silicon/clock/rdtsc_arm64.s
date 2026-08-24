// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build arm64 && !purego

#include "textflag.h"

// func rdtsc() uint64
TEXT ·rdtsc(SB), NOSPLIT, $0-8
	ISB
	MRS CNTVCT_EL0, R0
	MOVD R0, ret+0(FP)
	RET
