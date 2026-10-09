// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gzip_test

import (
	"bytes"
	"crypto/rand"
	"hash/crc32"
	"io"
	"strings"
	"testing"

	"github.com/lemon4ksan/foundation/codec/compress/gzip"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

func TestGzipWriterAndReader(t *testing.T) {
	t.Parallel()

	data := []byte("Testing standalone gzip reader and writer compression in aoni.")

	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	w.Name = "test.txt"
	w.Comment = "unit test"

	n, err := w.Write(data)
	require.NoError(t, err)
	assert.Equal(t, len(data), n)

	err = w.Close()
	require.NoError(t, err)

	r, err := gzip.NewReader(&buf)
	require.NoError(t, err)
	assert.Equal(t, "test.txt", r.Name)
	assert.Equal(t, "unit test", r.Comment)

	decompressed, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, data, decompressed)

	err = r.Close()
	require.NoError(t, err)
}

func TestGzipWriterLevels(t *testing.T) {
	t.Parallel()

	data := []byte(strings.Repeat("Gzip compression levels test content. ", 30))

	levels := []int{
		gzip.BestSpeed,
		gzip.BestCompression,
		gzip.DefaultCompression,
		gzip.HuffmanOnly,
		gzip.StatelessCompression,
	}

	for _, level := range levels {
		var buf bytes.Buffer
		w, err := gzip.NewWriterLevel(&buf, level)
		require.NoError(t, err)

		_, err = w.Write(data)
		require.NoError(t, err)
		require.NoError(t, w.Close())

		r, err := gzip.NewReader(&buf)
		require.NoError(t, err)

		decompressed, err := io.ReadAll(r)
		require.NoError(t, err)
		assert.Equal(t, data, decompressed)
	}

	// Invalid level
	_, err := gzip.NewWriterLevel(&bytes.Buffer{}, 99)
	assert.Error(t, err)
}
func TestCRC32_DifferentialAgainstStdlib(t *testing.T) {
	t.Parallel()

	// Validate small buffer edge cases and byte-aligned boundary conditions
	for size := 0; size <= 512; size++ {
		data := make([]byte, size)
		_, _ = rand.Read(data)

		expected := crc32.ChecksumIEEE(data)
		actual := gzip.CRC32Update(0, data)

		assert.Equal(t, expected, actual)
	}

	// Validate large chunk throughput and non-power-of-two unaligned buffer slices
	largeSizes := []int{64 * 1024, 1024*1024 + 3, 4*1024*1024 + 7}
	for _, size := range largeSizes {
		data := make([]byte, size)
		_, _ = rand.Read(data)

		expected := crc32.ChecksumIEEE(data)
		actual := gzip.CRC32Update(0, data)

		assert.Equal(t, expected, actual)
	}

	// Verify rolling state commutativity: crc(A + B + C) == crc(crc(crc(0, A), B), C)
	partA := []byte("The quick brown fox ")
	partB := []byte("jumps over ")
	partC := []byte("the lazy dog.")
	full := []byte("The quick brown fox jumps over the lazy dog.")

	expected := crc32.ChecksumIEEE(full)

	var running uint32
	running = gzip.CRC32Update(running, partA)
	running = gzip.CRC32Update(running, partB)
	running = gzip.CRC32Update(running, partC)

	assert.Equal(t, expected, running)

	// Fuzz test sliding windows and arbitrary subslice offsets
	buf := make([]byte, 4096)
	_, _ = rand.Read(buf)

	for i := range 5000 {
		start := i % 2048
		end := start + (i % 2000)
		sub := buf[start:end]

		want := crc32.ChecksumIEEE(sub)
		got := gzip.CRC32Update(0, sub)

		if want != got {
			t.Fatalf("mismatch on subslice [%d:%d]: want %08x, got %08x", start, end, want, got)
		}
	}
}
