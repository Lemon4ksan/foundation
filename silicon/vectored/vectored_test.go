// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vectored_test

import (
	"bytes"
	"testing"

	"github.com/lemon4ksan/foundation/silicon/vectored"
	"github.com/lemon4ksan/foundation/testkit/assert"
	"github.com/lemon4ksan/foundation/testkit/require"
)

func TestBufferQueue_Basic(t *testing.T) {
	q := vectored.NewBufferQueue()
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
	assert.Equal(t, "POST /v1/chat HTTP/1.1\r\nHost: api.example.com\r\n\r\n{\"message\":\"hello\"}", buf.String())

	// Queue should be reset after WriteTo
	assert.Equal(t, 0, q.SlicesCount())
	assert.Equal(t, int64(0), q.TotalBytes())
}

func TestBufferQueue_Pool(t *testing.T) {
	q := vectored.AcquireBufferQueue()
	q.Push([]byte("test"))
	assert.Equal(t, 1, q.SlicesCount())

	vectored.ReleaseBufferQueue(q)

	q2 := vectored.AcquireBufferQueue()
	assert.Equal(t, 0, q2.SlicesCount())
	vectored.ReleaseBufferQueue(q2)
}

func BenchmarkBufferQueue_PushAndWrite(b *testing.B) {
	header := []byte("GET / HTTP/1.1\r\nHost: localhost\r\n\r\n")
	body := []byte("12345678901234567890")

	b.ReportAllocs()
	for b.Loop() {
		q := vectored.AcquireBufferQueue()
		q.Push(header)
		q.Push(body)

		var buf bytes.Buffer
		_, _ = q.WriteTo(&buf)

		vectored.ReleaseBufferQueue(q)
	}
}
