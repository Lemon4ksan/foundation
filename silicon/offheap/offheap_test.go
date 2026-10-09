// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package offheap_test

import (
	"bytes"
	"errors"
	"io"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/lemon4ksan/foundation/silicon/offheap"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

// ── Shared test types ────────────────────────────────────────────────────────

type testFrameHeader struct {
	StreamID uint32
	Length   uint32
	Flags    uint8
	Type     uint8
}

type testFastHeaderSlot struct {
	Key    [32]byte
	Val    [64]byte
	KeyLen uint8
	ValLen uint8
}

// ── Buffer ─────────────────────────────────────────────────────────────

func TestOffHeapBuffer_WriteRead(t *testing.T) {
	buf, err := offheap.NewBuffer(64 * 1024)
	require.NoError(t, err)

	require.NotNil(t, buf)
	defer buf.Release()

	n, err := buf.WriteString("GET /api/v1 HTTP/1.1\r\n\r\n")
	assert.NoError(t, err)
	assert.Equal(t, 24, n)
	assert.Equal(t, 24, buf.Len())

	bView := buf.Bytes()
	assert.Equal(t, "GET /api/v1 HTTP/1.1\r\n\r\n", string(bView))

	readDst := make([]byte, 64)
	rn, rErr := buf.Read(readDst)
	assert.NoError(t, rErr)
	assert.Equal(t, 24, rn)
	assert.Equal(t, "GET /api/v1 HTTP/1.1\r\n\r\n", string(readDst[:rn]))

	buf.Reset()
	assert.Equal(t, 0, buf.Len())
}

func TestBuffer_ReadCursor_Advances(t *testing.T) {
	buf, err := offheap.NewBuffer(64 * 1024)
	require.NoError(t, err)

	defer buf.Release()

	_, _ = buf.WriteString("ABCDEFGH")

	// First Read: should get "ABCD"
	p1 := make([]byte, 4)
	n, err := buf.Read(p1)
	require.NoError(t, err)
	assert.Equal(t, 4, n)
	assert.Equal(t, "ABCD", string(p1))

	// Second Read: should get next "EFGH", not repeat "ABCD"
	p2 := make([]byte, 4)
	n, err = buf.Read(p2)
	require.NoError(t, err)
	assert.Equal(t, 4, n)
	assert.Equal(t, "EFGH", string(p2))
}

func TestBuffer_ReadEOF(t *testing.T) {
	buf, err := offheap.NewBuffer(4096)
	require.NoError(t, err)

	defer buf.Release()

	_, _ = buf.WriteString("hello")

	p := make([]byte, 128)
	n, err := buf.Read(p)
	assert.NoError(t, err)
	assert.Equal(t, 5, n)

	// Next Read must return io.EOF
	n2, err2 := buf.Read(p)
	assert.Equal(t, 0, n2)
	assert.Equal(t, io.EOF, err2)
}

func TestBuffer_RewindRead(t *testing.T) {
	buf, err := offheap.NewBuffer(4096)
	require.NoError(t, err)

	defer buf.Release()

	_, _ = buf.WriteString("rewindable")

	p := make([]byte, 16)

	n, _ := buf.Read(p)
	assert.Equal(t, "rewindable", string(p[:n]))

	// After RewindRead the same data should be readable again
	buf.RewindRead()

	p2 := make([]byte, 16)
	n2, _ := buf.Read(p2)
	assert.Equal(t, "rewindable", string(p2[:n2]))

	// Len() must still reflect written bytes (unchanged)
	assert.Equal(t, len("rewindable"), buf.Len())
}

func TestBuffer_WriteString_NoAlloc(t *testing.T) {
	buf, err := offheap.NewBuffer(64 * 1024)
	require.NoError(t, err)

	defer buf.Release()

	allocs := testing.AllocsPerRun(1000, func() {
		buf.Reset()
		_, _ = buf.WriteString("content-type: application/json\r\n")
	})

	assert.Equal(t, float64(0), allocs, "WriteString must not allocate on the heap")
}

func TestBuffer_Full(t *testing.T) {
	buf, err := offheap.NewBuffer(8)
	require.NoError(t, err)

	defer buf.Release()

	_, err = buf.Write([]byte("12345678"))
	assert.NoError(t, err)
	assert.Equal(t, 8, buf.Len())

	_, err = buf.Write([]byte("X"))
	assert.ErrorIs(t, err, offheap.ErrBufferFull)
}

func TestBuffer_ClosedOps(t *testing.T) {
	buf, err := offheap.NewBuffer(4096)
	require.NoError(t, err)
	buf.Release()

	_, err = buf.Write([]byte("x"))
	assert.ErrorIs(t, err, offheap.ErrBufferClosed)

	_, err = buf.WriteString("x")
	assert.ErrorIs(t, err, offheap.ErrBufferClosed)

	_, err = buf.Read(make([]byte, 4))
	assert.ErrorIs(t, err, offheap.ErrBufferClosed)

	assert.Nil(t, buf.Bytes())
	assert.Equal(t, 0, buf.Len())
	assert.Equal(t, 0, buf.Cap())
}

// ── Arena ─────────────────────────────────────────────────────────────────────

func TestScopeRAII_AllocAndPanicResilience(t *testing.T) {
	scopeErr := offheap.Scope(2*1024*1024, func(arena *offheap.Arena) {
		require.NotNil(t, arena)

		b1 := arena.AllocBuffer(1024)
		require.NotNil(t, b1)

		_, wErr := b1.WriteString("hello offheap arena")
		assert.NoError(t, wErr)
		assert.Equal(t, "hello offheap arena", string(b1.Bytes()))
	})
	assert.NoError(t, scopeErr)

	// Verify panic resilience: Scope must release even when fn panics.
	defer func() {
		r := recover()
		assert.NotNil(t, r, "panic should propagate through Scope")
	}()

	_ = offheap.Scope(1024*1024, func(_ *offheap.Arena) {
		panic("testing panic resilience inside scope")
	})
}

func TestAllocStruct(t *testing.T) {
	err := offheap.Scope(64*1024, func(arena *offheap.Arena) {
		hdr := offheap.AllocStruct[testFrameHeader](arena)
		require.NotNil(t, hdr)

		hdr.StreamID = 100
		hdr.Length = 16384
		hdr.Flags = 0x1
		hdr.Type = 0x0

		assert.Equal(t, uint32(100), hdr.StreamID)
		assert.Equal(t, uint32(16384), hdr.Length)
		assert.Equal(t, uint8(0x1), hdr.Flags)
		assert.Equal(t, uint8(0x0), hdr.Type)

		slot := offheap.AllocStruct[testFastHeaderSlot](arena)
		require.NotNil(t, slot)

		copy(slot.Key[:], "content-type")
		copy(slot.Val[:], "application/json")
		slot.KeyLen = 12
		slot.ValLen = 16

		assert.Equal(t, uint8(12), slot.KeyLen)
		assert.Equal(t, uint8(16), slot.ValLen)
		assert.Equal(t, "content-type", string(slot.Key[:slot.KeyLen]))
		assert.Equal(t, "application/json", string(slot.Val[:slot.ValLen]))
	})
	assert.NoError(t, err)
}

func TestAllocStruct_AlignmentCorrectness(t *testing.T) {
	// AllocStruct must return naturally aligned pointers for all integer widths.
	type aligned8 struct{ X uint64 } // 8-byte alignment

	type aligned4 struct{ X uint32 } // 4-byte alignment

	type aligned2 struct{ X uint16 } // 2-byte alignment

	type aligned1 struct{ X uint8 } // 1-byte alignment

	type mixed struct { // mixed sizes: compiler pads
		A uint8
		B uint64
		C uint16
	}

	err := offheap.Scope(256*1024, func(a *offheap.Arena) {
		// Alternate small + large to provoke alignment padding
		for range 100 {
			p1 := offheap.AllocStruct[aligned1](a)
			require.NotNil(t, p1)

			p8 := offheap.AllocStruct[aligned8](a)
			require.NotNil(t, p8)
			assert.Equal(t, uintptr(0), uintptr(unsafe.Pointer(p8))%8,
				"uint64 pointer must be 8-byte aligned")

			p4 := offheap.AllocStruct[aligned4](a)
			require.NotNil(t, p4)
			assert.Equal(t, uintptr(0), uintptr(unsafe.Pointer(p4))%4,
				"uint32 pointer must be 4-byte aligned")

			p2 := offheap.AllocStruct[aligned2](a)
			require.NotNil(t, p2)
			assert.Equal(t, uintptr(0), uintptr(unsafe.Pointer(p2))%2,
				"uint16 pointer must be 2-byte aligned")

			pm := offheap.AllocStruct[mixed](a)
			require.NotNil(t, pm)
			assert.Equal(t, uintptr(0), uintptr(unsafe.Pointer(pm))%unsafe.Alignof(*pm),
				"mixed struct pointer must satisfy its natural alignment")
		}
	})
	require.NoError(t, err)
}

func TestAlloc_OddByteFollowedByStruct_AtomicSafety(t *testing.T) {
	type byteTag struct {
		Tag uint8
	}

	type atomicCounter struct {
		Count atomic.Int64
		Sum   atomic.Uint64
	}

	err := offheap.Scope(64*1024, func(a *offheap.Arena) {
		// Test multiple unaligned offset combinations (1, 3, 5, 7 bytes)
		for _, unalignedBytes := range []int{1, 3, 5, 7} {
			// Allocate unaligned odd bytes
			tag := a.Alloc(unalignedBytes)
			require.NotNil(t, tag)
			slice := unsafe.Slice((*byte)(tag), unalignedBytes)
			slice[0] = 0x55

			// Allocate 8-byte aligned struct
			counter := offheap.AllocStruct[atomicCounter](a)
			require.NotNil(t, counter)

			// Strict 8-byte alignment verification
			ptrVal := uintptr(unsafe.Pointer(counter))
			assert.Equal(t, uintptr(0), ptrVal%8, "atomic struct must be 8-byte aligned")

			// Atomic operations must not panic with unaligned access
			counter.Count.Store(100)
			counter.Count.Add(50)
			assert.Equal(t, int64(150), counter.Count.Load())

			counter.Sum.Store(200)
			counter.Sum.Add(25)
			assert.Equal(t, uint64(225), counter.Sum.Load())

			// Tag memory should remain uncorrupted
			assert.Equal(t, byte(0x55), slice[0])
		}
	})
	require.NoError(t, err)
}

func TestAllocAligned_VariousAlignments(t *testing.T) {
	err := offheap.Scope(64*1024, func(a *offheap.Arena) {
		alignments := []int{1, 2, 4, 8, 16, 32, 64}

		for _, align := range alignments {
			// Unalign the offset by allocating 1 odd byte
			odd := a.AllocAligned(1, 1)
			require.NotNil(t, odd)

			// Allocate aligned to target boundary
			ptr := a.AllocAligned(32, align)
			require.NotNil(t, ptr)

			assert.Equal(t, uintptr(0), uintptr(ptr)%uintptr(align),
				"pointer must satisfy specified alignment")
		}

		// Invalid cases
		assert.Nil(t, a.AllocAligned(0, 8))
		assert.Nil(t, a.AllocAligned(-5, 8))
		// Large allocation exceeding size
		assert.Nil(t, a.AllocAligned(100*1024, 8))
	})
	require.NoError(t, err)
}

func TestAllocStruct_ConcurrentAtomicSafety_Race(t *testing.T) {
	type atomicStruct struct {
		A atomic.Int64
		B atomic.Uint64
	}

	const goroutines = 8
	const iterations = 500

	pool := offheap.NewArenaPool(64 * 1024)

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func(gid int) {
			defer wg.Done()

			for i := 0; i < iterations; i++ {
				arena := pool.Acquire()

				// Allocate 1 byte to purposefully unalign offset
				p1 := arena.Alloc(1)
				if p1 != nil {
					*(*byte)(p1) = byte(gid)
				}

				// Allocate 8-byte aligned struct
				s := offheap.AllocStruct[atomicStruct](arena)
				if s != nil {
					if uintptr(unsafe.Pointer(s))%8 != 0 {
						panic("unaligned atomicStruct pointer!")
					}
					s.A.Store(int64(i))
					s.A.Add(1)
					s.B.Store(uint64(gid))
					s.B.Add(10)
					if s.A.Load() != int64(i+1) || s.B.Load() != uint64(gid+10) {
						panic("corrupted atomic value")
					}
				}

				pool.Release(arena)
			}
		}(g)
	}

	wg.Wait()
}

func TestAllocStruct_AlignmentOverflow_NoCorruption(t *testing.T) {
	// Allocate a very small arena. Verify that alignment padding at exhaustion
	// causes graceful heap fallback without corrupting arena.offset > arena.size.
	type u64 struct{ X uint64 } // 8-byte alignment, 8-byte size

	// Arena of 9 bytes: fits one u64 (8 bytes), then 1 byte remains.
	// Next AllocStruct[u64] needs 7 bytes of padding (offset=8 → already aligned... let's use 11)
	// Arena of 11 bytes: first u64 at 0 (8 bytes). offset=8. Remaining=3.
	// Next u64 needs: rem = 8%8=0, no padding. Needs 8 bytes. 8>3 → fallback. ✓
	// Use 13 bytes: first u64 at 0. offset=8. Remaining=5.
	// Next u64: rem=8%8=0, no pad. Needs 8. 8>5 → Alloc returns nil → fallback. ✓
	// Use 15 bytes: offset=8. Remaining=7. Needs 8 → fallback. ✓

	arena, err := offheap.NewArena(15)
	require.NoError(t, err)

	defer arena.Release()

	p1 := offheap.AllocStruct[u64](arena)
	require.NotNil(t, p1, "first allocation must succeed")
	p1.X = 0xDEADBEEF_CAFEBABE

	// Second must fall back to heap - but must NOT panic or corrupt state
	p2 := offheap.AllocStruct[u64](arena)
	require.NotNil(t, p2, "fallback must return valid heap pointer")
	p2.X = 0x1234

	// First value must still be intact
	assert.Equal(t, uint64(0xDEADBEEF_CAFEBABE), p1.X, "off-heap data must not be corrupted")
}

func TestArena_ResetAndReuse(t *testing.T) {
	arena, err := offheap.NewArena(64 * 1024)
	require.NoError(t, err)

	defer arena.Release()

	for i := range 10 {
		hdr := offheap.AllocStruct[testFrameHeader](arena)
		require.NotNil(t, hdr)
		hdr.StreamID = uint32(i * 100)

		arena.Reset()
	}

	// Must not panic or return nil after repeated Reset cycles.
}

// ── ArenaPool ─────────────────────────────────────────────────────────────────

func TestArenaPool_BasicAcquireRelease(t *testing.T) {
	pool := offheap.NewArenaPool(1024 * 1024)

	a := pool.Acquire()
	require.NotNil(t, a)

	hdr := offheap.AllocStruct[testFrameHeader](a)
	require.NotNil(t, hdr)
	hdr.StreamID = 999

	pool.Release(a)

	// Re-acquire: arena must be reset (offset 0), but memory valid.
	a2 := pool.Acquire()
	require.NotNil(t, a2)
	pool.Release(a2)
}

func TestArenaPool_PageSize(t *testing.T) {
	pool := offheap.NewArenaPool(512 * 1024)
	assert.Equal(t, 512*1024, pool.PageSize())
}

func TestArenaPool_DefaultPageSize(t *testing.T) {
	pool := offheap.NewArenaPool(0)
	assert.Equal(t, 2*1024*1024, pool.PageSize())
}

func TestArenaPool_Concurrent(t *testing.T) {
	pool := offheap.NewArenaPool(512 * 1024)

	const (
		goroutines = 64
		iters      = 200
	)

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			for i := range iters {
				a := pool.Acquire()
				if a == nil {
					continue
				}

				hdr := offheap.AllocStruct[testFrameHeader](a)
				if hdr != nil {
					hdr.StreamID = uint32(i)
				}

				pool.Release(a)
			}
		}()
	}

	wg.Wait()
}

// ── Fuzz ──────────────────────────────────────────────────────────────────────

func FuzzArena_AllocStruct(f *testing.F) {
	f.Add(1024, 8)
	f.Add(64, 1)
	f.Add(1<<20, 500)
	f.Add(1, 1)

	f.Fuzz(func(t *testing.T, arenaSize, allocCount int) {
		if arenaSize <= 0 || arenaSize > 32*1024*1024 {
			return
		}

		if allocCount < 0 || allocCount > 50_000 {
			return
		}

		arena, err := offheap.NewArena(arenaSize)
		if err != nil {
			return
		}
		defer arena.Release()

		for i := range allocCount {
			hdr := offheap.AllocStruct[testFrameHeader](arena)
			if hdr == nil {
				// Arena exhausted, fell back to heap - expected.
				break
			}

			hdr.StreamID = uint32(i)
		}

		// Must not panic after exhaustion + Reset cycle.
		arena.Reset()

		hdr := offheap.AllocStruct[testFrameHeader](arena)
		if hdr != nil {
			hdr.StreamID = 42
		}
	})
}

func TestOffHeap_RemainingEdgeCases(t *testing.T) {
	// RawBytes test
	buf, err := offheap.NewBuffer(1024)
	require.NoError(t, err)
	defer buf.Release()

	buf.WriteString("hello")
	raw := buf.RawBytes(10)
	assert.Equal(t, 10, len(raw))

	assert.Equal(t, 10, buf.Len())
	assert.Equal(t, 1024, buf.Cap())

	// Released buffer methods
	buf2, _ := offheap.NewBuffer(128)
	buf2.Release()
	assert.Equal(t, 0, buf2.Len())
	assert.Equal(t, 0, buf2.Cap())
	assert.Nil(t, buf2.RawBytes(10))
	assert.Nil(t, buf2.Bytes())
	_, err = buf2.Write([]byte("abc"))
	assert.Error(t, err)
	_, err = buf2.WriteString("abc")
	assert.Error(t, err)
	buf2.RewindRead()

	// StructBytes & CastBytes
	type Plain struct {
		A uint64
		B uint32
	}
	p := &Plain{A: 100, B: 200}
	sb := offheap.StructBytes(p)
	assert.Equal(t, int(unsafe.Sizeof(Plain{})), len(sb))

	var nilP *Plain
	assert.Nil(t, offheap.StructBytes(nilP))

	castBack := offheap.CastBytes[Plain](sb)
	require.Len(t, castBack, 1)
	assert.Equal(t, uint64(100), castBack[0].A)
	assert.Equal(t, uint32(200), castBack[0].B)

	// CastBytes too short
	tooShort := offheap.CastBytes[Plain](sb[:4])
	assert.Nil(t, tooShort)

	// Arena AllocBuffer and Scope
	arena, err := offheap.NewArena(4096)
	require.NoError(t, err)
	defer arena.Release()

	ab := arena.AllocBuffer(512)
	require.NotNil(t, ab)
	assert.Equal(t, 512, ab.Cap())

	// offheap.Scope
	called := false
	errScope := offheap.Scope(4096, func(a *offheap.Arena) {
		called = true
		_ = offheap.AllocStruct[testFrameHeader](a)
	})
	assert.NoError(t, errScope)
	assert.True(t, called)

	// Slab allocator released methods
	slab, err := offheap.NewSlabAllocator[testFrameHeader](64)
	require.NoError(t, err)
	assert.Equal(t, 0, slab.Len())
	assert.Equal(t, 64, slab.Cap())
	assert.Equal(t, 64, slab.Available())
	slab.Reset()
	slab.Release()
	assert.Equal(t, 0, slab.Len())
	slab.Release() // double release safety
}

// Merged from buffer_adversarial_test.go

func TestBuffer_Adversarial_ZeroAndNegativeCapacity(t *testing.T) {
	// Zero capacity should default to 64KB
	bufZero, err := offheap.NewBuffer(0)
	require.NoError(t, err)
	require.NotNil(t, bufZero)
	assert.Equal(t, 64*1024, bufZero.Cap())
	assert.Equal(t, 0, bufZero.Len())
	bufZero.Release()

	// Negative capacity should default to 64KB
	bufNeg, err := offheap.NewBuffer(-1024)
	require.NoError(t, err)
	require.NotNil(t, bufNeg)
	assert.Equal(t, 64*1024, bufNeg.Cap())
	bufNeg.Release()

	// Exact 1-byte buffer boundary
	buf1, err := offheap.NewBuffer(1)
	require.NoError(t, err)
	defer buf1.Release()

	assert.Equal(t, 1, buf1.Cap())
	n, err := buf1.Write([]byte{0x42})
	assert.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, 1, buf1.Len())

	// Overflowing by 1 byte
	n2, err2 := buf1.Write([]byte{0x43})
	assert.True(t, errors.Is(err2, offheap.ErrBufferFull))
	assert.Equal(t, 0, n2)
	assert.Equal(t, 1, buf1.Len())
}

func TestBuffer_Adversarial_HugeAllocations(t *testing.T) {
	// 16MB allocation directly from kernel
	const hugeSize = 16 * 1024 * 1024
	buf, err := offheap.NewBuffer(hugeSize)
	require.NoError(t, err)
	require.NotNil(t, buf)
	defer buf.Release()

	assert.Equal(t, hugeSize, buf.Cap())

	// Fill first and last chunk
	payload := bytes.Repeat([]byte{0xAB}, 64*1024)
	n, err := buf.Write(payload)
	assert.NoError(t, err)
	assert.Equal(t, len(payload), n)

	// Reslice to full 16MB via RawBytes
	raw := buf.RawBytes(hugeSize)
	require.NotNil(t, raw)
	assert.Equal(t, hugeSize, len(raw))

	// Touch beginning, middle, and end
	raw[0] = 0x11
	raw[hugeSize/2] = 0x22
	raw[hugeSize-1] = 0x33

	assert.Equal(t, byte(0x11), raw[0])
	assert.Equal(t, byte(0x22), raw[hugeSize/2])
	assert.Equal(t, byte(0x33), raw[hugeSize-1])
}

func TestBuffer_Adversarial_OperationsAfterReleaseAndNilReceiver(t *testing.T) {
	buf, err := offheap.NewBuffer(1024)
	require.NoError(t, err)
	_, _ = buf.WriteString("temporary data")

	// Release buffer
	buf.Release()

	// Double Release should be an idempotent no-op
	buf.Release()

	// All mutating and reading methods must safely return ErrBufferClosed or nil
	_, wErr := buf.Write([]byte("more"))
	assert.True(t, errors.Is(wErr, offheap.ErrBufferClosed))

	_, sErr := buf.WriteString("more string")
	assert.True(t, errors.Is(sErr, offheap.ErrBufferClosed))

	dst := make([]byte, 10)
	_, rErr := buf.Read(dst)
	assert.True(t, errors.Is(rErr, offheap.ErrBufferClosed))

	assert.Nil(t, buf.Bytes())
	assert.Nil(t, buf.RawBytes(100))
	assert.Equal(t, 0, buf.Len())
	assert.Equal(t, 0, buf.Cap())

	// Reset and RewindRead on released buffer should not panic
	buf.Reset()
	buf.RewindRead()

	// Nil receiver safety
	var nilBuf *offheap.Buffer
	_, nilWErr := nilBuf.Write([]byte("abc"))
	assert.True(t, errors.Is(nilWErr, offheap.ErrBufferClosed))

	_, nilSErr := nilBuf.WriteString("abc")
	assert.True(t, errors.Is(nilSErr, offheap.ErrBufferClosed))

	_, nilRErr := nilBuf.Read(dst)
	assert.True(t, errors.Is(nilRErr, offheap.ErrBufferClosed))

	assert.Nil(t, nilBuf.Bytes())
	assert.Nil(t, nilBuf.RawBytes(10))
	assert.Equal(t, 0, nilBuf.Len())
	assert.Equal(t, 0, nilBuf.Cap())
	nilBuf.Reset()
	nilBuf.RewindRead()
	nilBuf.Release()
}

func TestBuffer_Adversarial_BoundaryRawReslicing(t *testing.T) {
	const capSize = 4096
	buf, err := offheap.NewBuffer(capSize)
	require.NoError(t, err)
	defer buf.Release()

	// Zero or negative length
	assert.Nil(t, buf.RawBytes(0))
	assert.Nil(t, buf.RawBytes(-1))
	assert.Nil(t, buf.RawBytes(-100))

	// Exactly capSize
	full := buf.RawBytes(capSize)
	require.NotNil(t, full)
	assert.Equal(t, capSize, len(full))
	assert.Equal(t, capSize, buf.Len())

	// Exceeding capacity by 1 byte
	assert.Nil(t, buf.RawBytes(capSize+1))
	// Len() should remain unchanged after rejected reslice
	assert.Equal(t, capSize, buf.Len())

	// Extreme length
	assert.Nil(t, buf.RawBytes(1<<30))

	// Reslice to smaller
	half := buf.RawBytes(capSize / 2)
	require.NotNil(t, half)
	assert.Equal(t, capSize/2, len(half))
	assert.Equal(t, capSize/2, buf.Len())
}

func TestBuffer_Adversarial_SynchronizedConcurrentWrites(t *testing.T) {
	// Buffer documents that external synchronization is required.
	// We stress-test concurrent writers coordinating through a sync.Mutex.
	const totalWriters = 32
	const writesPerWorker = 50
	const chunkSize = 64

	buf, err := offheap.NewBuffer(totalWriters * writesPerWorker * chunkSize)
	require.NoError(t, err)
	defer buf.Release()

	var mu sync.Mutex
	var wg sync.WaitGroup

	for w := range totalWriters {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			payload := bytes.Repeat([]byte{byte(workerID)}, chunkSize)
			for range writesPerWorker {
				mu.Lock()
				n, wErr := buf.Write(payload)
				mu.Unlock()
				assert.NoError(t, wErr)
				assert.Equal(t, chunkSize, n)
			}
		}(w)
	}

	wg.Wait()

	expectedTotal := totalWriters * writesPerWorker * chunkSize
	assert.Equal(t, expectedTotal, buf.Len())

	// Validate sequential reading through io.Reader
	readDst := make([]byte, 256)
	totalRead := 0
	for {
		n, rErr := buf.Read(readDst)
		totalRead += n
		if errors.Is(rErr, io.EOF) {
			break
		}
		assert.NoError(t, rErr)
	}
	assert.Equal(t, expectedTotal, totalRead)
}

// Merged from buffer_finalizer_challenge_test.go

// TestChallenge_FinalizerReclaim ensures unreleased buffers are cleaned up by GC finalizers.
func TestChallenge_FinalizerReclaim(t *testing.T) {
	for range 100 {
		buf, err := offheap.NewBuffer(4096)
		require.NoError(t, err)
		_, _ = buf.WriteString("temporary data that should be reclaimed")
		_ = buf // dropped without Release()
	}

	// Trigger GC to run finalizers
	for range 3 {
		runtime.GC()
		time.Sleep(10 * time.Millisecond)
	}

	// Allocate another buffer to verify OS paging remains healthy
	bufAfter, err := offheap.NewBuffer(4096)
	require.NoError(t, err)
	defer bufAfter.Release()
	require.Equal(t, 4096, bufAfter.Cap())
}

// TestChallenge_BufferZeroAllocHotPath verifies that Write, Read, Reset hot paths do not allocate heap memory.
func TestChallenge_BufferZeroAllocHotPath(t *testing.T) {
	buf, err := offheap.NewBuffer(1024)
	require.NoError(t, err)
	defer buf.Release()

	payload := []byte("0123456789abcdef")
	readBuf := make([]byte, len(payload))

	allocs := testing.AllocsPerRun(100, func() {
		buf.Reset()
		_, _ = buf.Write(payload)
		buf.RewindRead()
		_, _ = buf.Read(readBuf)
	})

	require.Equal(t, float64(0), allocs, "hot path operations must have 0 heap allocations")
}

// Merged from offheap_fuzz_test.go

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
