// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package offheap

import (
	"math"
	"testing"
)

func TestRepro_Arena_AllocIntegerOverflow(t *testing.T) {
	arena, err := NewArena(64 * 1024)
	if err != nil {
		t.Fatalf("failed to create arena: %v", err)
	}
	defer arena.Release()

	// 1. math.MaxInt32 would overflow int32(n) + 7 to negative, bypassing bounds check
	ptr := arena.Alloc(math.MaxInt32)
	if ptr != nil {
		t.Fatalf("CRITICAL: Alloc(math.MaxInt32) succeeded and returned non-nil pointer %p", ptr)
	}

	// 2. A 64-bit int larger than 32-bit (e.g. 1<<32 + 16) would truncate in int32(n) to 16
	ptr = arena.Alloc((1 << 32) + 16)
	if ptr != nil {
		t.Fatalf("CRITICAL: Alloc((1<<32) + 16) truncated and returned non-nil pointer %p", ptr)
	}

	// 3. math.MaxInt
	ptr = arena.Alloc(math.MaxInt)
	if ptr != nil {
		t.Fatalf("CRITICAL: Alloc(math.MaxInt) succeeded and returned non-nil pointer %p", ptr)
	}
}
