// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bufkit_test

import (
	"bytes"
	"errors"
	"io"
	"sync"
	"testing"

	"github.com/lemon4ksan/foundation/bufkit"
	"github.com/lemon4ksan/foundation/testkit/assert"
	"github.com/lemon4ksan/foundation/testkit/require"
)

func TestAlignedBytes_And_IsAligned(t *testing.T) {
	t.Parallel()

	// 1. Zero/negative alignment
	b1 := bufkit.AlignedBytes(128, 0)
	assert.Equal(t, 128, len(b1))

	// 2. Non-power-of-2 alignment (e.g. 50 -> rounds to 64)
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

	chunks := c.Chunks()
	assert.Equal(t, 1, len(chunks))
	assert.Equal(t, "hello world", string(chunks[0]))

	// 3. Multi-chunk Write and Read across 4KB boundaries
	largePayload := make([]byte, 10000)
	for i := range largePayload {
		largePayload[i] = byte(i % 256)
	}

	c.Reset()
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
}

func TestRing_AllMethods_And_Concurrency(t *testing.T) {
	t.Parallel()

	// 1. Capacity rounding (< 2 rounded to 2, 7 rounded to 8)
	rSmall := bufkit.NewRing[int](1)
	assert.Equal(t, 2, rSmall.Cap())

	ring := bufkit.NewRing[int](7)
	assert.Equal(t, 8, ring.Cap())
	assert.Equal(t, 0, ring.Len())

	// 2. Pop on empty returns false
	_, ok := ring.Pop()
	assert.False(t, ok)

	// 3. Fill ring to capacity
	for i := 0; i < 8; i++ {
		assert.True(t, ring.Push(i+100))
	}
	assert.Equal(t, 8, ring.Len())

	// 4. Push on full returns false
	assert.False(t, ring.Push(999))

	// 5. Pop all items
	for i := 0; i < 8; i++ {
		val, ok := ring.Pop()
		assert.True(t, ok)
		assert.Equal(t, i+100, val)
	}
	assert.Equal(t, 0, ring.Len())

	// 6. Reset
	ring.Push(1)
	ring.Push(2)
	ring.Reset()
	assert.Equal(t, 0, ring.Len())
	_, ok = ring.Pop()
	assert.False(t, ok)

	// 7. SPSC Concurrency Test
	spscRing := bufkit.NewRing[int](128)
	const count = 100000

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < count; i++ {
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
