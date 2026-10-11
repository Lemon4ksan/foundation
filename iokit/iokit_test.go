// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package iokit

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
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

		rep.Reset()

		b3, err := io.ReadAll(rep)
		require.NoError(t, err)
		assert.Equal(t, "hello world", string(b3))
	})

	t.Run("partial_read_and_multiple_replays", func(t *testing.T) {
		t.Parallel()

		payload := "the quick brown fox jumps over the lazy dog"
		inner := io.NopCloser(strings.NewReader(payload))
		rep := AsReplayable(inner)
		require.NotNil(t, rep)

		// Partial read 1: read first 10 bytes
		buf1 := make([]byte, 10)
		n, err := io.ReadFull(rep, buf1)
		require.NoError(t, err)
		assert.Equal(t, 10, n)
		assert.Equal(t, "the quick ", string(buf1))

		// Reset and read full
		rep.Reset()
		full1, err := io.ReadAll(rep)
		require.NoError(t, err)
		assert.Equal(t, payload, string(full1))

		// Replay 2: full stream again
		rep.Reset()
		full2, err := io.ReadAll(rep)
		require.NoError(t, err)
		assert.Equal(t, payload, string(full2))

		// Partial read 2: read first 5 bytes
		rep.Reset()
		buf2 := make([]byte, 5)
		n, err = io.ReadFull(rep, buf2)
		require.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, "the q", string(buf2))

		// Reset and replay 3: full stream again
		rep.Reset()
		full3, err := io.ReadAll(rep)
		require.NoError(t, err)
		assert.Equal(t, payload, string(full3))
	})

	t.Run("new_replayable_body_constructor", func(t *testing.T) {
		t.Parallel()

		repNil, err := NewReplayableBody(nil)
		require.NoError(t, err)
		assert.Nil(t, repNil)

		inner := io.NopCloser(strings.NewReader("constructor test stream"))
		rep, err := NewReplayableBody(inner)
		require.NoError(t, err)
		require.NotNil(t, rep)

		b1, err := io.ReadAll(rep)
		require.NoError(t, err)
		assert.Equal(t, "constructor test stream", string(b1))

		rep.Reset()

		b2, err := io.ReadAll(rep)
		require.NoError(t, err)
		assert.Equal(t, "constructor test stream", string(b2))
	})

	t.Run("disk_spill_multiple_replays", func(t *testing.T) {
		t.Parallel()

		// Payload larger than DefaultReplayableThreshold (64KB), e.g. 128KB
		payload := strings.Repeat("0123456789abcdef", 8192) // 128KB
		inner := io.NopCloser(strings.NewReader(payload))
		rep, err := NewReplayableBody(inner)
		require.NoError(t, err)
		require.NotNil(t, rep)

		// Partial read 256 bytes
		prefix := make([]byte, 256)
		n, err := io.ReadFull(rep, prefix)
		require.NoError(t, err)
		assert.Equal(t, 256, n)
		assert.Equal(t, payload[:256], string(prefix))

		// Reset and replay full stream 1
		rep.Reset()
		full1, err := io.ReadAll(rep)
		require.NoError(t, err)
		assert.Equal(t, payload, string(full1))

		// Reset and replay full stream 2
		rep.Reset()
		full2, err := io.ReadAll(rep)
		require.NoError(t, err)
		assert.Equal(t, payload, string(full2))

		if reallyCloser, ok := rep.(interface{ ReallyClose() }); ok {
			reallyCloser.ReallyClose()
		}
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

func TestCopyBufPool_ZeroLengthSlice(t *testing.T) {
	// Not t.Parallel() to avoid racing with other tests accessing copyBufPool

	// 1. Artificially put a zero-length slice into copyBufPool
	zeroSlice := make([]byte, 32*1024)[:0]
	copyBufPool.Put(&zeroSlice)

	data := bytes.Repeat([]byte("X"), 5000)
	src := io.NopCloser(bytes.NewReader(data))
	var dst bytes.Buffer

	n, err := copyWithLimit(&dst, src, 10000)
	require.NoError(t, err)
	assert.Equal(t, int64(5000), n)
	assert.Equal(t, data, dst.Bytes())

	// 2. Test CopyZeroAlloc with a zero-length slice in the pool
	zeroSlice2 := make([]byte, 32*1024)[:0]
	copyBufPool.Put(&zeroSlice2)

	data2 := bytes.Repeat([]byte("Y"), 4000)
	type plainReader struct{ io.Reader }
	type plainWriter struct{ io.Writer }
	var dst2 bytes.Buffer
	n2, err := CopyZeroAlloc(&plainWriter{Writer: &dst2}, &plainReader{Reader: bytes.NewReader(data2)})
	require.NoError(t, err)
	assert.Equal(t, int64(4000), n2)
	assert.Equal(t, data2, dst2.Bytes())

	// 3. Test with cap=0 slice
	zeroCapSlice := make([]byte, 0)
	copyBufPool.Put(&zeroCapSlice)

	var dst3 bytes.Buffer
	n3, err := copyWithLimit(&dst3, io.NopCloser(bytes.NewReader(data)), 10000)
	require.NoError(t, err)
	assert.Equal(t, int64(5000), n3)
	assert.Equal(t, data, dst3.Bytes())
}

func TestMultiReadBody_ClosedAfterReallyClose(t *testing.T) {
	t.Parallel()

	payload := strings.Repeat("A", 100)
	inner := io.NopCloser(strings.NewReader(payload))

	mrb, err := NewMultiReadBody(inner, 512, false)
	require.NoError(t, err)

	reallyCloser, ok := mrb.(interface{ ReallyClose() })
	require.True(t, ok)
	reallyCloser.ReallyClose()

	var buf [10]byte
	_, rErr := mrb.Read(buf[:])
	assert.Error(t, rErr)

	cErr := mrb.Close()
	assert.Error(t, cErr)
}

func TestStripBOM_ComprehensiveEdgeCases(t *testing.T) {
	t.Parallel()

	// 1. nil reader
	assert.Nil(t, StripBOM(nil))

	// 2. bufio.Reader with UTF-16BE
	bom16BE := []byte{0xFE, 0xFF, 'A', 'B'}
	r16BE := bufio.NewReader(bytes.NewReader(bom16BE))
	out16BE, err := io.ReadAll(StripBOM(r16BE))
	require.NoError(t, err)
	assert.Equal(t, "AB", string(out16BE))

	// 3. bufio.Reader with short content (< 2 bytes)
	rShort := bufio.NewReader(bytes.NewReader([]byte{'Z'}))
	outShort, err := io.ReadAll(StripBOM(rShort))
	require.NoError(t, err)
	assert.Equal(t, "Z", string(outShort))

	// 4. bufio.Reader empty
	rEmpty := bufio.NewReader(bytes.NewReader(nil))
	outEmpty, err := io.ReadAll(StripBOM(rEmpty))
	require.NoError(t, err)
	assert.Equal(t, "", string(outEmpty))

	// 5. non-bufio reader with empty stream (n == 0)
	type unbufferedWrapper struct{ io.Reader }
	rawEmpty := &unbufferedWrapper{Reader: bytes.NewReader(nil)}
	outRawEmpty, err := io.ReadAll(StripBOM(rawEmpty))
	require.NoError(t, err)
	assert.Equal(t, "", string(outRawEmpty))

	// 6. non-bufio reader with UTF-16BE
	raw16BE := &unbufferedWrapper{Reader: bytes.NewReader([]byte{0xFE, 0xFF, 'h', 'i'})}
	outRaw16BE, err := io.ReadAll(StripBOM(raw16BE))
	require.NoError(t, err)
	assert.Equal(t, "hi", string(outRaw16BE))

	// 7. non-bufio reader with UTF-16LE
	raw16LE := &unbufferedWrapper{Reader: bytes.NewReader([]byte{0xFF, 0xFE, 'o', 'k'})}
	outRaw16LE, err := io.ReadAll(StripBOM(raw16LE))
	require.NoError(t, err)
	assert.Equal(t, "ok", string(outRaw16LE))

	// 8. non-bufio reader exact UTF-8 BOM only (off == n)
	rawExact8 := &unbufferedWrapper{Reader: bytes.NewReader([]byte{0xEF, 0xBB, 0xBF})}
	outExact8, err := io.ReadAll(StripBOM(rawExact8))
	require.NoError(t, err)
	assert.Equal(t, "", string(outExact8))

	// 9. non-bufio reader exact UTF-16 BOM only (off == n)
	rawExact16 := &unbufferedWrapper{Reader: bytes.NewReader([]byte{0xFF, 0xFE})}
	outExact16, err := io.ReadAll(StripBOM(rawExact16))
	require.NoError(t, err)
	assert.Equal(t, "", string(outExact16))

	// 10. non-bufio reader non-BOM bytes (off == 0)
	rawNonBOM := &unbufferedWrapper{Reader: bytes.NewReader([]byte{1, 2, 3, 4})}
	outNonBOM, err := io.ReadAll(StripBOM(rawNonBOM))
	require.NoError(t, err)
	assert.Equal(t, []byte{1, 2, 3, 4}, outNonBOM)
}

func TestCopyZeroAlloc_EdgeCases(t *testing.T) {
	t.Parallel()

	// 1. nil reader or nil writer
	n, err := CopyZeroAlloc(nil, strings.NewReader("foo"))
	require.NoError(t, err)
	assert.Equal(t, int64(0), n)

	var buf bytes.Buffer
	n, err = CopyZeroAlloc(&buf, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(0), n)

	// 2. WriterTo reader to a non-ReaderFrom writer
	type plainWriter struct{ io.Writer }
	pw := &plainWriter{Writer: &buf}
	buf.Reset()
	wt := strings.NewReader("payload-writer-to")
	n, err = CopyZeroAlloc(pw, wt)
	require.NoError(t, err)
	assert.Equal(t, int64(17), n)
	assert.Equal(t, "payload-writer-to", buf.String())
}

type failWriter struct {
	err error
}

func (f *failWriter) Write(_ []byte) (int, error) {
	return 0, f.err
}

func TestCopyWithLimit_Errors(t *testing.T) {
	t.Parallel()

	testErr := errors.New("read fault")
	errR := &ioErrorReader{data: []byte("partial"), err: testErr}
	var buf bytes.Buffer

	// Reader returns error during copyWithLimit
	_, err := copyWithLimit(&buf, io.NopCloser(errR), 1024)
	assert.ErrorIs(t, err, testErr)

	// Writer returns error during copyWithLimit
	writeErr := errors.New("write fault")
	fw := &failWriter{err: writeErr}
	_, err = copyWithLimit(fw, io.NopCloser(strings.NewReader("some data")), 1024)
	assert.ErrorIs(t, err, writeErr)
}

type failingReplayableBody struct {
	io.ReadCloser
}

func (f *failingReplayableBody) Reset()                 {}
func (f *failingReplayableBody) BufferedPrefix() []byte { return nil }

func TestReadAllString_And_ReadAllBytes_NilAndErrors(t *testing.T) {
	t.Parallel()

	// 1. nil ReplayableBody
	s, err := ReadAllString(nil)
	require.NoError(t, err)
	assert.Equal(t, "", s)

	b, err := ReadAllBytes(nil)
	require.NoError(t, err)
	assert.Nil(t, b)

	// 2. error during read
	readErr := errors.New("replayable read fault")
	failing := &failingReplayableBody{ReadCloser: io.NopCloser(&ioErrorReader{err: readErr})}

	s, err = ReadAllString(failing)
	assert.ErrorIs(t, err, readErr)
	assert.Equal(t, "", s)

	b, err = ReadAllBytes(failing)
	assert.ErrorIs(t, err, readErr)
	assert.Nil(t, b)
}

func TestNewReplayableBody_EdgeCases(t *testing.T) {
	t.Parallel()

	// 1. nil
	rb, err := NewReplayableBody(nil)
	require.NoError(t, err)
	assert.Nil(t, rb)

	// 2. Already a ReplayableBody wrapped inside an Unwrap closer
	mrb, err := NewMultiReadBody(io.NopCloser(strings.NewReader("nested")), 1024, false)
	require.NoError(t, err)
	wrapped := &customReadCloserWrapper{ReadCloser: mrb}
	rb2, err := NewReplayableBody(wrapped)
	require.NoError(t, err)
	assert.NotNil(t, rb2)

	// 3. Reader error during NewReplayableBody
	readErr := errors.New("source read error")
	_, err = NewReplayableBody(io.NopCloser(&ioErrorReader{err: readErr}))
	assert.ErrorIs(t, err, readErr)
}

type customReadCloserWrapper struct {
	io.ReadCloser
}

func (c *customReadCloserWrapper) Unwrap() io.Closer {
	return c.ReadCloser
}

func TestProgressReader_CloserEdgeCases(t *testing.T) {
	t.Parallel()

	// Reader does NOT implement io.Closer
	pr := &ProgressReader{
		Reader:     strings.NewReader("hello"),
		OnProgress: func(_, _ int64) {},
		Total:      5,
	}
	assert.NoError(t, pr.Close())
}

type errorCloser struct {
	io.Reader
	err error
}

func (e *errorCloser) Close() error {
	return e.err
}

func TestDecompressReadCloser_UnwrapAndErrors(t *testing.T) {
	t.Parallel()

	closer := &mockTrackedCloser{Reader: strings.NewReader("payload")}
	d := &DecompressReadCloser{
		Reader: strings.NewReader("payload"),
		Closer: closer,
	}

	// Unwrap
	assert.Same(t, closer, d.Unwrap())

	// Close when Reader is not io.Closer
	assert.NoError(t, d.Close())
	assert.True(t, closer.closed)

	// Close when Reader fails
	rErr := errors.New("reader close error")
	cErr := errors.New("closer close error")
	dErr1 := &DecompressReadCloser{
		Reader: &errorCloser{err: rErr},
		Closer: io.NopCloser(nil),
	}
	assert.ErrorIs(t, dErr1.Close(), rErr)

	// Close when Closer fails
	dErr2 := &DecompressReadCloser{
		Reader: strings.NewReader("data"),
		Closer: &errorCloser{err: cErr},
	}
	assert.ErrorIs(t, dErr2.Close(), cErr)
}

func TestPooledGzipReader_ErrorAndUnwrap(t *testing.T) {
	t.Parallel()

	// 1. Invalid gzip header returns error from NewPooledGzipReader
	_, err := NewPooledGzipReader(strings.NewReader("not a valid gzip header stream"))
	assert.Error(t, err)

	// 2. Valid gzip with plain io.Reader (not io.Closer)
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	_, err = gw.Write([]byte("pooled gzip test"))
	require.NoError(t, gw.Close())

	type plainGzipReader struct{ io.Reader }
	plain := &plainGzipReader{Reader: bytes.NewReader(buf.Bytes())}
	pgr, err := NewPooledGzipReader(plain)
	require.NoError(t, err)

	// Unwrap on non-closer returns nil
	if u, ok := pgr.(interface{ Unwrap() io.Closer }); ok {
		assert.Nil(t, u.Unwrap())
	}
	assert.NoError(t, pgr.Close())

	// 3. Valid gzip with io.Closer reader -> Unwrap returns it
	buf.Reset()
	gw = gzip.NewWriter(&buf)
	_, err = gw.Write([]byte("pooled closer test"))
	require.NoError(t, gw.Close())

	tracked := &mockTrackedCloser{Reader: bytes.NewReader(buf.Bytes())}
	pgrCloser, err := NewPooledGzipReader(tracked)
	require.NoError(t, err)

	if u, ok := pgrCloser.(interface{ Unwrap() io.Closer }); ok {
		assert.Same(t, tracked, u.Unwrap())
	}
	assert.NoError(t, pgrCloser.Close())
	assert.True(t, tracked.closed)
}

func TestResponseBodyReadCloser_BytesFallback(t *testing.T) {
	t.Parallel()

	rb := &ResponseBodyReadCloser{ReadCloser: io.NopCloser(strings.NewReader("test"))}
	data, onOffHeap := rb.Bytes()
	assert.Nil(t, data)
	assert.False(t, onOffHeap)
}

func TestMultiReadBody_ComprehensiveEdgeCases(t *testing.T) {
	t.Parallel()

	// 1. nil ReadCloser
	mrbNil, err := NewMultiReadBody(nil, 1024, false)
	require.NoError(t, err)
	assert.Nil(t, mrbNil)

	// 2. OffHeap path error during copyWithLimit
	readErr := errors.New("stream err offheap")
	failingR := io.NopCloser(&ioErrorReader{data: []byte("initial"), err: readErr})
	_, err = NewMultiReadBody(failingR, 70*1024, false)
	assert.ErrorIs(t, err, readErr)

	// 3. RAM path error during copyWithLimit
	failingR2 := io.NopCloser(&ioErrorReader{data: []byte("initial"), err: readErr})
	_, err = NewMultiReadBody(failingR2, 512, false)
	assert.ErrorIs(t, err, readErr)

	// 4. Disk-backed MultiReadBody: Reset() method and stream failure
	diskData := strings.Repeat("D", 2048)
	mrbDisk, err := NewMultiReadBody(io.NopCloser(strings.NewReader(diskData)), 512, false)
	require.NoError(t, err)
	mrbTyped, ok := mrbDisk.(*MultiReadBody)
	require.True(t, ok)
	mrbTyped.Reset()

	bDisk, err := io.ReadAll(mrbDisk)
	require.NoError(t, err)
	assert.Equal(t, diskData, string(bDisk))
	mrbTyped.ReallyClose()

	// 5. Disk stream failure during writeInitialAndStreamData
	failingDiskR := io.NopCloser(&ioErrorReader{data: []byte(strings.Repeat("X", 600)), err: readErr})
	_, err = NewMultiReadBody(failingDiskR, 512, false)
	assert.ErrorIs(t, err, readErr)

	// 6. ReallyClose idempotence and closed checks
	mrbRAM, err := NewMultiReadBody(io.NopCloser(strings.NewReader("ram data")), 512, false)
	require.NoError(t, err)
	mrbRAMTyped := mrbRAM.(*MultiReadBody)
	mrbRAMTyped.ReallyClose()
	mrbRAMTyped.ReallyClose() // idempotent, does not panic

	var buf [10]byte
	_, rErr := mrbRAMTyped.Read(buf[:])
	assert.ErrorIs(t, rErr, os.ErrClosed)

	cErr := mrbRAMTyped.Close()
	assert.ErrorIs(t, cErr, os.ErrClosed)
}

func TestBufioReadCloser_EdgeCases(t *testing.T) {
	t.Parallel()

	src := strings.NewReader("sample payload")
	closer := &mockTrackedCloser{Reader: src}
	brc := NewBufioReadCloser(src, closer)

	// Close first time
	require.NoError(t, brc.Close())
	assert.True(t, closer.closed)

	// Read after Close -> returns 0, io.EOF
	var buf [10]byte
	n, err := brc.Read(buf[:])
	assert.Equal(t, 0, n)
	assert.ErrorIs(t, err, io.EOF)

	// Peek after Close -> returns nil, io.EOF
	peek, err := brc.Peek(4)
	assert.Nil(t, peek)
	assert.ErrorIs(t, err, io.EOF)

	// Close second time -> returns nil
	brc.Closer = nil
	assert.NoError(t, brc.Close())

	// ReleaseBufioReader(nil) should not panic
	ReleaseBufioReader(nil)
}

func TestBufferedConn_BufferedReadBranch(t *testing.T) {
	t.Parallel()

	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	go func() {
		_, _ = c1.Write([]byte("peeked_and_buffered_data"))
	}()

	br := bufio.NewReader(c2)
	// Fill buffer so br.Buffered() > 0
	peek, err := br.Peek(6)
	require.NoError(t, err)
	assert.Equal(t, "peeked", string(peek))
	assert.True(t, br.Buffered() > 0)

	bConn := &BufferedConn{
		Conn: c2,
		R:    br,
	}

	buf := make([]byte, 6)
	n, err := bConn.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 6, n)
	assert.Equal(t, "peeked", string(buf))
}
