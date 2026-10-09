// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package offheap

import (
	"math"
	"sync/atomic"
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

func TestRepro_Arena_AllocNoSizePadding(t *testing.T) {
	arena, err := NewArena(64 * 1024)
	if err != nil {
		t.Fatalf("failed to create arena: %v", err)
	}
	defer arena.Release()

	// Alloc(1) must advance a.offset by exactly 1 byte, NOT pad size to 8 bytes.
	ptr1 := arena.Alloc(1)
	if ptr1 == nil {
		t.Fatal("Alloc(1) failed")
	}
	if arena.offset != 1 {
		t.Fatalf("expected arena.offset == 1 after Alloc(1), got %d", arena.offset)
	}

	// Next Alloc(1) aligns offset to 8 (due to default 8-byte alignment) and advances to 9.
	ptr2 := arena.Alloc(1)
	if ptr2 == nil {
		t.Fatal("Alloc(1) failed")
	}
	if arena.offset != 9 {
		t.Fatalf("expected arena.offset == 9, got %d", arena.offset)
	}
	if uintptr(ptr2)%8 != 0 {
		t.Fatalf("expected ptr2 to be 8-byte aligned, got %p", ptr2)
	}
}

func TestRepro_Arena_AllocAligned_DynamicAlignment(t *testing.T) {
	arena, err := NewArena(64 * 1024)
	if err != nil {
		t.Fatalf("failed to create arena: %v", err)
	}
	defer arena.Release()

	// 1. Allocate 1 byte with 1-byte alignment: offset becomes 1.
	p1 := arena.AllocAligned(1, 1)
	if p1 == nil {
		t.Fatal("AllocAligned(1, 1) failed")
	}
	if arena.offset != 1 {
		t.Fatalf("expected offset 1, got %d", arena.offset)
	}

	// 2. Allocate 1 byte with 1-byte alignment: offset becomes 2 (no padding waste).
	p2 := arena.AllocAligned(1, 1)
	if p2 == nil {
		t.Fatal("AllocAligned(1, 1) failed")
	}
	if arena.offset != 2 {
		t.Fatalf("expected offset 2, got %d", arena.offset)
	}

	// 3. Allocate 8 bytes with 8-byte alignment: offset must align from 2 to 8, then advance to 16.
	p8 := arena.AllocAligned(8, 8)
	if p8 == nil {
		t.Fatal("AllocAligned(8, 8) failed")
	}
	if uintptr(p8)%8 != 0 {
		t.Fatalf("expected p8 8-byte aligned, got %p", p8)
	}
	if arena.offset != 16 {
		t.Fatalf("expected offset 16, got %d", arena.offset)
	}

	// 4. Test atomic safety on p8 under 64-bit alignment requirement.
	atomicPtr := (*int64)(p8)
	atomic.StoreInt64(atomicPtr, 12345)
	if val := atomic.AddInt64(atomicPtr, 54321); val != 66666 {
		t.Fatalf("unexpected atomic value: %d", val)
	}
}
