// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !amd64 || purego

package simd

// ParallelExtract64 falls back to SWAR bit extraction on non-amd64 architectures.
func ParallelExtract64(val, mask uint64) uint64 {
	return extractBitsSWAR(val, mask)
}
