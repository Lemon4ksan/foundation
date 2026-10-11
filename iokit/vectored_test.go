// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package iokit_test

import (
	"bytes"
	"testing"

	"github.com/lemon4ksan/foundation/iokit"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

func TestBufferQueue_Basic(t *testing.T) {
	q := iokit.NewBufferQueue()
	assert.Equal(t, 0, q.SlicesCount())
	assert.Equal(t, int64(0), q.TotalBytes())

	q.Push([]byte("POST /v1/chat HTTP/1.1\r\n"))
	q.Push([]byte("Host: api.example.com\r\n\r\n"))
	q.Push([]byte(`{"message":"hello"}`))
	q.Push([]byte("")) // Should be ignored

	assert.Equal(t, 3, q.SlicesCount())
	assert.Equal(t, int64(68), q.TotalBytes())

	var buf bytes.Buffer
	n, err := q.WriteTo(&buf)
	require.NoError(t, err)
	assert.Equal(t, int64(68), n)
	assert.Equal(
		t,
		"POST /v1/chat HTTP/1.1\r\nHost: api.example.com\r\n\r\n{\"message\":\"hello\"}",
		string(buf.Bytes()),
	)

	// Queue should be reset after WriteTo
	assert.Equal(t, 0, q.SlicesCount())
	assert.Equal(t, int64(0), q.TotalBytes())
}

func TestBufferQueue_Pool(t *testing.T) {
	q := iokit.AcquireBufferQueue()
	q.Push([]byte("test"))
	assert.Equal(t, 1, q.SlicesCount())

	iokit.ReleaseBufferQueue(q)

	q2 := iokit.AcquireBufferQueue()
	assert.Equal(t, 0, q2.SlicesCount())
	iokit.ReleaseBufferQueue(q2)
}

func TestBufferQueue_Iterators(t *testing.T) {
	q := iokit.NewBufferQueue()

	// Empty
	count := 0
	for range q.Buffers() {
		count++
	}
	assert.Equal(t, 0, count)

	q.Push([]byte("foo"))
	q.Push([]byte("bar"))
	q.Push([]byte("baz"))

	var collected []string
	for b := range q.Buffers() {
		collected = append(collected, string(b))
	}
	assert.Equal(t, []string{"foo", "bar", "baz"}, collected)

	// Early break
	var firstTwo []string
	for b := range q.Buffers() {
		firstTwo = append(firstTwo, string(b))
		if len(firstTwo) == 2 {
			break
		}
	}
	assert.Equal(t, []string{"foo", "bar"}, firstTwo)

	// All() with indices
	indices := make([]int, 0)
	for idx := range q.All() {
		indices = append(indices, idx)
	}
	assert.Equal(t, []int{0, 1, 2}, indices)

	// All() early break
	earlyIndices := make([]int, 0)
	for idx := range q.All() {
		earlyIndices = append(earlyIndices, idx)
		if idx == 1 {
			break
		}
	}
	assert.Equal(t, []int{0, 1}, earlyIndices)
}

func TestBufferQueue_Limits(t *testing.T) {
	q := iokit.WithMaxSlices(2)
	assert.False(t, q.IsFull())

	q.Push([]byte("1"))
	assert.False(t, q.IsFull())

	q.Push([]byte("2"))
	assert.True(t, q.IsFull())
}

func TestBufferQueue_EmptyWrite(t *testing.T) {
	q := iokit.NewBufferQueue()
	var buf bytes.Buffer
	n, err := q.WriteTo(&buf)
	require.NoError(t, err)
	assert.Equal(t, int64(0), n)
	assert.Equal(t, 0, buf.Len())
}

func TestBufferQueue_CapacityRetention(t *testing.T) {
	t.Parallel()

	q := iokit.NewBufferQueue()
	for range 20 {
		q.Push([]byte("test"))
		var buf bytes.Buffer
		_, err := q.WriteTo(&buf)
		require.NoError(t, err)
	}

	assert.True(t, q.Cap() >= 16)
}

func TestBufferQueue_EdgeCases(t *testing.T) {
	t.Parallel()

	// 1. WithMaxSlices with non-positive values should default to DefaultMaxVectoredSlices (16)
	qZero := iokit.WithMaxSlices(0)
	assert.Equal(t, 16, qZero.Cap())

	qNeg := iokit.WithMaxSlices(-5)
	assert.Equal(t, 16, qNeg.Cap())

	// 2. ReleaseBufferQueue with nil should not panic
	iokit.ReleaseBufferQueue(nil)

	// 3. ReleaseBufferQueue with oversized queue should not be returned to pool
	qHuge := iokit.WithMaxSlices(128)
	for range 100 {
		qHuge.Push([]byte("chunk"))
	}
	iokit.ReleaseBufferQueue(qHuge)
}
