package cpukit

// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import (
	"runtime"
	"sync"
	"testing"

	"golang.org/x/sys/cpu"

	"github.com/lemon4ksan/foundation/testing/assert"
)

func TestDetectGFNI_Internal(t *testing.T) {
	detected := detectGFNI()
	assert.Equal(t, hasGFNI, detected)

	t.Logf("Host CPU detected GFNI: %v (GOARCH=%s)", detected, runtime.GOARCH)
	t.Logf("golang.org/x/sys/cpu X86.HasAVX512GFNI: %v", cpu.X86.HasAVX512GFNI)
	t.Logf("golang.org/x/sys/cpu X86.HasAVX512F: %v", cpu.X86.HasAVX512F)

	if runtime.GOARCH != "amd64" {
		assert.False(t, detected, "non-x86 platforms must report false for GFNI")
	}

	if cpu.X86.HasAVX512GFNI {
		assert.True(t, detected, "if AVX512GFNI is present, GFNI must be reported true")
	}
}

func TestHasGFNI_ConcurrentSafety(t *testing.T) {
	const goroutines = 64
	const iterations = 1000

	expected := HasGFNI()
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			for range iterations {
				res := HasGFNI()
				if res != expected {
					t.Errorf("inconsistent HasGFNI result: expected %v, got %v", expected, res)
				}
			}
		}()
	}

	wg.Wait()
}
