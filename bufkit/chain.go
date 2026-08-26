// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bufkit

import (
	"errors"
	"io"
	"sync"
)

// DefaultChunkSize specifies the default fixed allocation size for memory chunks (4096 bytes / 4KB).
const DefaultChunkSize = 4096

var chunkPool = sync.Pool{
	New: func() any {
		b := make([]byte, DefaultChunkSize)
		return &b
	},
}

func acquireChunk() []byte {
	return *chunkPool.Get().(*[]byte)
}

func releaseChunk(b []byte) {
	if cap(b) == DefaultChunkSize {
		chunkPool.Put(&b)
	}
}

// Chain is a zero-allocation scatter-gather buffer composed of fixed-size pooled memory chunks.
// It avoids continuous memory reallocation and copying when growing dynamic payloads.
type Chain struct {
	chunks  [][]byte
	headIdx int // Read offset within chunks[0]
	tailIdx int // Write offset within chunks[len(chunks)-1]
	length  int // Total unread bytes across all chunks
}

// NewChain instantiates an empty [Chain].
func NewChain() *Chain {
	return &Chain{
		chunks: make([][]byte, 0, 4),
	}
}

// Len returns the total number of unread bytes in the buffer chain.
func (c *Chain) Len() int {
	return c.length
}

// Write appends the contents of p to the chain, acquiring new pooled chunks as needed.
func (c *Chain) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

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

// WriteString appends a string to the chain without heap allocation.
func (c *Chain) WriteString(s string) (int, error) {
	if len(s) == 0 {
		return 0, nil
	}

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
	if len(c.chunks) == 0 || c.tailIdx == DefaultChunkSize {
		c.chunks = append(c.chunks, acquireChunk())
		c.tailIdx = 0
	}

	c.chunks[len(c.chunks)-1][c.tailIdx] = b
	c.tailIdx++
	c.length++
	return nil
}

// Read reads up to len(p) bytes into p from the chain.
func (c *Chain) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if c.length == 0 {
		return 0, io.EOF
	}

	var n int
	for len(p) > 0 && len(c.chunks) > 0 {
		currentChunk := c.chunks[0]
		var available int
		if len(c.chunks) == 1 {
			available = c.tailIdx - c.headIdx
		} else {
			available = DefaultChunkSize - c.headIdx
		}

		if available <= 0 {
			releaseChunk(currentChunk)
			c.chunks = c.chunks[1:]
			c.headIdx = 0
			continue
		}

		toCopy := min(len(p), available)
		copy(p[:toCopy], currentChunk[c.headIdx:c.headIdx+toCopy])

		c.headIdx += toCopy
		n += toCopy
		c.length -= toCopy
		p = p[toCopy:]

		if len(c.chunks) > 1 && c.headIdx == DefaultChunkSize {
			releaseChunk(currentChunk)
			c.chunks = c.chunks[1:]
			c.headIdx = 0
		}
	}

	return n, nil
}

// WriteTo writes all unread chain data directly to an [io.Writer] without flattening or intermediary allocation.
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
			if err != nil {
				c.headIdx += n
				c.length -= n
				return totalWritten, err
			}
		}

		releaseChunk(chunk)
		c.chunks = c.chunks[1:]
		c.headIdx = 0
	}

	c.length = 0
	c.tailIdx = 0
	return totalWritten, nil
}

// Bytes returns a contiguous byte slice containing all unread data in the chain.
func (c *Chain) Bytes() []byte {
	if c.length == 0 {
		return nil
	}

	buf := make([]byte, c.length)
	var offset int

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

		n := copy(buf[offset:], chunk[start:end])
		offset += n
	}

	return buf
}

// Chunks returns slices of all active chunks for vectored write operations (such as [net.Buffers]).
func (c *Chain) Chunks() [][]byte {
	if c.length == 0 {
		return nil
	}

	res := make([][]byte, 0, len(c.chunks))
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
			res = append(res, chunk[start:end])
		}
	}
	return res
}

// Reset clears all data in the chain and recycles allocated chunks back to the pool.
func (c *Chain) Reset() {
	for _, chunk := range c.chunks {
		releaseChunk(chunk)
	}
	c.chunks = c.chunks[:0]
	c.headIdx = 0
	c.tailIdx = 0
	c.length = 0
}

// Release completely frees the chain and returns all chunks to the pool.
func (c *Chain) Release() {
	c.Reset()
}

var ErrBufferFull = errors.New("bufkit: buffer is full")
var ErrBufferEmpty = errors.New("bufkit: buffer is empty")
