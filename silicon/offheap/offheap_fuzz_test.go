// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package offheap_test

import (
	"bytes"
	"testing"
	"unsafe"

	"github.com/lemon4ksan/foundation/silicon/offheap"
)

// FuzzArenaAlloc tests single-cycle pointer bump allocation and boundary limits against arbitrary allocation patterns.
func FuzzArenaAlloc(f *testing.F) {
	f.Add(64*1024, 128, 5)
	f.Add(4096, 4096, 1)
	f.Add(1024, 2048, 2)
	f.Add(0, -1, 0)
	f.Add(1024*1024, 65536, 10)

	f.Fuzz(func(t *testing.T, arenaSize, allocSize, numAllocs int) {
		if arenaSize <= 0 || arenaSize > 16*1024*1024 {
			return
		}
		if numAllocs < 0 || numAllocs > 100 {
			return
		}

		arena, err := offheap.NewArena(arenaSize)
		if err != nil {
			return
		}
		defer arena.Release()

		for i := 0; i < numAllocs; i++ {
			ptr := arena.Alloc(allocSize)
			if allocSize <= 0 {
				if ptr != nil {
					t.Fatalf("expected nil for non-positive alloc size %d", allocSize)
				}
				continue
			}

			if ptr != nil {
				// Verify memory is writable
				slice := unsafe.Slice((*byte)(ptr), allocSize)
				slice[0] = 0xAA
				slice[len(slice)-1] = 0xBB
			}
		}

		arena.Reset()
	})
}

// FuzzBufferWriteAndRead verifies off-heap buffer expansion, byte streaming, and string parsing.
func FuzzBufferWriteAndRead(f *testing.F) {
	f.Add([]byte("hello world"), []byte("second write chunk"))
	f.Add([]byte(""), []byte(""))
	f.Add(bytes.Repeat([]byte{0xFF}, 1024), []byte{0x00, 0x01})

	f.Fuzz(func(t *testing.T, chunkA, chunkB []byte) {
		if len(chunkA) > 1024*1024 || len(chunkB) > 1024*1024 {
			return
		}

		totalLen := len(chunkA) + len(chunkB)
		if totalLen == 0 {
			totalLen = 16
		}

		_ = offheap.Scope(totalLen*2+1024, func(a *offheap.Arena) {
			buf := a.AllocBuffer(totalLen)
			if buf == nil {
				return
			}

			nA, errA := buf.Write(chunkA)
			if errA != nil || nA != len(chunkA) {
				t.Fatalf("write chunkA failed: %v", errA)
			}

			nB, errB := buf.Write(chunkB)
			if errB != nil || nB != len(chunkB) {
				t.Fatalf("write chunkB failed: %v", errB)
			}

			expected := append(chunkA, chunkB...)
			if !bytes.Equal(buf.Bytes(), expected) {
				t.Fatalf("buffer mismatch: got %d bytes, expected %d", buf.Len(), len(expected))
			}

			if string(buf.Bytes()) != string(expected) {
				t.Fatalf("buffer string mismatch")
			}
		})
	})
}

// FuzzSlabPoolAlloc tests lock-free slab chunk slicing and boundaries.
func FuzzSlabPoolAlloc(f *testing.F) {
	f.Add(256, 10)
	f.Add(512, 100)
	f.Add(64, 0)
	f.Add(1024, 2)

	type PODSlot struct {
		ID   uint64
		Data [64]byte
	}

	f.Fuzz(func(t *testing.T, slotCount, allocCount int) {
		if slotCount <= 0 || slotCount > 4096 {
			return
		}
		if allocCount < 0 || allocCount > 128 {
			return
		}

		slab, err := offheap.NewSlabAllocator[PODSlot](slotCount)
		if err != nil {
			return
		}
		defer slab.Release()

		allocated := make([]*PODSlot, 0, allocCount)
		for i := 0; i < allocCount; i++ {
			ptr := slab.Alloc()
			if ptr != nil {
				ptr.ID = uint64(i + 1)
				allocated = append(allocated, ptr)
			}
		}

		for _, ptr := range allocated {
			_ = slab.Free(ptr)
		}
	})
}
