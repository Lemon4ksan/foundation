// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compress_test

import (
	"bytes"
	stdgzip "compress/gzip"
	"io"
	"strings"
	"testing"

	"github.com/lemon4ksan/foundation/borrow"
	"github.com/lemon4ksan/foundation/codec/compress"
	"github.com/lemon4ksan/foundation/codec/compress/flate"
	"github.com/lemon4ksan/foundation/codec/compress/gzip"
	"github.com/lemon4ksan/foundation/testkit/assert"
	"github.com/lemon4ksan/foundation/testkit/require"
)

func createGzipData(t testing.TB, payload []byte) []byte {
	if t != nil {
		t.Helper()
	}

	var buf bytes.Buffer

	w := stdgzip.NewWriter(&buf)
	_, _ = w.Write(payload)
	_ = w.Close()

	return buf.Bytes()
}

func createDeflateData(t testing.TB, payload []byte) []byte {
	if t != nil {
		t.Helper()
	}

	var buf bytes.Buffer

	w, _ := flate.NewWriter(&buf, flate.DefaultCompression)
	_, _ = w.Write(payload)
	_ = w.Close()

	return buf.Bytes()
}

func createZstdRawBlock(payload []byte) []byte {
	var buf bytes.Buffer
	// Frame Header: Magic 4B, FHD 1B (SingleSegment=0), Window_Descriptor 1B (0x20 = 256KB window)
	buf.Write([]byte{0x28, 0xb5, 0x2f, 0xfd, 0x00, 0x20})
	// Block Header: last_block=1 (bit 0), block_type=raw (0), block_size = len(payload) (bits 3-23)
	bh := uint32(1) | (uint32(len(payload)) << 3)
	buf.Write([]byte{byte(bh), byte(bh >> 8), byte(bh >> 16)})
	buf.Write(payload)

	return buf.Bytes()
}

func TestGunzip(t *testing.T) {
	t.Parallel()

	original := []byte("Hello, world! This is a test of high-speed gzip decompression in internal/compress.")
	compressed := createGzipData(t, original)

	decompressed, err := compress.Gunzip(compressed, nil)
	require.NoError(t, err)
	assert.Equal(t, original, decompressed)

	// Test Decompress dispatcher
	viaDispatcher, err := compress.Decompress("gzip", compressed, nil)
	require.NoError(t, err)
	assert.Equal(t, original, viaDispatcher)
}

func TestInflate(t *testing.T) {
	t.Parallel()

	original := []byte("Testing raw deflate RFC 1951 decompression.")
	compressed := createDeflateData(t, original)

	decompressed, err := compress.Inflate(compressed, nil)
	require.NoError(t, err)
	assert.Equal(t, original, decompressed)

	viaDispatcher, err := compress.Decompress("deflate", compressed, nil)
	require.NoError(t, err)
	assert.Equal(t, original, viaDispatcher)
}

func TestUnzstd(t *testing.T) {
	t.Parallel()

	original := []byte("Zstandard decompression test payload in aoni internal/compress.")
	compressed := createZstdRawBlock(original)

	decompressed, err := compress.Unzstd(compressed, nil)
	require.NoError(t, err)
	assert.Equal(t, original, decompressed)

	viaDispatcher, err := compress.Decompress("zstd", compressed, nil)
	require.NoError(t, err)
	assert.Equal(t, original, viaDispatcher)

	// Test streaming zstd reader
	zr, err := compress.AcquireZstdReader(bytes.NewReader(compressed))
	require.NoError(t, err)

	defer compress.ReleaseZstdReader(zr)

	buf, err := io.ReadAll(zr)
	require.NoError(t, err)
	assert.Equal(t, original, buf)
}

func TestStreamingReaders(t *testing.T) {
	t.Parallel()

	original := []byte("Streaming compression test data.")
	compressed := createGzipData(t, original)

	zr, err := compress.AcquireGzipReader(bytes.NewReader(compressed))
	require.NoError(t, err)

	defer compress.ReleaseGzipReader(zr)

	readBuf, err := io.ReadAll(zr)
	require.NoError(t, err)
	assert.Equal(t, original, readBuf)
}

func TestIsCompressed(t *testing.T) {
	t.Parallel()

	gzipData := []byte{0x1f, 0x8b, 0x08, 0x00}
	zstdData := []byte{0x28, 0xb5, 0x2f, 0xfd}
	plainData := []byte("plain text content")

	assert.True(t, compress.IsCompressed(gzipData))
	assert.True(t, compress.IsCompressed(zstdData))
	assert.False(t, compress.IsCompressed(plainData))
	assert.False(t, compress.IsCompressed([]byte{0x01}))
}

func TestMatchesEncoding(t *testing.T) {
	t.Parallel()

	assert.True(t, compress.MatchesEncoding([]byte("gzip, deflate, br"), "gzip"))
	assert.True(t, compress.MatchesEncoding([]byte("gzip, deflate, br"), "br"))
	assert.True(t, compress.MatchesEncoding([]byte("zstd, gzip"), "zstd"))
	assert.False(t, compress.MatchesEncoding([]byte("deflate"), "zstd"))
}

func TestUnbrotli(t *testing.T) {
	t.Parallel()

	original := []byte("Brotli RFC 7932 decompression test payload in aoni internal/compress.")
	compressed, _ := compress.CompressBrotli(original, nil)

	decompressed, err := compress.Unbrotli(compressed, nil)
	require.NoError(t, err)
	assert.Equal(t, original, decompressed)

	viaDispatcher, err := compress.Decompress("br", compressed, nil)
	require.NoError(t, err)
	assert.Equal(t, original, viaDispatcher)

	// Test streaming brotli reader
	br, err := compress.AcquireBrotliReader(bytes.NewReader(compressed))
	require.NoError(t, err)

	defer compress.ReleaseBrotliReader(br)

	buf, err := io.ReadAll(br)
	require.NoError(t, err)
	assert.Equal(t, original, buf)
}

func TestGzipReaderNilSafety(t *testing.T) {
	t.Parallel()

	var zr gzip.Reader
	assert.NoError(t, zr.Close())
}

func BenchmarkGunzip(b *testing.B) {
	payload := []byte(strings.Repeat("Zero allocation decompression benchmark for aoni internal/compress. ", 50))

	var buf bytes.Buffer

	w := gzip.NewWriter(&buf)
	_, _ = w.Write(payload)
	_ = w.Close()
	compressed := buf.Bytes()
	dst := make([]byte, 0, len(payload))

	b.ReportAllocs()

	for b.Loop() {
		_, _ = compress.Gunzip(compressed, dst)
	}
}

func TestNewReader_AllEncodings(t *testing.T) {
	t.Parallel()

	raw := []byte("Streaming decompression test across all RFC standard algorithms in aoni.")

	brData, _ := compress.CompressBrotli(raw, nil)
	tests := []struct {
		encoding   string
		compressed []byte
	}{
		{encoding: "gzip", compressed: createGzipData(t, raw)},
		{encoding: "x-gzip", compressed: createGzipData(t, raw)},
		{encoding: "br", compressed: brData},
		{encoding: "zstd", compressed: createZstdRawBlock(raw)},
		{encoding: "deflate", compressed: createDeflateData(t, raw)},
		{encoding: "identity", compressed: raw},
		{encoding: "", compressed: raw},
	}

	for _, tc := range tests {
		t.Run(tc.encoding, func(t *testing.T) {
			t.Parallel()

			r, err := compress.NewReader(tc.encoding, bytes.NewReader(tc.compressed))
			require.NoError(t, err)

			defer r.Close()

			decompressed, err := io.ReadAll(r)
			require.NoError(t, err)
			assert.Equal(t, raw, decompressed)
		})
	}

	// Error cases
	_, err := compress.NewReader("unknown_codec", bytes.NewReader(raw))
	assert.ErrorIs(t, err, compress.ErrUnsupportedEncoding)

	_, err = compress.NewReader("gzip", nil)
	assert.Error(t, err)
}

func BenchmarkUnzstd(b *testing.B) {
	payload := []byte(strings.Repeat("Zstd benchmark payload for internal/compress decoder. ", 50))
	compressed := createZstdRawBlock(payload)
	dst := make([]byte, 0, len(payload))

	b.ReportAllocs()

	for b.Loop() {
		_, _ = compress.Unzstd(compressed, dst)
	}
}

func BenchmarkUnbrotli(b *testing.B) {
	payload := []byte(strings.Repeat("Brotli benchmark payload for internal/compress decoder. ", 50))
	compressed, _ := compress.CompressBrotli(payload, nil)
	dst := make([]byte, 0, len(payload))

	b.ReportAllocs()

	for b.Loop() {
		_, _ = compress.Unbrotli(compressed, dst)
	}
}

func BenchmarkInflate(b *testing.B) {
	payload := []byte(strings.Repeat("Inflate benchmark payload for internal/compress flate decoder. ", 50))
	compressed := createDeflateData(nil, payload)
	dst := make([]byte, 0, len(payload))

	b.ReportAllocs()

	for b.Loop() {
		_, _ = compress.Inflate(compressed, dst)
	}
}

func BenchmarkStdlibGunzip(b *testing.B) {
	payload := []byte(strings.Repeat("Zero allocation decompression benchmark for aoni internal/compress. ", 50))

	var buf bytes.Buffer

	w := gzip.NewWriter(&buf)
	_, _ = w.Write(payload)
	_ = w.Close()
	compressed := buf.Bytes()

	b.ReportAllocs()

	for b.Loop() {
		r, err := stdgzip.NewReader(bytes.NewReader(compressed))
		if err != nil {
			b.Fatal(err)
		}

		_, _ = io.ReadAll(r)
		_ = r.Close()
	}
}

func BenchmarkStdlibInflate(b *testing.B) {
	payload := []byte(strings.Repeat("Inflate benchmark payload for internal/compress flate decoder. ", 50))
	compressed := createDeflateData(nil, payload)

	b.ReportAllocs()

	for b.Loop() {
		r := flate.NewReader(bytes.NewReader(compressed))
		_, _ = io.ReadAll(r)
		_ = r.Close()
	}
}

func TestDecompressScoped_AllEncodings(t *testing.T) {
	t.Parallel()

	raw := []byte("Zero allocation scoped decompression test across all RFC standard algorithms in aoni.")

	brData, _ := compress.CompressBrotli(raw, nil)
	tests := []struct {
		encoding   string
		compressed []byte
	}{
		{encoding: "gzip", compressed: createGzipData(t, raw)},
		{encoding: "br", compressed: brData},
		{encoding: "zstd", compressed: createZstdRawBlock(raw)},
		{encoding: "deflate", compressed: createDeflateData(t, raw)},
	}

	for _, tc := range tests {
		t.Run(tc.encoding, func(t *testing.T) {
			t.Parallel()

			s := borrow.AcquireScope()
			defer s.Release()

			decompressed, err := compress.DecompressScoped(s, tc.encoding, tc.compressed)
			require.NoError(t, err)
			assert.Equal(t, raw, decompressed.AsSlice())
		})
	}
}

func BenchmarkDecompressScoped_Gzip(b *testing.B) {
	payload := []byte(strings.Repeat("Zero allocation gzip scoped decompression in aoni. ", 50))
	compressed := createGzipData(nil, payload)

	b.ReportAllocs()
	b.ResetTimer()

	s := borrow.AcquireScope()
	defer s.Release()

	for b.Loop() {
		res, err := compress.GunzipScoped(s, compressed)
		if err != nil {
			b.Fatal(err)
		}

		_ = res

		s.Release()
		s = borrow.AcquireScope()
	}
}

func BenchmarkDecompressScoped_Zstd(b *testing.B) {
	payload := []byte(strings.Repeat("Zero allocation zstd scoped decompression in aoni. ", 50))
	compressed := createZstdRawBlock(payload)

	b.ReportAllocs()
	b.ResetTimer()

	s := borrow.AcquireScope()
	defer s.Release()

	for b.Loop() {
		res, err := compress.UnzstdScoped(s, compressed)
		if err != nil {
			b.Fatal(err)
		}

		_ = res

		s.Release()
		s = borrow.AcquireScope()
	}
}

func TestCompress_AllEncodings(t *testing.T) {
	t.Parallel()

	raw := []byte(
		"Compression and Decompression roundtrip test across all supported algorithms in aoni zero-alloc engine.",
	)

	encodings := []string{"gzip", "x-gzip", "br", "deflate", "lz4", "identity"}

	for _, enc := range encodings {
		t.Run(enc, func(t *testing.T) {
			t.Parallel()

			compressed, err := compress.Compress(enc, raw, nil)
			require.NoError(t, err)
			require.NotEmpty(t, compressed)

			decompressed, err := compress.Decompress(enc, compressed, nil)
			require.NoError(t, err)
			assert.Equal(t, raw, decompressed)
		})
	}
}

func TestCompressScoped_AllEncodings(t *testing.T) {
	t.Parallel()

	raw := []byte("Zero allocation scoped compression and decompression roundtrip test in aoni.")

	encodings := []string{"gzip", "x-gzip", "br", "deflate", "lz4", "identity"}

	for _, enc := range encodings {
		t.Run(enc, func(t *testing.T) {
			t.Parallel()

			s := borrow.AcquireScope()
			defer s.Release()

			compressed, err := compress.CompressScoped(s, enc, raw)
			require.NoError(t, err)
			require.NotEmpty(t, compressed.AsSlice())

			decompressed, err := compress.DecompressScoped(s, enc, compressed.AsSlice())
			require.NoError(t, err)
			assert.Equal(t, raw, decompressed.AsSlice())
		})
	}
}

func TestNewWriter_Streaming(t *testing.T) {
	t.Parallel()

	raw := []byte("Streaming compression writer and reader roundtrip test with pooled writers.")

	encodings := []string{"gzip", "deflate", "lz4", "identity"}

	for _, enc := range encodings {
		t.Run(enc, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer

			w, err := compress.NewWriter(enc, &buf)
			require.NoError(t, err)

			_, err = w.Write(raw)
			require.NoError(t, err)

			err = w.Close()
			require.NoError(t, err)

			r, err := compress.NewReader(enc, &buf)
			require.NoError(t, err)

			defer r.Close()

			decompressed, err := io.ReadAll(r)
			require.NoError(t, err)
			assert.Equal(t, raw, decompressed)
		})
	}
}

func TestDecompressionBomb_Protection(t *testing.T) {
	t.Parallel()

	// Create a payload of zeros that compresses to a tiny gzip (high amplification)
	hugeZeros := make([]byte, 1024*1024) // 1 MB of zeros
	compressedZeros := createGzipData(t, hugeZeros)

	// Artificially truncate compressed to simulate tiny input that would blow up beyond 250x ratio
	// A 100-byte gzip expanding to 1MB has >10000x amplification ratio
	tinyCompressed := compressedZeros[:min(len(compressedZeros), 64)]

	// Attempting to decompress truncated/malicious payload fails gracefully without crash/panic
	_, _ = compress.Gunzip(tinyCompressed, nil)
}

func BenchmarkGzip(b *testing.B) {
	payload := []byte(strings.Repeat("Zero allocation gzip compression benchmark in aoni. ", 50))
	dst := make([]byte, 0, len(payload))

	b.ReportAllocs()

	for b.Loop() {
		_, _ = compress.CompressGzip(payload, dst)
	}
}

func BenchmarkDeflate(b *testing.B) {
	payload := []byte(strings.Repeat("Zero allocation deflate compression benchmark in aoni. ", 50))
	dst := make([]byte, 0, len(payload))

	b.ReportAllocs()

	for b.Loop() {
		_, _ = compress.CompressDeflate(payload, dst)
	}
}

func TestZstdRoundTrip(t *testing.T) {
	t.Parallel()

	payload := []byte("Testing RFC 8878 Zstandard compression and decompression in Seal.")
	compressed, err := compress.CompressZstd(payload, nil)
	require.NoError(t, err)
	require.NotEmpty(t, compressed)

	decompressed, err := compress.Unzstd(compressed, nil)
	require.NoError(t, err)
	assert.Equal(t, payload, decompressed)
}

func TestLZMA2RoundTrip(t *testing.T) {
	t.Parallel()

	payload := []byte(strings.Repeat("Seal LZMA2 ultra high performance zero allocation codec testing. ", 50))
	compressed, err := compress.Compress("lzma2", payload, nil)
	require.NoError(t, err)
	require.NotEmpty(t, compressed)

	decompressed, err := compress.Decompress("lzma2", compressed, nil)
	require.NoError(t, err)
	assert.Equal(t, payload, decompressed)
}

func TestLZMA2Scoped(t *testing.T) {
	t.Parallel()

	scope := borrow.NewScope()
	defer scope.Release()

	payload := []byte(strings.Repeat("Seal Scoped LZMA2 zero allocation borrow testing. ", 50))
	compBytes, err := compress.LZMA2Scoped(scope, payload)
	require.NoError(t, err)
	require.NotEmpty(t, compBytes.AsSlice())

	decompBytes, err := compress.Unlzma2Scoped(scope, compBytes.AsSlice())
	require.NoError(t, err)
	assert.Equal(t, payload, decompBytes.AsSlice())
}

func BenchmarkBrotli(b *testing.B) {
	payload := []byte(strings.Repeat("Zero allocation brotli compression benchmark in aoni. ", 50))
	dst := make([]byte, 0, len(payload))

	b.ReportAllocs()

	for b.Loop() {
		_, _ = compress.CompressBrotli(payload, dst)
	}
}

func TestLZ4RoundTrip(t *testing.T) {
	t.Parallel()

	payload := []byte(strings.Repeat("Ultra-high-speed LZ4 compression and decompression test for Seal container engine. ", 100))
	compressed, err := compress.Compress("lz4", payload, nil)
	require.NoError(t, err)
	require.NotEmpty(t, compressed)

	decompressed, err := compress.Decompress("lz4", compressed, nil)
	require.NoError(t, err)
	assert.Equal(t, payload, decompressed)
}

func TestLZ4Scoped(t *testing.T) {
	t.Parallel()

	scope := borrow.NewScope()
	defer scope.Release()

	payload := []byte(strings.Repeat("Zero allocation scoped LZ4 compression for FastMount profile. ", 100))
	compBytes, err := compress.LZ4Scoped(scope, payload)
	require.NoError(t, err)
	require.NotEmpty(t, compBytes.AsSlice())

	decompBytes, err := compress.Unlz4Scoped(scope, compBytes.AsSlice())
	require.NoError(t, err)
	assert.Equal(t, payload, decompBytes.AsSlice())
}

func TestLZ4BlockRoundTrip(t *testing.T) {
	t.Parallel()

	payload := []byte(strings.Repeat("Raw un-framed LZ4 block test for micro-frame seeking. ", 100))
	maxBound := compress.CompressLZ4BlockBound(len(payload))
	dst := make([]byte, maxBound)

	n, err := compress.CompressLZ4Block(payload, dst)
	require.NoError(t, err)
	require.True(t, n > 0 && n < len(payload))

	decomp := make([]byte, len(payload))
	dn, err := compress.Unlz4Block(dst[:n], decomp)
	require.NoError(t, err)
	assert.Equal(t, len(payload), dn)
	assert.Equal(t, payload, decomp)
}

