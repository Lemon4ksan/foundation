// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package offheap_test

import (
	"bytes"
	"errors"
	"io"
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

		for range numAllocs {
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
		for i := range allocCount {
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

// FuzzOffHeapBuffer tests direct OS kernel-backed OffHeapBuffer creation,
// writing, reading, seeking, reslicing, string parsing, and boundary safety.
func FuzzOffHeapBuffer(f *testing.F) {
	f.Add(4096, []byte("foundation offheap payload"), "string chunk")
	f.Add(0, []byte(""), "")
	f.Add(65536, bytes.Repeat([]byte{0x42}, 1024), "secondary text")
	f.Add(128, []byte("\x00\x01\x02\x03\xff"), "unicode: 🚀 世界")

	f.Fuzz(func(t *testing.T, capSize int, chunk []byte, strChunk string) {
		// Test capacity clamping/validation
		if capSize > 4*1024*1024 {
			return
		}
		if len(chunk) > 1024*1024 || len(strChunk) > 1024*1024 {
			return
		}

		buf, err := offheap.NewBuffer(capSize)
		if err != nil {
			return
		}
		defer buf.Release()

		// Initial capacity should be >= requested (or default 64KB if capSize <= 0)
		if buf.Cap() <= 0 {
			t.Fatalf("expected positive capacity, got %d", buf.Cap())
		}
		if buf.Len() != 0 {
			t.Fatalf("expected 0 initial length, got %d", buf.Len())
		}

		// Write bytes
		nWritten, writeErr := buf.Write(chunk)
		if writeErr != nil {
			if !errors.Is(writeErr, offheap.ErrBufferFull) {
				t.Fatalf("unexpected write error: %v", writeErr)
			}
		} else if nWritten != len(chunk) {
			t.Fatalf("short write without error: wrote %d, want %d", nWritten, len(chunk))
		}

		// Write string
		nStrWritten, strErr := buf.WriteString(strChunk)
		if strErr != nil {
			if !errors.Is(strErr, offheap.ErrBufferFull) {
				t.Fatalf("unexpected write string error: %v", strErr)
			}
		} else if nStrWritten != len(strChunk) {
			t.Fatalf("short write string without error: wrote %d, want %d", nStrWritten, len(strChunk))
		}

		// Validate Bytes() view
		activeBytes := buf.Bytes()
		if len(activeBytes) != buf.Len() {
			t.Fatalf("Bytes() length mismatch: got %d, want %d", len(activeBytes), buf.Len())
		}

		// Read back via Read (io.Reader)
		if buf.Len() > 0 {
			readBuf := make([]byte, buf.Len())
			nRead, readErr := buf.Read(readBuf)
			if readErr != nil && !errors.Is(readErr, io.EOF) {
				t.Fatalf("read failed: %v", readErr)
			}
			if !bytes.Equal(readBuf[:nRead], activeBytes[:nRead]) {
				t.Fatalf("read content mismatch")
			}

			// Subsequent read should return EOF
			dummy := make([]byte, 1)
			_, eofErr := buf.Read(dummy)
			if !errors.Is(eofErr, io.EOF) {
				t.Fatalf("expected io.EOF on fully read buffer, got %v", eofErr)
			}

			// RewindRead and verify we can read again
			buf.RewindRead()
			nRewound, _ := buf.Read(readBuf)
			if nRewound != nRead {
				t.Fatalf("rewound read length mismatch: got %d, want %d", nRewound, nRead)
			}
		}

		// Test RawBytes
		_ = buf.RawBytes(buf.Cap() / 2)
		_ = buf.RawBytes(buf.Cap() + 1) // should return nil

		// Test Reset
		buf.Reset()
		if buf.Len() != 0 {
			t.Fatalf("expected 0 length after Reset, got %d", buf.Len())
		}

		// Test operations after Release
		buf.Release()
		_, postReleaseWrite := buf.Write([]byte("test"))
		if !errors.Is(postReleaseWrite, offheap.ErrBufferClosed) {
			t.Fatalf("expected ErrBufferClosed after release, got %v", postReleaseWrite)
		}
		_, postReleaseRead := buf.Read(make([]byte, 10))
		if !errors.Is(postReleaseRead, offheap.ErrBufferClosed) {
			t.Fatalf("expected ErrBufferClosed after release, got %v", postReleaseRead)
		}
		if buf.Bytes() != nil {
			t.Fatalf("expected nil Bytes() after release")
		}
	})
}
