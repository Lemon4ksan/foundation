// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package endian_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/silicon/endian"
)

func BenchmarkLoad64LE(b *testing.B) {
	buf := make([]byte, 16)
	endian.Store64(buf, 0, 0x1122334455667788)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = endian.Load64(buf, 0)
	}
}

func BenchmarkStore64LE(b *testing.B) {
	buf := make([]byte, 16)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		endian.Store64(buf, 0, 0x1122334455667788)
	}
}

func BenchmarkUint64LE(b *testing.B) {
	buf := make([]byte, 16)
	endian.PutUint64LE(buf, 0x1122334455667788)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = endian.Uint64LE(buf)
	}
}

func BenchmarkPutUint64LE(b *testing.B) {
	buf := make([]byte, 16)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		endian.PutUint64LE(buf, 0x1122334455667788)
	}
}

func BenchmarkLoadBE64(b *testing.B) {
	buf := make([]byte, 16)
	endian.StoreBE64(buf, 0, 0x1122334455667788)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = endian.LoadBE64(buf, 0)
	}
}

func BenchmarkStoreBE64(b *testing.B) {
	buf := make([]byte, 16)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		endian.StoreBE64(buf, 0, 0x1122334455667788)
	}
}

func BenchmarkUint64BE(b *testing.B) {
	buf := make([]byte, 16)
	endian.PutUint64BE(buf, 0x1122334455667788)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = endian.Uint64BE(buf)
	}
}

func BenchmarkPutUint64BE(b *testing.B) {
	buf := make([]byte, 16)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		endian.PutUint64BE(buf, 0x1122334455667788)
	}
}
