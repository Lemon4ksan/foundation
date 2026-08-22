// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package cpukit provides unified hardware feature detection and CPU capability probes.
package cpukit

import (
	"unsafe"

	"golang.org/x/sys/cpu"
)

// HasAVX2 reports whether the current x86 CPU supports Intel/AMD AVX2 vector instructions.
func HasAVX2() bool {
	return cpu.X86.HasAVX2
}

// HasBMI1 reports whether the current x86 CPU supports Bit Manipulation Instruction Set 1.
func HasBMI1() bool {
	return cpu.X86.HasBMI1
}

// HasBMI2 reports whether the current x86 CPU supports Bit Manipulation Instruction Set 2.
func HasBMI2() bool {
	return cpu.X86.HasBMI2
}

// HasBMI reports whether the current x86 CPU supports both BMI1 and BMI2 extensions.
func HasBMI() bool {
	return cpu.X86.HasBMI1 && cpu.X86.HasBMI2
}

// HasAVX512 reports whether the current x86 CPU supports AVX-512 Foundation instructions.
func HasAVX512() bool {
	return cpu.X86.HasAVX512F
}

// HasNEON reports whether the current ARM CPU supports Advanced SIMD (NEON).
func HasNEON() bool {
	return cpu.ARM64.HasASIMD
}

// HasAESNI reports whether the current CPU supports hardware-accelerated AES cryptography.
func HasAESNI() bool {
	return cpu.X86.HasAES || cpu.ARM64.HasAES
}

// CacheLineSize returns the CPU cache line size in bytes (typically 64).
func CacheLineSize() int {
	return int(unsafe.Sizeof(cpu.CacheLinePad{}))
}
