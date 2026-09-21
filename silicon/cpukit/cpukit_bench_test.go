// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cpukit_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/silicon/cpukit"
)

func BenchmarkHasAVX2(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = cpukit.HasAVX2()
	}
}

func BenchmarkHasBMI(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = cpukit.HasBMI()
	}
}

func BenchmarkHasAESNI(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = cpukit.HasAESNI()
	}
}

func BenchmarkCacheLineSize(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = cpukit.CacheLineSize()
	}
}

func BenchmarkHasGFNI(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = cpukit.HasGFNI()
	}
}
