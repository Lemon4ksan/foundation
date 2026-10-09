// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cpukit_test

import (
	"runtime"
	"sync"
	"testing"

	"github.com/lemon4ksan/foundation/silicon/cpukit"
	"github.com/lemon4ksan/foundation/testing/assert"
)

func TestCPUKitProbes(t *testing.T) {
	t.Parallel()

	_ = cpukit.HasAVX2()
	_ = cpukit.HasBMI1()
	_ = cpukit.HasBMI2()
	_ = cpukit.HasBMI()
	_ = cpukit.HasAVX512()
	_ = cpukit.HasNEON()
	_ = cpukit.HasAESNI()
	_ = cpukit.HasGFNI()

	cls := cpukit.CacheLineSize()
	assert.GreaterOrEqual(t, cls, 32)
}

func TestHasGFNI_ExecutionSafety(t *testing.T) {
	t.Parallel()

	// Calling HasGFNI on any host architecture must execute safely
	// without crashing, raising SIGILL, or panicking.
	val := cpukit.HasGFNI()

	if runtime.GOARCH != "amd64" {
		assert.False(t, val, "non-x86 platforms must safely return false for GFNI")
	}

	// Verify idempotency over multiple calls
	for range 100 {
		assert.Equal(t, val, cpukit.HasGFNI())
	}
}

func TestHasGFNI_ZeroAllocations(t *testing.T) {
	allocs := testing.AllocsPerRun(100, func() {
		_ = cpukit.HasGFNI()
	})
	assert.Equal(t, float64(0), allocs, "HasGFNI must be zero-allocation")
}

func TestHasGFNI_ConcurrentExecution(t *testing.T) {
	t.Parallel()

	expected := cpukit.HasGFNI()
	const workers = 32
	const loops = 1000

	var wg sync.WaitGroup
	wg.Add(workers)

	for range workers {
		go func() {
			defer wg.Done()
			for range loops {
				res := cpukit.HasGFNI()
				if res != expected {
					t.Errorf("concurrent HasGFNI returned %v, expected %v", res, expected)
				}
			}
		}()
	}

	wg.Wait()
}

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
