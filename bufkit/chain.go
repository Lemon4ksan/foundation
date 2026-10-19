// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bufkit

import (
	"errors"
	"io"
	"iter"
	"sync"
	"unsafe"
)

// DefaultChunkSize is the fixed allocation size for pooled memory chunks (4096 bytes),
// matching standard OS page boundaries to minimize fragmentation and maximize [sync.Pool] reuse.
const DefaultChunkSize = 4096

type chunk [DefaultChunkSize]byte

var chunkPool = sync.Pool{
	New: func() any {
		return new(chunk)
	},
}

func acquireChunk() []byte {
	c := chunkPool.Get().(*chunk)
	return c[:]
}

func releaseChunk(b []byte) {
	if cap(b) == DefaultChunkSize {
		c := (*chunk)(unsafe.Pointer(unsafe.SliceData(b)))
		chunkPool.Put(c)
	}
}

// Chain is a scatter-gather buffer composed of fixed-size pooled memory chunks,
// avoiding continuous reallocation and copying when growing dynamic payloads.
// The zero value is empty and ready to use.
type Chain struct {
	initial [8][]byte // inline storage for up to 8 chunks (32KB) without heap slice allocation
	chunks  [][]byte
	headIdx int // Read offset within chunks[0]
	tailIdx int // Write offset within chunks[len(chunks)-1]
	length  int // Total unread bytes across all chunks
}

// NewChain returns an empty [Chain] backed by inline chunk storage.
func NewChain() *Chain {
	c := &Chain{}
	c.chunks = c.initial[:0]
	return c
}

// Len returns the total number of unread bytes in the buffer chain.
func (c *Chain) Len() int {
	return c.length
}

// ensureInit ensures chunks slice points to inline storage if uninitialized.
func (c *Chain) ensureInit() {
	if c.chunks == nil {
		c.chunks = c.initial[:0]
	}
}

// Write appends the contents of p to the chain, acquiring new pooled chunks as needed
// without reallocating previously written data.
func (c *Chain) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	c.ensureInit()
	totalWritten := len(p)
	c.length += totalWritten

	for len(p) > 0 {
		if len(c.chunks) == 0 || c.tailIdx == DefaultChunkSize {
			c.chunks = append(c.chunks, acquireChunk())
			c.tailIdx = 0
		}

		lastChunk := c.chunks[len(c.chunks)-1]
		available := DefaultChunkSize - c.tailIdx
		toCopy := min(len(p), available)

		copy(lastChunk[c.tailIdx:], p[:toCopy])
		c.tailIdx += toCopy
		p = p[toCopy:]
	}

	return totalWritten, nil
}

// WriteString appends s to the chain without intermediate []byte conversion.
func (c *Chain) WriteString(s string) (int, error) {
	if len(s) == 0 {
		return 0, nil
	}

	c.ensureInit()
	totalWritten := len(s)
	c.length += totalWritten

	for len(s) > 0 {
		if len(c.chunks) == 0 || c.tailIdx == DefaultChunkSize {
			c.chunks = append(c.chunks, acquireChunk())
			c.tailIdx = 0
		}

		lastChunk := c.chunks[len(c.chunks)-1]
		available := DefaultChunkSize - c.tailIdx
		toCopy := min(len(s), available)

		copy(lastChunk[c.tailIdx:], s[:toCopy])
		c.tailIdx += toCopy
		s = s[toCopy:]
	}

	return totalWritten, nil
}

// WriteByte appends a single byte to the chain.
func (c *Chain) WriteByte(b byte) error {
	c.ensureInit()
	if len(c.chunks) == 0 || c.tailIdx == DefaultChunkSize {
		c.chunks = append(c.chunks, acquireChunk())
		c.tailIdx = 0
	}

	c.chunks[len(c.chunks)-1][c.tailIdx] = b
	c.tailIdx++
	c.length++
	return nil
}

// popFrontChunk recycles the leading chunk, shifts remaining chunks left to preserve
// backing slice capacity, clears the vacated slot for GC, and resets headIdx to 0.
func (c *Chain) popFrontChunk() {
	releaseChunk(c.chunks[0])
	copy(c.chunks, c.chunks[1:])
	c.chunks[len(c.chunks)-1] = nil
	c.chunks = c.chunks[:len(c.chunks)-1]
	c.headIdx = 0
}

// ReadByte reads and returns the next byte from the chain, or [io.EOF] if the buffer is empty.
func (c *Chain) ReadByte() (byte, error) {
	if c.length == 0 {
		return 0, io.EOF
	}

	currentChunk := c.chunks[0]
	b := currentChunk[c.headIdx]
	c.headIdx++
	c.length--
	if len(c.chunks) > 1 && c.headIdx == DefaultChunkSize {
		c.popFrontChunk()
	} else if len(c.chunks) == 1 && c.headIdx == c.tailIdx {
		c.headIdx = 0
		c.tailIdx = 0
	}
	return b, nil
}

// Read reads up to len(p) bytes into p, recycling exhausted chunks back to the global pool,
// and returns [io.EOF] if the buffer is empty.
func (c *Chain) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if c.length == 0 {
		return 0, io.EOF
	}

	var n int
	for len(p) > 0 && c.length > 0 {
		currentChunk := c.chunks[0]
		var available int
		if len(c.chunks) == 1 {
			available = c.tailIdx - c.headIdx
		} else {
			available = DefaultChunkSize - c.headIdx
		}

		toCopy := min(len(p), available)
		copy(p[:toCopy], currentChunk[c.headIdx:c.headIdx+toCopy])

		c.headIdx += toCopy
		n += toCopy
		c.length -= toCopy
		p = p[toCopy:]

		if len(c.chunks) > 1 && c.headIdx == DefaultChunkSize {
			c.popFrontChunk()
		} else if len(c.chunks) == 1 && c.headIdx == c.tailIdx {
			c.headIdx = 0
			c.tailIdx = 0
		}
	}

	return n, nil
}

// Peek returns the next n unread bytes without advancing the read position,
// or [io.EOF] if fewer than n bytes are available.
// When n fits within the leading chunk, Peek returns a zero-copy slice directly
// into the pooled chunk; otherwise it allocates and copies across chunk boundaries.
//
// The zero-copy slice is valid only until the next mutating operation (Read, Write,
// Discard, Reset, or Release), which may recycle the underlying chunk back to [sync.Pool].
// Use [Chain.Next] or copy the slice if the data must be retained.
func (c *Chain) Peek(n int) ([]byte, error) {
	if n <= 0 {
		return nil, nil
	}
	if n > c.length {
		return nil, io.EOF
	}

	firstChunk := c.chunks[0]
	var available int
	if len(c.chunks) == 1 {
		available = c.tailIdx - c.headIdx
	} else {
		available = DefaultChunkSize - c.headIdx
	}

	if n <= available {
		return firstChunk[c.headIdx : c.headIdx+n], nil
	}

	buf := make([]byte, n)
	offset := 0
	remaining := n

	for i, ch := range c.chunks {
		var start, end int
		if i == 0 {
			start = c.headIdx
		} else {
			start = 0
		}

		if i == len(c.chunks)-1 {
			end = c.tailIdx
		} else {
			end = DefaultChunkSize
		}

		chunkAvail := end - start
		toCopy := min(remaining, chunkAvail)
		copy(buf[offset:], ch[start:start+toCopy])
		offset += toCopy
		remaining -= toCopy
		if remaining == 0 {
			break
		}
	}

	return buf, nil
}

// Discard advances the read position by up to n bytes without copying data and returns the number of bytes discarded.
func (c *Chain) Discard(n int) (int, error) {
	if n <= 0 {
		return 0, nil
	}
	if c.length == 0 {
		return 0, io.EOF
	}

	toDiscard := min(n, c.length)
	remaining := toDiscard

	for remaining > 0 && c.length > 0 {
		var available int
		if len(c.chunks) == 1 {
			available = c.tailIdx - c.headIdx
		} else {
			available = DefaultChunkSize - c.headIdx
		}

		step := min(remaining, available)
		c.headIdx += step
		c.length -= step
		remaining -= step

		if len(c.chunks) > 1 && c.headIdx == DefaultChunkSize {
			c.popFrontChunk()
		} else if len(c.chunks) == 1 && c.headIdx == c.tailIdx {
			c.headIdx = 0
			c.tailIdx = 0
		}
	}

	return toDiscard, nil
}

// Next returns a newly allocated slice containing the next n bytes from the buffer and advances the read position.
// If fewer than n bytes are available, Next returns all remaining unread bytes; if n <= 0 or the buffer is empty, it returns nil.
func (c *Chain) Next(n int) []byte {
	if n <= 0 || c.length == 0 {
		return nil
	}

	n = min(n, c.length)
	buf := make([]byte, n)
	_, _ = c.Read(buf)
	return buf
}

// ReadFrom reads data from r until EOF or error directly into pooled chunks without intermediate heap allocations,
// returning the total bytes read and any encountered error (except [io.EOF]).
func (c *Chain) ReadFrom(r io.Reader) (int64, error) {
	var totalRead int64
	c.ensureInit()

	for {
		if len(c.chunks) == 0 || c.tailIdx == DefaultChunkSize {
			c.chunks = append(c.chunks, acquireChunk())
			c.tailIdx = 0
		}

		lastChunk := c.chunks[len(c.chunks)-1]
		available := DefaultChunkSize - c.tailIdx

		n, err := r.Read(lastChunk[c.tailIdx : c.tailIdx+available])
		if n > 0 {
			c.tailIdx += n
			c.length += n
			totalRead += int64(n)
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				return totalRead, nil
			}
			return totalRead, err
		}
	}
}

// WriteTo writes all unread chain data directly to w without flattening or intermediary allocation,
// recycling chunks sequentially as they are written.
func (c *Chain) WriteTo(w io.Writer) (int64, error) {
	if c.length == 0 {
		return 0, nil
	}

	var totalWritten int64

	for len(c.chunks) > 0 {
		chunk := c.chunks[0]
		var validSlice []byte
		if len(c.chunks) == 1 {
			validSlice = chunk[c.headIdx:c.tailIdx]
		} else {
			validSlice = chunk[c.headIdx:DefaultChunkSize]
		}

		if len(validSlice) > 0 {
			n, err := w.Write(validSlice)
			totalWritten += int64(n)
			c.headIdx += n
			c.length -= n
			if err != nil {
				return totalWritten, err
			}
		}

		c.popFrontChunk()
	}

	if cap(c.chunks) <= len(c.initial) {
		c.chunks = c.initial[:0]
	}
	c.length = 0
	c.tailIdx = 0
	return totalWritten, nil
}

// Bytes returns a newly allocated contiguous byte slice containing all unread data in the chain.
func (c *Chain) Bytes() []byte {
	if c.length == 0 {
		return nil
	}

	buf := make([]byte, c.length)
	_ = c.AppendBytes(buf[:0])
	return buf
}

// AppendBytes appends all unread bytes in the chain to dst and returns the resulting slice,
// avoiding allocations if dst has sufficient capacity.
func (c *Chain) AppendBytes(dst []byte) []byte {
	for chunk := range c.ChunksSeq() {
		dst = append(dst, chunk...)
	}
	return dst
}

// Chunks returns slices of all active chunks for vectored write operations (such as [net.Buffers]).
func (c *Chain) Chunks() [][]byte {
	if c.length == 0 {
		return nil
	}

	return c.AppendChunks(make([][]byte, 0, len(c.chunks)))
}

// AppendChunks appends slices of all active chunks to dst and returns the resulting slice,
// avoiding allocations if dst has sufficient capacity.
func (c *Chain) AppendChunks(dst [][]byte) [][]byte {
	for chunk := range c.ChunksSeq() {
		dst = append(dst, chunk)
	}
	return dst
}

// ChunksSeq returns an iterator yielding slices of all active chunks without allocating an intermediary slice.
func (c *Chain) ChunksSeq() iter.Seq[[]byte] {
	return func(yield func([]byte) bool) {
		if c.length == 0 {
			return
		}

		for i, chunk := range c.chunks {
			var start, end int
			if i == 0 {
				start = c.headIdx
			} else {
				start = 0
			}

			if i == len(c.chunks)-1 {
				end = c.tailIdx
			} else {
				end = DefaultChunkSize
			}

			if end > start {
				if !yield(chunk[start:end]) {
					return
				}
			}
		}
	}
}

// Reset clears all data in the chain and recycles allocated chunks back to the pool for reuse.
func (c *Chain) Reset() {
	for i, chunk := range c.chunks {
		releaseChunk(chunk)
		c.chunks[i] = nil
	}
	clear(c.initial[:])
	if cap(c.chunks) <= len(c.initial) {
		c.chunks = c.initial[:0]
	} else {
		clear(c.chunks[:cap(c.chunks)])
		c.chunks = c.chunks[:0]
	}
	c.headIdx = 0
	c.tailIdx = 0
	c.length = 0
}

// Release completely frees the chain and returns all chunks to the pool; it is an alias for [Chain.Reset].
func (c *Chain) Release() {
	c.Reset()
}
