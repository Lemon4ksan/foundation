// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xxhash_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/codec/compress/zstd/xxhash"
)

func BenchmarkSum64_16B(b *testing.B) {
	data := make([]byte, 16)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = xxhash.Sum64(data)
	}
}

func BenchmarkSum64_1KB(b *testing.B) {
	data := make([]byte, 1024)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = xxhash.Sum64(data)
	}
}

func BenchmarkSum64_128KB(b *testing.B) {
	data := make([]byte, 128*1024)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = xxhash.Sum64(data)
	}
}

func BenchmarkSum64String(b *testing.B) {
	str := "Call me Ishmael. Some years ago--never mind how long precisely-"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = xxhash.Sum64String(str)
	}
}

func BenchmarkDigestWrite_1KB(b *testing.B) {
	d := xxhash.New()
	data := make([]byte, 1024)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.Reset()
		_, _ = d.Write(data)
		_ = d.Sum64()
	}
}
