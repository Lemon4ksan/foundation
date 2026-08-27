// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package iokit

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/lemon4ksan/foundation/testkit/assert"
	"github.com/lemon4ksan/foundation/testkit/require"
)

type threadSafeReader struct {
	mu   sync.Mutex
	data []byte
	pos  int
}

func (r *threadSafeReader) Read(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.pos >= len(r.data) {
		return 0, io.EOF
	}

	n := copy(p, r.data[r.pos:])
	r.pos += n

	return n, nil
}

type customTestCloser struct {
	io.Closer
	marker int
}

func (c *customTestCloser) Unwrap() io.Closer {
	return c.Closer
}

type mockTrackedCloser struct {
	io.Reader
	closed bool
}

func (m *mockTrackedCloser) Close() error {
	m.closed = true
	return nil
}

type ioErrorReader struct {
	data []byte
	err  error
}

func (r *ioErrorReader) Read(p []byte) (int, error) {
	if len(r.data) > 0 {
		n := copy(p, r.data)
		r.data = r.data[n:]

		return n, nil
	}

	return 0, r.err
}

func (r *ioErrorReader) Close() error {
	return nil
}

func TestUnwrapTo(t *testing.T) {
	t.Parallel()

	inner := io.NopCloser(strings.NewReader("test"))
	wrapped := &customTestCloser{Closer: inner, marker: 999}

	target, ok := UnwrapTo[*customTestCloser](wrapped)
	assert.True(t, ok)
	assert.Equal(t, 999, target.marker)

	_, ok = UnwrapTo[string](wrapped)
	assert.False(t, ok)
}

func TestUnwrapBody(t *testing.T) {
	t.Parallel()

	rawCloser := &mockTrackedCloser{Reader: strings.NewReader("raw")}
	wrapped := &customTestCloser{Closer: rawCloser, marker: 123}

	unwrapped := UnwrapBody(wrapped)
	assert.Same(t, rawCloser, unwrapped)
}

func TestCopyZeroAlloc(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	src := strings.NewReader("zero_alloc_copy_payload")

	n, err := CopyZeroAlloc(&buf, src)
	require.NoError(t, err)
	assert.Equal(t, int64(23), n)
	assert.Equal(t, "zero_alloc_copy_payload", buf.String())
}

func TestAsReplayable_Operations(t *testing.T) {
	t.Parallel()

	t.Run("nil_input", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, AsReplayable(nil))
	})

	t.Run("wrap_and_read_multiple_times", func(t *testing.T) {
		t.Parallel()

		inner := io.NopCloser(strings.NewReader("hello world"))
		rep := AsReplayable(inner)
		require.NotNil(t, rep)

		b1, err := io.ReadAll(rep)
		require.NoError(t, err)
		assert.Equal(t, "hello world", string(b1))

		rep.Reset()

		b2, err := io.ReadAll(rep)
		require.NoError(t, err)
		assert.Equal(t, "hello world", string(b2))
	})

	t.Run("unwrap_already_replayable", func(t *testing.T) {
		t.Parallel()

		inner := io.NopCloser(strings.NewReader("data"))
		rep1 := AsReplayable(inner)

		rep2 := AsReplayable(rep1)
		assert.Equal(t, rep1, rep2)
	})
}

func TestReadAllHelpers(t *testing.T) {
	t.Parallel()

	t.Run("read_all_string_and_bytes", func(t *testing.T) {
		t.Parallel()

		inner := io.NopCloser(strings.NewReader("reusable stream content"))
		rep := AsReplayable(inner)

		str, err := ReadAllString(rep)
		require.NoError(t, err)
		assert.Equal(t, "reusable stream content", str)

		data, err := ReadAllBytes(rep)
		require.NoError(t, err)
		assert.Equal(t, []byte("reusable stream content"), data)
	})

	t.Run("helpers_with_nil_input", func(t *testing.T) {
		t.Parallel()

		str, err := ReadAllString(nil)
		require.NoError(t, err)
		assert.Empty(t, str)

		data, err := ReadAllBytes(nil)
		require.NoError(t, err)
		assert.Nil(t, data)
	})
}

func TestProgressReader(t *testing.T) {
	t.Parallel()

	inner := io.NopCloser(strings.NewReader("abcdefghij"))

	var (
		lastCurrent int64
		lastTotal   int64
	)

	pr := &ProgressReader{
		Reader: inner,
		Total:  10,
		OnProgress: func(current, total int64) {
			lastCurrent = current
			lastTotal = total
		},
	}

	buf := make([]byte, 4)
	n, err := pr.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 4, n)
	assert.Equal(t, int64(4), lastCurrent)
	assert.Equal(t, int64(10), lastTotal)

	all, err := io.ReadAll(pr)
	require.NoError(t, err)
	assert.Equal(t, "efghij", string(all))
	assert.Equal(t, int64(10), lastCurrent)

	assert.NoError(t, pr.Close())
}

func TestContextCancelingReadCloser(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	inner := io.NopCloser(strings.NewReader("test"))

	cc := &ContextCancelingReadCloser{
		ReadCloser: inner,
		Cancel:     cancel,
	}

	assert.NoError(t, ctx.Err())

	err := cc.Close()
	assert.NoError(t, err)
	assert.ErrorIs(t, ctx.Err(), context.Canceled)
	assert.Equal(t, inner, cc.Unwrap())
}

func TestDecompressReadCloser(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	_, err := gw.Write([]byte("gzipped data payload"))
	require.NoError(t, gw.Close())

	gr, err := gzip.NewReader(&buf)
	require.NoError(t, err)

	rc := &DecompressReadCloser{
		Reader: gr,
		Closer: io.NopCloser(nil),
	}

	out, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, "gzipped data payload", string(out))

	assert.NoError(t, rc.Close())
}

func TestPooledGzipReader(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	_, err := gw.Write([]byte("compressed string"))
	require.NoError(t, gw.Close())

	pgr, err := NewPooledGzipReader(&buf)
	require.NoError(t, err)

	out, err := io.ReadAll(pgr)
	require.NoError(t, err)
	assert.Equal(t, "compressed string", string(out))

	assert.NoError(t, pgr.Close())
}

func TestLimitCheckingReadCloser(t *testing.T) {
	t.Parallel()

	inner := io.NopCloser(strings.NewReader("1234567890"))
	l := &LimitCheckingReadCloser{
		ReadCloser: inner,
		Limit:      5,
	}

	buf := make([]byte, 4)
	n, err := l.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 4, n)

	_, err = l.Read(buf)
	assert.ErrorIs(t, err, ErrResponseTooLarge)
	assert.Equal(t, inner, l.Unwrap())
}

func TestMultiReadBody_RAM(t *testing.T) {
	t.Parallel()

	payload := "small RAM multi read buffer"
	inner := io.NopCloser(strings.NewReader(payload))

	mrb, err := NewMultiReadBody(inner, 1024, true)
	require.NoError(t, err)
	require.NotNil(t, mrb)

	b1, err := io.ReadAll(mrb)
	require.NoError(t, err)
	assert.Equal(t, payload, string(b1))

	require.NoError(t, mrb.Close())

	b2, err := io.ReadAll(mrb)
	require.NoError(t, err)
	assert.Equal(t, payload, string(b2))

	if respCloser, ok := mrb.(interface{ ReallyClose() }); ok {
		respCloser.ReallyClose()
	}
}

func TestMultiReadBody_DiskSpill(t *testing.T) {
	t.Parallel()

	payload := strings.Repeat("A", 2048)
	inner := io.NopCloser(strings.NewReader(payload))

	mrb, err := NewMultiReadBody(inner, 512, false)
	require.NoError(t, err)
	require.NotNil(t, mrb)

	b1, err := io.ReadAll(mrb)
	require.NoError(t, err)
	assert.Equal(t, payload, string(b1))

	require.NoError(t, mrb.Close())

	b2, err := io.ReadAll(mrb)
	require.NoError(t, err)
	assert.Equal(t, payload, string(b2))

	if respCloser, ok := mrb.(interface{ ReallyClose() }); ok {
		respCloser.ReallyClose()
	}
}

func TestMultiReadBody_LimitExceededNoDisk(t *testing.T) {
	t.Parallel()

	payload := strings.Repeat("B", 2048)
	inner := io.NopCloser(strings.NewReader(payload))

	_, err := NewMultiReadBody(inner, 512, true)
	assert.ErrorIs(t, err, ErrBufferLimitExceeded)
}

func TestBufioReadCloser(t *testing.T) {
	t.Parallel()

	src := strings.NewReader("bufio stream test payload")
	closer := &mockTrackedCloser{Reader: src}

	brc := NewBufioReadCloser(src, closer)
	require.NotNil(t, brc)

	peek, err := brc.Peek(5)
	require.NoError(t, err)
	assert.Equal(t, "bufio", string(peek))

	readBuf := make([]byte, 12)
	n, err := brc.Read(readBuf)
	require.NoError(t, err)
	assert.Equal(t, 12, n)
	assert.Equal(t, "bufio stream", string(readBuf))

	assert.NotNil(t, brc.BufioReader())
	assert.NoError(t, brc.Close())
	assert.True(t, closer.closed)
}

func TestLimitToContentLength(t *testing.T) {
	t.Parallel()

	src := strings.NewReader("1234567890extra_garbage")
	limited := LimitToContentLength(src, 10)

	all, err := io.ReadAll(limited)
	require.NoError(t, err)
	assert.Equal(t, "1234567890", string(all))

	assert.Nil(t, LimitToContentLength(nil, 10))
	assert.Equal(t, src, LimitToContentLength(src, -1))
}
