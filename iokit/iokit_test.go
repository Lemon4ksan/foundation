// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package iokit

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

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

type volatileBytesReader struct {
	data []byte
}

func (v *volatileBytesReader) Read(p []byte) (int, error) {
	return copy(p, v.data), io.EOF
}

func (v *volatileBytesReader) Bytes() ([]byte, bool) {
	return v.data, true
}

func TestBytesReader_And_BOM(t *testing.T) {
	t.Parallel()

	// 1. InspectBytes
	assert.False(t, func() bool { _, _, ok := InspectBytes(nil); return ok }())

	buf := bytes.NewBufferString("buffered-content")
	data, vol, ok := InspectBytes(buf)
	assert.True(t, ok)
	assert.False(t, vol)
	assert.Equal(t, "buffered-content", string(data))

	volReader := &volatileBytesReader{data: []byte("volatile-content")}
	dataVol, isVol, okVol := InspectBytes(volReader)
	assert.True(t, okVol)
	assert.True(t, isVol)
	assert.Equal(t, "volatile-content", string(dataVol))

	_, _, okPlain := InspectBytes(strings.NewReader("plain"))
	assert.False(t, okPlain)

	// 2. ReadAllSafe
	safe1, err := ReadAllSafe(buf)
	require.NoError(t, err)
	assert.Equal(t, "buffered-content", string(safe1))

	safeVol, err := ReadAllSafe(volReader)
	require.NoError(t, err)
	assert.Equal(t, "volatile-content", string(safeVol))

	safeEmpty, err := ReadAllSafe(bytes.NewBuffer(nil))
	require.NoError(t, err)
	assert.Nil(t, safeEmpty)

	safePlain, err := ReadAllSafe(strings.NewReader("streamed"))
	require.NoError(t, err)
	assert.Equal(t, "streamed", string(safePlain))

	// 3. StripBOMBytes
	utf8BOM := []byte{0xEF, 0xBB, 0xBF, 'h', 'i'}
	assert.Equal(t, []byte("hi"), StripBOMBytes(utf8BOM))

	utf16BE := []byte{0xFE, 0xFF, 'o', 'k'}
	assert.Equal(t, []byte("ok"), StripBOMBytes(utf16BE))

	utf16LE := []byte{0xFF, 0xFE, 'g', 'o'}
	assert.Equal(t, []byte("go"), StripBOMBytes(utf16LE))

	assert.Equal(t, []byte("clean"), StripBOMBytes([]byte("clean")))

	// 4. StripBOM Reader
	assert.Nil(t, StripBOM(nil))

	brUTF8 := bufio.NewReader(bytes.NewReader(utf8BOM))
	strippedUTF8 := StripBOM(brUTF8)
	outUTF8, err := io.ReadAll(strippedUTF8)
	require.NoError(t, err)
	assert.Equal(t, "hi", string(outUTF8))

	brUTF16 := bufio.NewReader(bytes.NewReader(utf16LE))
	strippedUTF16 := StripBOM(brUTF16)
	outUTF16, err := io.ReadAll(strippedUTF16)
	require.NoError(t, err)
	assert.Equal(t, "go", string(outUTF16))
}

func TestExplicitBufferedBody(t *testing.T) {
	t.Parallel()

	raw := io.NopCloser(strings.NewReader("stream_tail"))
	body := &ExplicitBufferedBody{
		Stream: raw,
		Prefix: []byte("prefix_head_"),
	}

	assert.Equal(t, []byte("prefix_head_"), body.BufferedPrefix())

	// First Read
	b1, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.Equal(t, "prefix_head_stream_tail", string(b1))

	// Rewind and Read again
	body.Stream = io.NopCloser(strings.NewReader("stream_tail"))
	body.Rewind()
	b2, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.Equal(t, "prefix_head_stream_tail", string(b2))

	assert.NoError(t, body.Close())
}

func TestMultiReadBody_OffHeap_And_ResponseBodyReadCloser(t *testing.T) {
	t.Parallel()

	// 1. OffHeap buffer (threshold >= 64KB, e.g. 70KB)
	largeData := strings.Repeat("X", 65536)
	inner := io.NopCloser(strings.NewReader(largeData))

	mrb, err := NewMultiReadBody(inner, 70*1024, true)
	require.NoError(t, err)
	require.NotNil(t, mrb)

	respBody := &ResponseBodyReadCloser{ReadCloser: mrb}
	data, onOffHeap := respBody.Bytes()
	assert.True(t, onOffHeap)
	assert.Equal(t, largeData, string(data))
	assert.Same(t, mrb, respBody.Unwrap())

	// Read via ResponseBodyReadCloser
	b1, err := io.ReadAll(respBody)
	require.NoError(t, err)
	assert.Equal(t, largeData, string(b1))

	// Reset via Close
	assert.NoError(t, mrb.Close())
	b2, err := io.ReadAll(mrb)
	require.NoError(t, err)
	assert.Equal(t, largeData, string(b2))

	// ReallyClose via ResponseBodyReadCloser.Close()
	assert.NoError(t, respBody.Close())

	// 2. OffHeap exceeding threshold by 1 byte with disableDisk=true
	overflowData := strings.Repeat("Y", 70*1024+1)
	innerOverflow := io.NopCloser(strings.NewReader(overflowData))
	_, err = NewMultiReadBody(innerOverflow, 70*1024, true)
	assert.ErrorIs(t, err, ErrBufferLimitExceeded)

	// 3. OffHeap exceeding threshold by 1 byte with disableDisk=false -> spills to disk
	innerDisk := io.NopCloser(strings.NewReader(overflowData))
	mrbDisk, err := NewMultiReadBody(innerDisk, 70*1024, false)
	require.NoError(t, err)
	require.NotNil(t, mrbDisk)

	bDisk, err := io.ReadAll(mrbDisk)
	require.NoError(t, err)
	assert.Equal(t, overflowData, string(bDisk))
	_ = mrbDisk.Close()
	if rc, ok := mrbDisk.(interface{ ReallyClose() }); ok {
		rc.ReallyClose()
	}
}

func TestBufferedConn(t *testing.T) {
	t.Parallel()

	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	go func() {
		_, _ = c1.Write([]byte("buffered_stream_data"))
	}()

	br := bufio.NewReader(c2)
	bConn := &BufferedConn{
		Conn: c2,
		R:    br,
	}

	buf := make([]byte, 8)
	n, err := bConn.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 8, n)
	assert.Equal(t, "buffered", string(buf))
}

func TestJitterReader_And_CopyZeroAlloc(t *testing.T) {
	t.Parallel()

	// 1. JitterReader
	jr := &JitterReader{
		ReadCloser: io.NopCloser(strings.NewReader("jitter payload")),
		Delay:      10 * time.Millisecond,
	}
	all, err := io.ReadAll(jr)
	require.NoError(t, err)
	assert.Equal(t, "jitter payload", string(all))

	// 2. CopyZeroAlloc with nil
	n, err := CopyZeroAlloc(nil, nil)
	assert.Equal(t, int64(0), n)
	require.NoError(t, err)

	// 3. CopyZeroAlloc with ReaderFrom
	var buf bytes.Buffer
	n, err = CopyZeroAlloc(&buf, strings.NewReader("readerfrom payload"))
	require.NoError(t, err)
	assert.Equal(t, int64(18), n)
	assert.Equal(t, "readerfrom payload", buf.String())

	// 4. StripBOM with standard non-bufio reader
	rawReader := strings.NewReader(string([]byte{0xEF, 0xBB, 0xBF, 'x', 'y'}))
	stripped := StripBOM(rawReader)
	out, err := io.ReadAll(stripped)
	require.NoError(t, err)
	assert.Equal(t, "xy", string(out))
}
