// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compress_test

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"sync"
	"testing"

	"github.com/lemon4ksan/foundation/borrow"
	"github.com/lemon4ksan/foundation/codec/compress"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

// TestRefactoredCompression_ZstdBoundaries tests RFC 8878 Zstandard single-segment frame headers
// and raw block chunking across all boundary thresholds (1-byte FCS, 2-byte FCS, 4-byte FCS, and 128KB block boundaries).
func TestRefactoredCompression_ZstdBoundaries(t *testing.T) {
	t.Parallel()

	boundarySizes := []int{
		0,
		1,
		2,
		3,
		15,
		255, // 1-byte FCS max
		256, // 2-byte FCS min (fcs = 0)
		257,
		1024,
		65535,
		65536,
		65791, // 2-byte FCS max (65536 + 255)
		65792, // 4-byte FCS min (65536 + 256)
		131071,
		131072, // 128 KB single block boundary
		131073, // 128 KB + 1 (spans 2 blocks)
		262144, // 256 KB (spans exactly 2 blocks)
		262145, // spans 3 blocks
		300000,
	}

	for _, sz := range boundarySizes {
		sz := sz
		t.Run(fmt.Sprintf("Size_%d", sz), func(t *testing.T) {
			t.Parallel()

			payload := make([]byte, sz)
			for i := range payload {
				payload[i] = byte(i*31 + 7)
			}

			// 1. Direct Zstd / Unzstd
			compressed, err := compress.Zstd(payload, nil)
			require.NoError(t, err)
			if sz > 0 {
				require.NotEmpty(t, compressed)
			}

			decompressed, err := compress.Unzstd(compressed, nil)
			if sz == 0 {
				assert.Empty(t, decompressed)
			} else {
				require.NoError(t, err)
				assert.Equal(t, payload, decompressed)
			}

			// 2. Dispatcher Decompress
			viaDecomp, err := compress.Decompress("zstd", compressed, nil)
			require.NoError(t, err)
			if sz == 0 {
				assert.Empty(t, viaDecomp)
			} else {
				assert.Equal(t, payload, viaDecomp)
			}

			// 3. Preallocated dst buffer with zero initial length
			dstBuf := make([]byte, 0, len(compressed))
			compressedDst, err := compress.Zstd(payload, dstBuf)
			require.NoError(t, err)
			assert.Equal(t, compressed, compressedDst)

			// 4. Scoped variants
			scope := borrow.AcquireScope()
			defer scope.Release()

			scopedComp, err := compress.ZstdScoped(scope, payload)
			require.NoError(t, err)
			if sz > 0 {
				scopedDecomp, err := compress.UnzstdScoped(scope, scopedComp.AsSlice())
				require.NoError(t, err)
				assert.Equal(t, payload, scopedDecomp.AsSlice())
			}
		})
	}
}

// TestRefactoredCompression_MultiIterationStress runs multi-goroutine concurrent round-trips
// across all refactored entry points (Gzip, Deflate, Brotli, Zstd, LZMA2, LZ4, LZ4Block).
func TestRefactoredCompression_MultiIterationStress(t *testing.T) {
	t.Parallel()

	algorithms := []struct {
		name       string
		compress   func(src, dst []byte) ([]byte, error)
		decompress func(src, dst []byte) ([]byte, error)
	}{
		{
			name:       "Gzip",
			compress:   func(src, dst []byte) ([]byte, error) { return compress.Gzip(src, dst) },
			decompress: func(src, dst []byte) ([]byte, error) { return compress.Gunzip(src, dst) },
		},
		{
			name:       "Deflate",
			compress:   func(src, dst []byte) ([]byte, error) { return compress.Deflate(src, dst) },
			decompress: func(src, dst []byte) ([]byte, error) { return compress.Inflate(src, dst) },
		},
		{
			name:       "Brotli",
			compress:   func(src, dst []byte) ([]byte, error) { return compress.Brotli(src, dst) },
			decompress: func(src, dst []byte) ([]byte, error) { return compress.Unbrotli(src, dst) },
		},
		{
			name:       "Zstd",
			compress:   func(src, dst []byte) ([]byte, error) { return compress.Zstd(src, dst) },
			decompress: func(src, dst []byte) ([]byte, error) { return compress.Unzstd(src, dst) },
		},
		{
			name:       "LZMA2",
			compress:   func(src, dst []byte) ([]byte, error) { return compress.LZMA2(src, dst) },
			decompress: func(src, dst []byte) ([]byte, error) { return compress.Unlzma2(src, dst) },
		},
		{
			name:       "LZ4",
			compress:   func(src, dst []byte) ([]byte, error) { return compress.LZ4(src, dst) },
			decompress: func(src, dst []byte) ([]byte, error) { return compress.Unlz4(src, dst) },
		},
	}

	testPayloads := [][]byte{
		[]byte("Short test string."),
		bytes.Repeat([]byte("Repetitive payload for compression stress testing. "), 64),
		bytes.Repeat([]byte{0x00}, 2048),
		bytes.Repeat([]byte{0xAA, 0x55}, 1024),
	}

	for _, algo := range algorithms {
		algo := algo
		t.Run(algo.name, func(t *testing.T) {
			t.Parallel()

			const concurrency = 8
			const iterations = 10

			var wg sync.WaitGroup
			wg.Add(concurrency)

			for c := 0; c < concurrency; c++ {
				go func(workerID int) {
					defer wg.Done()
					for iter := 0; iter < iterations; iter++ {
						payload := testPayloads[(workerID+iter)%len(testPayloads)]

						// Fresh buffer
						comp, err := algo.compress(payload, nil)
						if err != nil {
							t.Errorf("%s compress error: %v", algo.name, err)
							return
						}
						decomp, err := algo.decompress(comp, nil)
						if err != nil {
							t.Errorf("%s decompress error: %v", algo.name, err)
							return
						}
						if !bytes.Equal(payload, decomp) {
							t.Errorf("%s payload mismatch", algo.name)
							return
						}

						// Reused destination buffer (zero length, adequate capacity)
						dstComp := make([]byte, 0, len(comp)+16)
						compReused, err := algo.compress(payload, dstComp)
						if err != nil {
							t.Errorf("%s compress with dst error: %v", algo.name, err)
							return
						}
						dstDecomp := make([]byte, 0, len(payload)+16)
						decompReused, err := algo.decompress(compReused, dstDecomp)
						if err != nil {
							t.Errorf("%s decompress with dst error: %v", algo.name, err)
							return
						}
						if !bytes.Equal(payload, decompReused) {
							t.Errorf("%s reused payload mismatch", algo.name)
							return
						}
					}
				}(c)
			}

			wg.Wait()
		})
	}
}

// TestRefactoredCompression_LZ4BlockStress verifies raw block LZ4 round-trip,
// LZ4BlockBound calculations, and buffer boundary constraints.
func TestRefactoredCompression_LZ4BlockStress(t *testing.T) {
	t.Parallel()

	sizes := []int{1, 4, 16, 64, 256, 1024, 4096, 65536}

	for _, sz := range sizes {
		sz := sz
		t.Run(fmt.Sprintf("Size_%d", sz), func(t *testing.T) {
			t.Parallel()

			payload := make([]byte, sz)
			for i := range payload {
				payload[i] = byte(i%251 + (i % 3))
			}

			bound := compress.LZ4BlockBound(len(payload))
			require.True(t, bound >= len(payload))

			dst := make([]byte, bound)
			n, err := compress.LZ4Block(payload, dst)
			require.NoError(t, err)
			require.True(t, n > 0 && n <= bound)

			decomp := make([]byte, len(payload))
			dn, err := compress.Unlz4Block(dst[:n], decomp)
			require.NoError(t, err)
			assert.Equal(t, len(payload), dn)
			assert.Equal(t, payload, decomp)

			// Destination buffer too small for decompressed output
			if sz > 8 {
				tooSmall := make([]byte, sz/2)
				_, err = compress.Unlz4Block(dst[:n], tooSmall)
				assert.Error(t, err)
			}
		})
	}
}

// TestRefactoredCompression_PathologicalPayloads evaluates handling of edge payloads:
// large all-zeros, large all-ones, high entropy noise, and alternating patterns.
func TestRefactoredCompression_PathologicalPayloads(t *testing.T) {
	t.Parallel()

	randData := make([]byte, 32768)
	_, _ = rand.Read(randData)

	testCases := []struct {
		name string
		data []byte
	}{
		{"AllZeros_64KB", bytes.Repeat([]byte{0x00}, 64*1024)},
		{"AllOnes_64KB", bytes.Repeat([]byte{0xFF}, 64*1024)},
		{"Alternating_64KB", bytes.Repeat([]byte{0xAA, 0x55}, 32*1024)},
		{"HighEntropy_32KB", randData},
		{"SingleByte_Zero", []byte{0x00}},
		{"SingleByte_One", []byte{0x01}},
		{"TwoBytes", []byte{0x42, 0x99}},
	}

	codecs := []string{"gzip", "deflate", "br", "zstd", "lz4"}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			for _, codec := range codecs {
				codec := codec
				t.Run(codec, func(t *testing.T) {
					compressed, err := compress.Compress(codec, tc.data, nil)
					require.NoError(t, err)

					decompressed, err := compress.Decompress(codec, compressed, nil)
					require.NoError(t, err)
					assert.Equal(t, tc.data, decompressed)
				})
			}
		})
	}
}

// TestRefactoredCompression_CorruptedPayloads ensures that random noise, truncated data,
// and bit-flipped streams fail gracefully with error, without panic or crash.
func TestRefactoredCompression_CorruptedPayloads(t *testing.T) {
	t.Parallel()

	validData := []byte("Valid baseline payload that will be corrupted to test decompressor resilience.")

	codecs := []struct {
		name       string
		compress   func([]byte) ([]byte, error)
		decompress func([]byte) ([]byte, error)
	}{
		{
			name:       "gzip",
			compress:   func(b []byte) ([]byte, error) { return compress.Gzip(b, nil) },
			decompress: func(b []byte) ([]byte, error) { return compress.Gunzip(b, nil) },
		},
		{
			name:       "deflate",
			compress:   func(b []byte) ([]byte, error) { return compress.Deflate(b, nil) },
			decompress: func(b []byte) ([]byte, error) { return compress.Inflate(b, nil) },
		},
		{
			name:       "br",
			compress:   func(b []byte) ([]byte, error) { return compress.Brotli(b, nil) },
			decompress: func(b []byte) ([]byte, error) { return compress.Unbrotli(b, nil) },
		},
		{
			name:       "zstd",
			compress:   func(b []byte) ([]byte, error) { return compress.Zstd(b, nil) },
			decompress: func(b []byte) ([]byte, error) { return compress.Unzstd(b, nil) },
		},
		{
			name:       "lz4",
			compress:   func(b []byte) ([]byte, error) { return compress.LZ4(b, nil) },
			decompress: func(b []byte) ([]byte, error) { return compress.Unlz4(b, nil) },
		},
		{
			name:       "lzma2",
			compress:   func(b []byte) ([]byte, error) { return compress.LZMA2(b, nil) },
			decompress: func(b []byte) ([]byte, error) { return compress.Unlzma2(b, nil) },
		},
	}

	for _, c := range codecs {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			validComp, err := c.compress(validData)
			require.NoError(t, err)

			// 1. Truncated payloads
			if len(validComp) > 4 {
				truncated := validComp[:len(validComp)/2]
				_, err = c.decompress(truncated)
				// Truncated data should return an error
				assert.Error(t, err)
			}

			// 2. Completely garbage random bytes
			garbage := []byte{0xDE, 0xAD, 0xBE, 0xEF, 0xCA, 0xFE, 0xBA, 0xBE, 0x01, 0x23, 0x45, 0x67}
			_, _ = c.decompress(garbage) // must not panic

			// 3. Bit flips in payload body
			if len(validComp) > 8 {
				corrupted := make([]byte, len(validComp))
				copy(corrupted, validComp)
				corrupted[len(corrupted)/2] ^= 0xFF
				_, _ = c.decompress(corrupted) // must not panic
			}
		})
	}
}

// TestRefactoredCompression_ScopedRoundTripStress validates scoped compression and decompression
// across all algorithms under repeated allocations and releases.
func TestRefactoredCompression_ScopedRoundTripStress(t *testing.T) {
	t.Parallel()

	data := []byte("Scoped compression engine test with memory arena isolation.")

	scope := borrow.AcquireScope()
	defer scope.Release()

	// Gzip scoped
	gzComp, err := compress.GzipScoped(scope, data)
	require.NoError(t, err)
	gzDecomp, err := compress.GunzipScoped(scope, gzComp.AsSlice())
	require.NoError(t, err)
	assert.Equal(t, data, gzDecomp.AsSlice())

	// Deflate scoped
	dfComp, err := compress.DeflateScoped(scope, data)
	require.NoError(t, err)
	dfDecomp, err := compress.InflateScoped(scope, dfComp.AsSlice())
	require.NoError(t, err)
	assert.Equal(t, data, dfDecomp.AsSlice())

	// Brotli scoped
	brComp, err := compress.BrotliScoped(scope, data)
	require.NoError(t, err)
	brDecomp, err := compress.UnbrotliScoped(scope, brComp.AsSlice())
	require.NoError(t, err)
	assert.Equal(t, data, brDecomp.AsSlice())

	// Zstd scoped
	zsComp, err := compress.ZstdScoped(scope, data)
	require.NoError(t, err)
	zsDecomp, err := compress.UnzstdScoped(scope, zsComp.AsSlice())
	require.NoError(t, err)
	assert.Equal(t, data, zsDecomp.AsSlice())

	// LZMA2 scoped
	lzmComp, err := compress.LZMA2Scoped(scope, data)
	require.NoError(t, err)
	lzmDecomp, err := compress.Unlzma2Scoped(scope, lzmComp.AsSlice())
	require.NoError(t, err)
	assert.Equal(t, data, lzmDecomp.AsSlice())

	// LZ4 scoped
	lz4Comp, err := compress.LZ4Scoped(scope, data)
	require.NoError(t, err)
	lz4Decomp, err := compress.Unlz4Scoped(scope, lz4Comp.AsSlice())
	require.NoError(t, err)
	assert.Equal(t, data, lz4Decomp.AsSlice())
}
