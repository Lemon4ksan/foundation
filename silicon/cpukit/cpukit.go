// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cpukit

import (
	"unsafe"

	"golang.org/x/sys/cpu"
)

var (
	hasAVX2   = cpu.X86.HasAVX2
	hasBMI1   = cpu.X86.HasBMI1
	hasBMI2   = cpu.X86.HasBMI2
	hasBMI    = cpu.X86.HasBMI1 && cpu.X86.HasBMI2
	hasAVX512 = cpu.X86.HasAVX512F
	hasNEON   = cpu.ARM64.HasASIMD
	hasAESNI  = cpu.X86.HasAES || cpu.ARM64.HasAES
)

const cacheLineSize = int(unsafe.Sizeof(cpu.CacheLinePad{}))

// HasAVX2 reports whether the current x86 CPU supports Intel/AMD AVX2 vector instructions.
//
//go:inline
func HasAVX2() bool {
	return hasAVX2
}

// HasBMI1 reports whether the current x86 CPU supports Bit Manipulation Instruction Set 1.
//
//go:inline
func HasBMI1() bool {
	return hasBMI1
}

// HasBMI2 reports whether the current x86 CPU supports Bit Manipulation Instruction Set 2.
//
//go:inline
func HasBMI2() bool {
	return hasBMI2
}

// HasBMI reports whether the current x86 CPU supports both BMI1 and BMI2 extensions.
//
//go:inline
func HasBMI() bool {
	return hasBMI
}

// HasAVX512 reports whether the current x86 CPU supports AVX-512 Foundation instructions.
//
//go:inline
func HasAVX512() bool {
	return hasAVX512
}

// HasNEON reports whether the current ARM CPU supports Advanced SIMD (NEON).
//
//go:inline
func HasNEON() bool {
	return hasNEON
}

// HasAESNI reports whether the current CPU supports hardware-accelerated AES cryptography.
//
//go:inline
func HasAESNI() bool {
	return hasAESNI
}

// CacheLineSize returns the CPU cache line size in bytes (typically 64).
//
//go:inline
func CacheLineSize() int {
	return cacheLineSize
}

// HasGFNI reports whether the CPU supports Galois Field New Instructions (GFNI).
//
//go:inline
func HasGFNI() bool {
	return hasGFNI
}
