// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build amd64 && !purego

package cpukit

import "golang.org/x/sys/cpu"

// cpuid executes the CPUID instruction for the specified leaf (eaxArg) and sub-leaf (ecxArg).
// Returns the values of eax, ebx, ecx, and edx.
//
//go:noescape
func cpuid(eaxArg, ecxArg uint32) (eax, ebx, ecx, edx uint32)

var hasGFNI = detectGFNI()

func detectGFNI() bool {
	// CPUID leaf 7, sub-leaf 0, ECX bit 8 (1 << 8) is the standard Intel/AMD GFNI feature flag.
	const cpuidGFNI = 1 << 8

	maxID, _, _, _ := cpuid(0, 0)
	if maxID >= 7 {
		_, _, ecx, _ := cpuid(7, 0)
		if (ecx & cpuidGFNI) != 0 {
			return true
		}
	}

	// Also check if AVX-512 GFNI was detected by x/sys/cpu.
	return cpu.X86.HasAVX512GFNI
}
