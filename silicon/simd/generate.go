// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package simd

//go:generate c2plan9 -c ../../csrc/equalfold.c -o equalfold_amd64.s -stub equalfold_amd64.go -pkg simd
//go:generate c2plan9 -c ../../csrc/fastscan.c -o fastscan_amd64.s -stub fastscan_amd64.go -pkg simd
//go:generate c2plan9 -c ../../csrc/match.c -o match_amd64.s -stub match_amd64.go -pkg simd
//go:generate c2plan9 -c ../../csrc/hash.c -o hash_amd64.s -stub hash_amd64.go -pkg simd
