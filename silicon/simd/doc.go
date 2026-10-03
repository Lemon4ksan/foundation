// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package simd provides native PLAN9 Go assembly executing 256-bit AVX2
// and BMI2 vector instructions for high-throughput memory transformations.
//
// Standard Go scalar loops process memory 1 to 8 bytes per CPU instruction.
// In high-throughput streaming environments, scalar byte-by-byte XOR masking
// and parsing consume a substantial fraction of total CPU cycles. Utilizing
// 256-bit AVX2 vector instructions processes 32 bytes per clock cycle, enabling
// memory transformations to operate at full hardware line speed.
//
// Hardware support for AVX2 and BMI2 is automatically verified at boot,
// falling back to optimized 64-bit scalar routines on non-AVX2 CPUs.
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: 256-bit vector SIMD processing for memory transformations.
//
// Rejected compromise: Standard Go scalar loops process memory 1 to 8 bytes per CPU instruction, causing a bottleneck in high-throughput streams.
//
// Accepted cost: Assembly code is difficult to maintain and requires architecture-specific implementations (amd64).
//
// Allocations: unconstrained
package simd
