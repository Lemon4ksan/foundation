// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command c2plan9 converts C, C++, and LLVM IR source files into Plan 9 assembly for Go.
//
// The tool automates the process of invoking Clang to compile hardware-accelerated intrinsic
// functions into object files and disassembling them into Go-compatible Plan 9 assembly format.
// It supports both AMD64 (with AVX2/FMA/PCLMUL) and ARM64 (with SIMD/Crypto) targets. The tool
// can also emit corresponding Go stub files to bridge the generated assembly with Go packages.
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: automated C-to-Plan9 assembly translation
//
// Rejected compromise: hand-writing complex SIMD instructions directly in Plan 9 assembly syntax
//
// Accepted cost: introducing an external compilation toolchain (Clang/LLVM) dependency for code generation
//
// Allocations: unconstrained
package main
