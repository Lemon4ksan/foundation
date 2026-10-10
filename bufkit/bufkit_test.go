// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bufkit_test

import (
	"bytes"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"unsafe"

	"github.com/lemon4ksan/foundation/bufkit"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

// ============================================================================
// 1. Aligned Memory Tests
// ============================================================================

func TestAlignedBytes_And_IsAligned(t *testing.T) {
	t.Parallel()

	// 1. Zero/negative size and alignment
	assert.Nil(t, bufkit.AlignedBytes(0, 64))
	assert.Nil(t, bufkit.AlignedBytes(-10, 64))

	b1 := bufkit.AlignedBytes(128, 0)
	assert.Equal(t, 128, len(b1))

	// 2. Non-power-of-2 alignment (e.g. 50 -> rounds up to 64)
	b2 := bufkit.AlignedBytes(256, 50)
	assert.Equal(t, 256, len(b2))
	assert.True(t, bufkit.IsAligned(b2, 64))

	// 3. CacheLineSize and PageSize
	bCache := bufkit.AlignedBytes(1024, bufkit.CacheLineSize)
	assert.True(t, bufkit.IsAligned(bCache, bufkit.CacheLineSize))

	bPage := bufkit.AlignedBytes(8192, bufkit.PageSize)
	assert.True(t, bufkit.IsAligned(bPage, bufkit.PageSize))

	// 4. IsAligned edge cases
	assert.True(t, bufkit.IsAligned(nil, 64))
	assert.True(t, bufkit.IsAligned([]byte{1, 2, 3}, 1))
	assert.True(t, bufkit.IsAligned([]byte{1, 2, 3}, 0))
}

// ============================================================================
// 2. Scatter-Gather Chain Buffer Tests
// ============================================================================

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestChain_AllMethods(t *testing.T) {
	t.Parallel()

	c := bufkit.NewChain()
	defer c.Release()

	// 1. Empty chain operations
	assert.Equal(t, 0, c.Len())
	assert.Nil(t, c.Bytes())
	assert.Nil(t, c.Chunks())

	n, err := c.Write(nil)
	assert.Equal(t, 0, n)
	require.NoError(t, err)

	n, err = c.WriteString("")
	assert.Equal(t, 0, n)
	require.NoError(t, err)

	rn, rerr := c.Read(make([]byte, 10))
	assert.Equal(t, 0, rn)
	assert.ErrorIs(t, rerr, io.EOF)

	rn, rerr = c.Read(nil)
	assert.Equal(t, 0, rn)
	require.NoError(t, rerr)

	wn, werr := c.WriteTo(&bytes.Buffer{})
	assert.Equal(t, int64(0), wn)
	require.NoError(t, werr)

	// 2. Write, WriteString, WriteByte
	_, _ = c.WriteString("hello ")
	_ = c.WriteByte('w')
	_, _ = c.Write([]byte("orld"))

	assert.Equal(t, "hello world", string(c.Bytes()))
	assert.Equal(t, 11, c.Len())

	// AppendBytes
	appended := c.AppendBytes(make([]byte, 0, 16))
	assert.Equal(t, "hello world", string(appended))

	chunks := c.Chunks()
	assert.Equal(t, 1, len(chunks))
	assert.Equal(t, "hello world", string(chunks[0]))

	var seqChunks [][]byte
	for chunk := range c.ChunksSeq() {
		seqChunks = append(seqChunks, chunk)
	}
	assert.Equal(t, 1, len(seqChunks))
	assert.Equal(t, "hello world", string(seqChunks[0]))

	dstChunks := make([][]byte, 0, 4)
	dstChunks = c.AppendChunks(dstChunks)
	assert.Equal(t, 1, len(dstChunks))
	assert.Equal(t, "hello world", string(dstChunks[0]))

	// ChunksSeq early break
	for range c.ChunksSeq() {
		break
	}

	// 3. Multi-chunk Write and Read across 4KB boundaries
	largePayload := make([]byte, 10000)
	for i := range largePayload {
		largePayload[i] = byte(i % 256)
	}

	c.Reset()
	assert.Nil(t, c.Chunks())
	assert.Nil(t, c.AppendChunks(nil))
	assert.Nil(t, c.AppendBytes(nil))

	_, _ = c.Write(largePayload)
	assert.Equal(t, 10000, c.Len())
	assert.Equal(t, largePayload, c.Bytes())

	// Partial Read
	part := make([]byte, 5000)
	nRead, err := io.ReadFull(c, part)
	require.NoError(t, err)
	assert.Equal(t, 5000, nRead)
	assert.Equal(t, largePayload[:5000], part)
	assert.Equal(t, 5000, c.Len())

	// WriteTo remaining 5000 bytes
	var buf bytes.Buffer
	wCount, err := c.WriteTo(&buf)
	require.NoError(t, err)
	assert.Equal(t, int64(5000), wCount)
	assert.Equal(t, largePayload[5000:], buf.Bytes())
	assert.Equal(t, 0, c.Len())

	// 4. WriteTo error handling
	_, _ = c.WriteString("data")
	_, err = c.WriteTo(errWriter{})
	assert.Error(t, err)

	// 5. Zero-value struct initialization on write
	var uninit bufkit.Chain
	nInit, err := uninit.WriteString("init on write")
	require.NoError(t, err)
	assert.Equal(t, 13, nInit)
	assert.Equal(t, 13, uninit.Len())
	uninit.Release()

	// 6. Chain beyond inline capacity (> 32KB / > 8 chunks)
	cLarge := bufkit.NewChain()
	_, _ = cLarge.Write(make([]byte, 40960))
	assert.Equal(t, 40960, cLarge.Len())
	cLarge.Reset()
	assert.Equal(t, 0, cLarge.Len())
	cLarge.Release()
}

func TestChain_StreamingExtensions(t *testing.T) {
	t.Parallel()

	c := bufkit.NewChain()
	defer c.Release()

	// 1. ReadByte on empty returns io.EOF
	_, err := c.ReadByte()
	assert.ErrorIs(t, err, io.EOF)

	// 2. ReadByte single byte
	_ = c.WriteByte('A')
	_ = c.WriteByte('B')
	b1, err := c.ReadByte()
	require.NoError(t, err)
	assert.Equal(t, byte('A'), b1)
	b2, err := c.ReadByte()
	require.NoError(t, err)
	assert.Equal(t, byte('B'), b2)
	assert.Equal(t, 0, c.Len())

	// 3. Peek within single chunk
	_, _ = c.WriteString("abcdef")
	peeked, err := c.Peek(3)
	require.NoError(t, err)
	assert.Equal(t, "abc", string(peeked))
	assert.Equal(t, 6, c.Len()) // length unchanged!

	// Peek across multiple chunks
	c.Reset()
	chunkData := make([]byte, bufkit.DefaultChunkSize)
	copy(chunkData, "FIRST_CHUNK")
	_, _ = c.Write(chunkData)
	_, _ = c.WriteString("SECOND_CHUNK")

	peekCross, err := c.Peek(bufkit.DefaultChunkSize + 6)
	require.NoError(t, err)
	assert.Equal(t, bufkit.DefaultChunkSize+6, len(peekCross))
	assert.Equal(t, "SECOND", string(peekCross[bufkit.DefaultChunkSize:]))

	// Peek beyond capacity
	_, err = c.Peek(bufkit.DefaultChunkSize * 3)
	assert.ErrorIs(t, err, io.EOF)
	p0, err := c.Peek(0)
	require.NoError(t, err)
	assert.Nil(t, p0)

	// 4. Discard
	c.Reset()
	dEmpty, err := c.Discard(5)
	assert.Equal(t, 0, dEmpty)
	assert.ErrorIs(t, err, io.EOF)

	_, _ = c.WriteString("0123456789")
	discarded, err := c.Discard(4)
	require.NoError(t, err)
	assert.Equal(t, 4, discarded)
	assert.Equal(t, 6, c.Len())
	assert.Equal(t, "456789", string(c.Bytes()))

	d0, _ := c.Discard(0)
	assert.Equal(t, 0, d0)

	// Discard across multiple chunks
	c.Reset()
	_, _ = c.Write(make([]byte, 5000))
	dMulti, err := c.Discard(4500)
	require.NoError(t, err)
	assert.Equal(t, 4500, dMulti)
	assert.Equal(t, 500, c.Len())

	// 5. Next
	c.Reset()
	_, _ = c.WriteString("prefix_payload")
	pBytes := c.Next(7)
	assert.Equal(t, "prefix_", string(pBytes))
	assert.Equal(t, "payload", string(c.Bytes()))
	assert.Nil(t, c.Next(0))

	// Next spanning multiple chunks
	c.Reset()
	_, _ = c.Write(make([]byte, 4000))
	_, _ = c.Write(make([]byte, 1000))
	_ = c.Next(3900)
	// Now 100 left in first chunk; Next(200) spans across chunk boundary into chunk 2
	spanBytes := c.Next(200)
	assert.Equal(t, 200, len(spanBytes))
	assert.Equal(t, 900, c.Len())

	// 6. ReadFrom
	c.Reset()
	reader := bytes.NewReader([]byte("streamed from io.Reader"))
	nRead, err := c.ReadFrom(reader)
	require.NoError(t, err)
	assert.Equal(t, int64(23), nRead)
	assert.Equal(t, "streamed from io.Reader", string(c.Bytes()))
}

// ============================================================================
// 2b. Regression Tests: Chain.Next Memory Safety & Chain.Peek Zero-Copy
// ============================================================================

func TestChain_Next_MemorySafety_CrossChunkBoundaryAndRecycle(t *testing.T) {
	t.Parallel()

	c := bufkit.NewChain()
	defer c.Release()

	// 1. Populate multiple chunks with distinct deterministic patterns
	const totalBytes = bufkit.DefaultChunkSize * 3 // 3 full chunks = 12288 bytes
	fullPayload := make([]byte, totalBytes)
	for i := range fullPayload {
		fullPayload[i] = byte((i*17 + 11) % 251)
	}

	nWritten, err := c.Write(fullPayload)
	require.NoError(t, err)
	assert.Equal(t, totalBytes, nWritten)

	// 2. Advance close to chunk 0 boundary: leave 64 bytes in chunk 0
	const offsetBeforeBoundary = bufkit.DefaultChunkSize - 64
	discarded, err := c.Discard(offsetBeforeBoundary)
	require.NoError(t, err)
	assert.Equal(t, offsetBeforeBoundary, discarded)

	// 3. Next across chunk 0 into chunk 1: 64 bytes from chunk 0, 64 bytes from chunk 1.
	// This exhausts chunk 0, triggering its release to sync.Pool.
	expectedSpan1 := make([]byte, 128)
	copy(expectedSpan1, fullPayload[offsetBeforeBoundary:offsetBeforeBoundary+128])

	heldSlice1 := c.Next(128)
	require.Equal(t, 128, len(heldSlice1))
	assert.Equal(t, expectedSpan1, heldSlice1)

	// 4. Next remainder of chunk 1: leaves 0 bytes in chunk 1, exhausting it and triggering release.
	const chunk1Remaining = bufkit.DefaultChunkSize - 64
	expectedSpan2 := make([]byte, chunk1Remaining)
	copy(expectedSpan2, fullPayload[offsetBeforeBoundary+128:offsetBeforeBoundary+128+chunk1Remaining])

	heldSlice2 := c.Next(chunk1Remaining)
	require.Equal(t, chunk1Remaining, len(heldSlice2))
	assert.Equal(t, expectedSpan2, heldSlice2)

	// 5. Aggressively churn sync.Pool by allocating new chains and overwriting pooled chunks
	// with a dirty pattern (0xAA). If heldSlice1 or heldSlice2 point into pooled memory,
	// their contents would be overwritten and corrupted.
	for range 50 {
		churnChain := bufkit.NewChain()
		dirty := bytes.Repeat([]byte{0xAA}, bufkit.DefaultChunkSize*2)
		_, _ = churnChain.Write(dirty)
		churnChain.Reset()
		churnChain.Release()
	}

	// Also perform more reads and writes on the original chain c
	_, _ = c.WriteString("more active chain data")
	_ = c.Next(5)
	c.Reset()
	_, _ = c.Write(bytes.Repeat([]byte{0xBB}, bufkit.DefaultChunkSize))

	// 6. Assert that held slices remain completely pristine and uncorrupted
	assert.Equal(t, expectedSpan1, heldSlice1)
	assert.Equal(t, expectedSpan2, heldSlice2)
}

func TestChain_Next_SingleChunkExhaustion_HoldingSlice(t *testing.T) {
	t.Parallel()

	c := bufkit.NewChain()
	defer c.Release()

	// Write 2 full chunks: chunk 0 is all 0x33, chunk 1 is all 0x44
	chunk0 := bytes.Repeat([]byte{0x33}, bufkit.DefaultChunkSize)
	chunk1 := bytes.Repeat([]byte{0x44}, bufkit.DefaultChunkSize)
	_, _ = c.Write(chunk0)
	_, _ = c.Write(chunk1)

	// Next exactly 1 chunk: exhausts chunk 0 and advances to chunk 1.
	// In the buggy implementation, this returned chunk 0 directly and released it to sync.Pool.
	held := c.Next(bufkit.DefaultChunkSize)
	require.Equal(t, bufkit.DefaultChunkSize, len(held))
	assert.Equal(t, chunk0, held)

	// Force reuse of the recycled chunk from sync.Pool
	reuseChain := bufkit.NewChain()
	defer reuseChain.Release()
	dirty := bytes.Repeat([]byte{0x99}, bufkit.DefaultChunkSize)
	_, _ = reuseChain.Write(dirty)

	// Verify held slice still holds 0x33, unaffected by pool reuse
	for i := range held {
		if held[i] != 0x33 {
			t.Fatalf("held slice corrupted at index %d: expected 0x33, got 0x%02x", i, held[i])
		}
	}
}

func TestChain_Next_DistinctAllocationAddress(t *testing.T) {
	t.Parallel()

	c := bufkit.NewChain()
	defer c.Release()

	_, _ = c.WriteString("distinct memory allocation test")

	peeked, err := c.Peek(8)
	require.NoError(t, err)

	nextSlice := c.Next(8)
	assert.Equal(t, string(peeked), string(nextSlice))

	// nextSlice must be a freshly allocated buffer with a distinct memory address from the chunk
	addrPeek := uintptr(unsafe.Pointer(unsafe.SliceData(peeked)))
	addrNext := uintptr(unsafe.Pointer(unsafe.SliceData(nextSlice)))
	assert.NotEqual(t, addrPeek, addrNext)
}

func TestChain_Peek_ZeroCopy(t *testing.T) {
	t.Parallel()

	c := bufkit.NewChain()
	defer c.Release()

	payload := []byte("preview_zero_copy_buffer")
	_, err := c.Write(payload)
	require.NoError(t, err)

	// Peek within chunk
	peeked, err := c.Peek(len(payload))
	require.NoError(t, err)
	assert.Equal(t, payload, peeked)
	assert.Equal(t, len(payload), c.Len()) // Peek must not advance read cursor

	// Verify zero-copy: peeked slice data points to the exact same memory as chunk 0
	activeChunks := c.Chunks()
	require.True(t, len(activeChunks) > 0)
	assert.Equal(t, unsafe.SliceData(activeChunks[0]), unsafe.SliceData(peeked))

	// In-place mutation through peeked slice affects the chunk directly (zero-copy proof)
	peeked[0] = 'Z'
	assert.Equal(t, byte('Z'), c.Chunks()[0][0])
	peeked[0] = 'p' // restore

	// Peek across multiple chunks falls back to contiguous copy
	c.Reset()
	part1 := bytes.Repeat([]byte{'A'}, bufkit.DefaultChunkSize)
	part2 := bytes.Repeat([]byte{'B'}, 100)
	_, _ = c.Write(part1)
	_, _ = c.Write(part2)

	peekCross, err := c.Peek(bufkit.DefaultChunkSize + 50)
	require.NoError(t, err)
	assert.Equal(t, bufkit.DefaultChunkSize+50, len(peekCross))
	assert.Equal(t, byte('A'), peekCross[0])
	assert.Equal(t, byte('B'), peekCross[bufkit.DefaultChunkSize])
	assert.Equal(t, bufkit.DefaultChunkSize+100, c.Len()) // Len unchanged
}

func TestChain_Next_And_Peek_EdgeCases(t *testing.T) {
	t.Parallel()

	c := bufkit.NewChain()
	defer c.Release()

	// On empty chain
	assert.Nil(t, c.Next(0))
	assert.Nil(t, c.Next(-5))
	assert.Nil(t, c.Next(10))

	p0, err := c.Peek(0)
	require.NoError(t, err)
	assert.Nil(t, p0)

	pNeg, err := c.Peek(-5)
	require.NoError(t, err)
	assert.Nil(t, pNeg)

	pEmpty, err := c.Peek(10)
	assert.ErrorIs(t, err, io.EOF)
	assert.Nil(t, pEmpty)

	// Next requesting more than available returns all remaining bytes
	_, _ = c.WriteString("small")
	assert.Equal(t, 5, c.Len())
	remaining := c.Next(100)
	assert.Equal(t, "small", string(remaining))
	assert.Equal(t, 0, c.Len())
}

// ============================================================================
// 3. Lock-Free SPSC Ring Buffer Tests
// ============================================================================

func TestRing_AllMethods_And_Concurrency(t *testing.T) {
	t.Parallel()

	// 1. Capacity rounding (< 2 rounded to 2, 7 rounded to 8)
	rSmall := bufkit.NewRing[int](1)
	assert.Equal(t, 2, rSmall.Cap())

	ring := bufkit.NewRing[int](7)
	assert.Equal(t, 8, ring.Cap())
	assert.Equal(t, 0, ring.Len())
	assert.Equal(t, 8, ring.Free())
	assert.True(t, ring.IsEmpty())
	assert.False(t, ring.IsFull())

	// 2. Pop on empty returns false
	_, ok := ring.Pop()
	assert.False(t, ok)

	// 3. Fill ring to capacity
	for i := range 8 {
		assert.True(t, ring.Push(i+100))
	}
	assert.Equal(t, 8, ring.Len())
	assert.Equal(t, 0, ring.Free())
	assert.False(t, ring.IsEmpty())
	assert.True(t, ring.IsFull())

	// 4. Push on full returns false
	assert.False(t, ring.Push(999))

	// 5. Peek next item
	peekVal, ok := ring.Peek()
	assert.True(t, ok)
	assert.Equal(t, 100, peekVal)

	// 6. Pop all items
	for i := range 8 {
		val, ok := ring.Pop()
		assert.True(t, ok)
		assert.Equal(t, i+100, val)
	}
	assert.Equal(t, 0, ring.Len())
	assert.True(t, ring.IsEmpty())

	// 7. Reset
	ring.Push(1)
	ring.Push(2)
	ring.Reset()
	assert.Equal(t, 0, ring.Len())
	_, ok = ring.Pop()
	assert.False(t, ok)

	// 8. SPSC Concurrency Test
	spscRing := bufkit.NewRing[int](128)
	const count = 100000

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := range count {
			for !spscRing.Push(i) {
				// spin wait
			}
		}
	}()

	go func() {
		defer wg.Done()
		received := 0
		for received < count {
			val, ok := spscRing.Pop()
			if ok {
				if val != received {
					t.Errorf("expected %d, got %d", received, val)
				}
				received++
			}
		}
	}()

	wg.Wait()
}

func TestRing_BatchOperations(t *testing.T) {
	t.Parallel()

	r := bufkit.NewRing[int](8)

	// State check on empty
	assert.True(t, r.IsEmpty())
	assert.False(t, r.IsFull())
	assert.Equal(t, 8, r.Free())

	// Peek on empty
	peekVal, peekOk := r.Peek()
	assert.False(t, peekOk)
	assert.Equal(t, 0, peekVal)

	// PopBatch on empty
	assert.Equal(t, 0, r.PopBatch(make([]int, 4)))

	// PushBatch
	items := []int{10, 20, 30, 40, 50}
	pushed := r.PushBatch(items)
	assert.Equal(t, 5, pushed)
	assert.Equal(t, 5, r.Len())
	assert.Equal(t, 3, r.Free())
	assert.False(t, r.IsEmpty())
	assert.False(t, r.IsFull())

	// Peek on non-empty
	peekVal, peekOk = r.Peek()
	assert.True(t, peekOk)
	assert.Equal(t, 10, peekVal)
	assert.Equal(t, 5, r.Len()) // Peek must not consume

	// PopBatch
	dest := make([]int, 3)
	popped := r.PopBatch(dest)
	assert.Equal(t, 3, popped)
	assert.Equal(t, []int{10, 20, 30}, dest)
	assert.Equal(t, 2, r.Len())

	// PopBatch remaining
	dest2 := make([]int, 5)
	popped = r.PopBatch(dest2)
	assert.Equal(t, 2, popped)
	assert.Equal(t, 40, dest2[0])
	assert.Equal(t, 50, dest2[1])
	assert.Equal(t, 0, r.Len())

	// Fill to capacity
	fullPushed := r.PushBatch([]int{1, 2, 3, 4, 5, 6, 7, 8})
	assert.Equal(t, 8, fullPushed)
	assert.True(t, r.IsFull())
	assert.Equal(t, 0, r.Free())

	// PushBatch when full
	assert.Equal(t, 0, r.PushBatch([]int{9, 10}))

	// Empty push/pop batch
	assert.Equal(t, 0, r.PushBatch(nil))
	assert.Equal(t, 0, r.PopBatch(nil))

	// Wrap-around PushBatch and PopBatch
	rWrap := bufkit.NewRing[int](4)
	_ = rWrap.PushBatch([]int{1, 2, 3})
	_ = rWrap.PopBatch(make([]int, 2)) // head = 2, tail = 3
	// Push 3 items: slot 3, then slots 0 and 1
	wrapPushed := rWrap.PushBatch([]int{10, 20, 30})
	assert.Equal(t, 3, wrapPushed)
	assert.Equal(t, 4, rWrap.Len())

	// PopBatch wrapping across boundary
	wrapDst := make([]int, 4)
	wrapPopped := rWrap.PopBatch(wrapDst)
	assert.Equal(t, 4, wrapPopped)
	assert.Equal(t, []int{3, 10, 20, 30}, wrapDst)
	assert.Equal(t, 0, rWrap.Len())
}

func TestRing_WrapAround(t *testing.T) {
	t.Parallel()

	r := bufkit.NewRing[int](4)
	require.Equal(t, 4, r.Cap())

	for cycle := range 10 {
		for i := range 3 {
			val := cycle*10 + i
			require.True(t, r.Push(val))
		}
		require.Equal(t, 3, r.Len())

		for i := range 3 {
			expected := cycle*10 + i
			val, ok := r.Pop()
			require.True(t, ok)
			require.Equal(t, expected, val)
		}
		require.Equal(t, 0, r.Len())
	}
}

func TestRing_ZeroOrNegativeCapacity(t *testing.T) {
	t.Parallel()

	r0 := bufkit.NewRing[int](0)
	assert.Equal(t, 2, r0.Cap())

	rNeg := bufkit.NewRing[int](-5)
	assert.Equal(t, 2, rNeg.Cap())
}

func TestRing_Iterators(t *testing.T) {
	t.Parallel()

	r := bufkit.NewRing[int](8)

	// Empty ring
	count := 0
	for range r.All() {
		count++
	}
	assert.Equal(t, 0, count)

	for range r.Drain() {
		count++
	}
	assert.Equal(t, 0, count)

	// Add items
	for i := 1; i <= 5; i++ {
		require.True(t, r.Push(i*10))
	}
	require.Equal(t, 5, r.Len())

	// All() with early break
	var items []int
	for v := range r.All() {
		items = append(items, v)
		if len(items) == 2 {
			break
		}
	}
	assert.Equal(t, []int{10, 20}, items)
	assert.Equal(t, 5, r.Len())

	// Values() full traversal
	items = nil
	for v := range r.Values() {
		items = append(items, v)
	}
	assert.Equal(t, []int{10, 20, 30, 40, 50}, items)
	assert.Equal(t, 5, r.Len())

	// Drain() with early exit
	items = nil
	for v := range r.Drain() {
		items = append(items, v)
		if len(items) == 2 {
			break
		}
	}
	assert.Equal(t, []int{10, 20}, items)
	assert.Equal(t, 3, r.Len())

	// Drain() remaining
	items = nil
	for v := range r.Drain() {
		items = append(items, v)
	}
	assert.Equal(t, []int{30, 40, 50}, items)
	assert.Equal(t, 0, r.Len())
}

func TestRing_ByteSliceIterators(t *testing.T) {
	t.Parallel()

	r := bufkit.NewRing[[]byte](4)
	r.Push([]byte("chunk1"))
	r.Push([]byte("chunk2"))

	var collected []string
	for b := range r.All() {
		collected = append(collected, string(b))
	}
	assert.Equal(t, []string{"chunk1", "chunk2"}, collected)

	var drained []string
	for b := range r.Drain() {
		drained = append(drained, string(b))
	}
	assert.Equal(t, []string{"chunk1", "chunk2"}, drained)
	assert.Equal(t, 0, r.Len())
}

func TestEmpirical_Ring_SPSC_HighThroughputWrapAround(t *testing.T) {
	capacities := []int{2, 4, 16, 128}

	for _, capVal := range capacities {
		t.Run(t.Name(), func(t *testing.T) {
			ring := bufkit.NewRing[int](capVal)
			const totalItems = 200000

			var wg sync.WaitGroup
			wg.Add(2)

			// Producer
			go func() {
				defer wg.Done()
				for i := range totalItems {
					for !ring.Push(i) {
					}
				}
			}()

			// Consumer
			go func() {
				defer wg.Done()
				for expected := range totalItems {
					for {
						val, ok := ring.Pop()
						if ok {
							if val != expected {
								t.Errorf("capacity %d: expected %d, got %d", capVal, expected, val)
							}
							break
						}
					}
				}
			}()

			wg.Wait()

			if ring.Len() != 0 {
				t.Fatalf("expected ring to be empty after consuming all items, got len=%d", ring.Len())
			}
		})
	}
}

func TestEmpirical_Ring_ResetStress(t *testing.T) {
	ring := bufkit.NewRing[int](8)

	for cycle := range 100 {
		for i := range 8 {
			if !ring.Push(cycle*100 + i) {
				t.Fatalf("cycle %d: push failed at item %d", cycle, i)
			}
		}
		if ring.Len() != 8 {
			t.Fatalf("cycle %d: expected len 8, got %d", cycle, ring.Len())
		}
		ring.Reset()
		if ring.Len() != 0 {
			t.Fatalf("cycle %d: expected len 0 after reset, got %d", cycle, ring.Len())
		}
		if _, ok := ring.Pop(); ok {
			t.Fatalf("cycle %d: pop on empty ring after reset succeeded", cycle)
		}
	}
}

func TestZeroAllocations(t *testing.T) {
	// Chain zero-allocation test
	chain := bufkit.NewChain()
	defer chain.Release()

	allocsChain := testing.AllocsPerRun(100, func() {
		chain.WriteString("hello world")
		chain.Reset()
	})
	assert.Equal(t, 0.0, allocsChain)

	// Ring zero-allocation test
	ring := bufkit.NewRing[int](64)
	allocsRing := testing.AllocsPerRun(100, func() {
		ring.Push(123)
		_, _ = ring.Pop()
	})
	assert.Equal(t, 0.0, allocsRing)

	// Ring batch zero-allocation test
	inBuf := []int{1, 2, 3, 4}
	outBuf := make([]int, 4)
	allocsBatch := testing.AllocsPerRun(100, func() {
		ring.PushBatch(inBuf)
		ring.PopBatch(outBuf)
	})
	assert.Equal(t, 0.0, allocsBatch)
}

// ============================================================================
// 4. Benchmarks
// ============================================================================

func BenchmarkChain_WriteReset(b *testing.B) {
	c := bufkit.NewChain()
	defer c.Release()

	payload := make([]byte, 16384) // 4 chunks
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, _ = c.Write(payload)
		c.Reset()
	}
}

func BenchmarkRing_PushPop_Sequential(b *testing.B) {
	ring := bufkit.NewRing[int](1024)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		ring.Push(42)
		ring.Pop()
	}
}

func BenchmarkRing_PushPop_Batch(b *testing.B) {
	ring := bufkit.NewRing[int](1024)
	inBatch := make([]int, 32)
	outBatch := make([]int, 32)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		ring.PushBatch(inBatch)
		ring.PopBatch(outBatch)
	}
}

func BenchmarkRing_SPSC_Pipeline(b *testing.B) {
	ring := bufkit.NewRing[int](4096)
	var active atomic.Bool
	active.Store(true)
	var wg sync.WaitGroup

	b.ReportAllocs()
	b.ResetTimer()

	wg.Go(func() {
		i := 0
		for active.Load() {
			if ring.Push(i) {
				i++
			}
		}
	})

	for b.Loop() {
		for {
			if _, ok := ring.Pop(); ok {
				break
			}
		}
	}

	b.StopTimer()
	active.Store(false)
	wg.Wait()
}

func BenchmarkRing_SPSC_Push_Parallel(b *testing.B) {
	ring := bufkit.NewRing[int](4096)
	var active atomic.Bool
	active.Store(true)

	var wg sync.WaitGroup
	wg.Go(func() {
		for active.Load() {
			ring.Pop()
		}
	})

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for !ring.Push(1) {
		}
	}

	b.StopTimer()
	active.Store(false)
	wg.Wait()
}

func BenchmarkRing_SPSC_Pop_Parallel(b *testing.B) {
	ring := bufkit.NewRing[int](4096)
	var active atomic.Bool
	active.Store(true)

	var wg sync.WaitGroup
	wg.Go(func() {
		for active.Load() {
			ring.Push(1)
		}
	})

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for {
			if _, ok := ring.Pop(); ok {
				break
			}
		}
	}

	b.StopTimer()
	active.Store(false)
	wg.Wait()
}

func BenchmarkRing_Reset(b *testing.B) {
	ring := bufkit.NewRing[int](1024)
	for i := range 1024 {
		ring.Push(i)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		ring.Reset()
	}
}
