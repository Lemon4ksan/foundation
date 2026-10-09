// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !amd64 || purego

package cpukit

import "golang.org/x/sys/cpu"

var hasGFNI = detectGFNI()

func detectGFNI() bool {
	// On non-x86 platforms (e.g. arm64), cpu.X86 feature flags are all false.
	// On purego or other architectures, check AVX-512 GFNI safely via x/sys/cpu.
	return cpu.X86.HasAVX512GFNI
}
