// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package simd_test

import (
	"bytes"
	"math/rand/v2"
	"sync"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/silicon/simd"
	"github.com/lemon4ksan/foundation/testing/assert"
)

// TestChallenge_CRLFCRLF_ShortAndBoundary_Revalidation specifically tests the failure cases
// that previously failed under GOEXPERIMENT="none" (off-by-4 defect).
func TestChallenge_CRLFCRLF_ShortAndBoundary_Revalidation(t *testing.T) {
	testCases := []struct {
		name        string
		data        []byte
		expectedEnd int
		expectedRem string
	}{
		{
			name:        "minimal_CRLFCRLF",
			data:        []byte("\r\n\r\n"),
			expectedEnd: 4,
			expectedRem: "",
		},
		{
			name:        "minimal_CRLFCRLF_with_body",
			data:        []byte("\r\n\r\nBODY"),
			expectedEnd: 4,
			expectedRem: "BODY",
		},
		{
			name:        "short_HTTP_under_32_bytes",
			data:        []byte("OK\r\n\r\nHELLO"),
			expectedEnd: 6,
			expectedRem: "HELLO",
		},
		{
			name:        "exactly_30_bytes_total_with_CRLFCRLF",
			data:        []byte("012345678901234567890\r\n\r\nHELLO"), // 21 + 4 + 5 = 30 bytes
			expectedEnd: 25,
			expectedRem: "HELLO",
		},
		{
			name:        "exactly_31_bytes_total_with_CRLFCRLF",
			data:        []byte("0123456789012345678901\r\n\r\nHELLO"), // 22 + 4 + 5 = 31 bytes
			expectedEnd: 26,
			expectedRem: "HELLO",
		},
		{
			name:        "exactly_32_bytes_total_with_CRLFCRLF",
			data:        []byte("01234567890123456789012\r\n\r\nHELLO"), // 23 + 4 + 5 = 32 bytes
			expectedEnd: 27,
			expectedRem: "HELLO",
		},
		{
			name:        "33_bytes_total_with_CRLFCRLF",
			data:        []byte("012345678901234567890123\r\n\r\nHELLO"), // 24 + 4 + 5 = 33 bytes
			expectedEnd: 28,
			expectedRem: "HELLO",
		},
		{
			name:        "exactly_31_bytes_ending_with_CRLFCRLF",
			data:        append(bytes.Repeat([]byte("A"), 27), []byte("\r\n\r\n")...),
			expectedEnd: 31,
			expectedRem: "",
		},
		{
			name:        "exactly_32_bytes_ending_with_CRLFCRLF",
			data:        append(bytes.Repeat([]byte("A"), 28), []byte("\r\n\r\n")...),
			expectedEnd: 32,
			expectedRem: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			refIdx := simd.IndexCRLFCRLF(tc.data)
			vecIdx := simd.IndexCRLFCRLFVector(tc.data)

			assert.Equal(t, tc.expectedEnd, refIdx, "IndexCRLFCRLF expected index")
			assert.Equal(t, tc.expectedEnd, vecIdx, "IndexCRLFCRLFVector expected index")
			assert.Equal(t, refIdx, vecIdx, "equivalence between Vector and scalar IndexCRLFCRLF")

			if vecIdx >= 0 {
				rem := string(tc.data[vecIdx:])
				assert.Equal(t, tc.expectedRem, rem, "remaining body slice")
			}
		})
	}
}

// TestChallenge_LegacyLFLF_NoOvershoot_NoPanic rigorously verifies that legacy \n\n boundaries
// do NOT overshoot by 2 bytes (or 4 bytes) and never panic out-of-bounds.
func TestChallenge_LegacyLFLF_NoOvershoot_NoPanic(t *testing.T) {
	// 1. Discrete short cases
	shortCases := []struct {
		name        string
		data        []byte
		expectedEnd int
		expectedRem string
	}{
		{"minimal_LFLF", []byte("\n\n"), 2, ""},
		{"minimal_LFLF_with_body", []byte("\n\nBODY"), 2, "BODY"},
		{"one_char_prefix_LFLF", []byte("A\n\n"), 3, ""},
		{"one_char_prefix_LFLF_body", []byte("A\n\nB"), 3, "B"},
		{"two_char_prefix_LFLF", []byte("AB\n\n"), 4, ""},
		{"three_char_prefix_LFLF", []byte("ABC\n\n"), 5, ""},
		{"prefix_LFLF_30_bytes", append(bytes.Repeat([]byte("X"), 28), []byte("\n\n")...), 30, ""},
		{"prefix_LFLF_31_bytes", append(bytes.Repeat([]byte("X"), 29), []byte("\n\n")...), 31, ""},
		{"prefix_LFLF_32_bytes", append(bytes.Repeat([]byte("X"), 30), []byte("\n\n")...), 32, ""},
		{"prefix_LFLF_33_bytes", append(bytes.Repeat([]byte("X"), 31), []byte("\n\n")...), 33, ""},
		{"prefix_LFLF_64_bytes", append(bytes.Repeat([]byte("X"), 62), []byte("\n\n")...), 64, ""},
	}

	for _, tc := range shortCases {
		t.Run(tc.name, func(t *testing.T) {
			refIdx := simd.IndexCRLFCRLF(tc.data)
			vecIdx := simd.IndexCRLFCRLFVector(tc.data)

			assert.Equal(t, tc.expectedEnd, refIdx)
			assert.Equal(t, tc.expectedEnd, vecIdx)

			// Strict 3-index capacity bound: if vecIdx > len, slice expression panics immediately
			bounded := tc.data[:len(tc.data):len(tc.data)]
			rem := string(bounded[vecIdx:])
			assert.Equal(t, tc.expectedRem, rem)
		})
	}

	// 2. Exhaustive test: \n\n placed at the EXACT END of every length from 2 to 1024
	for l := 2; l <= 1024; l++ {
		raw := make([]byte, l)
		for i := 0; i < l-2; i++ {
			raw[i] = 'H'
		}
		raw[l-2] = '\n'
		raw[l-1] = '\n'

		// Bound capacity strictly to l
		strictSlice := raw[:l:l]

		ref := simd.IndexCRLFCRLF(strictSlice)
		got := simd.IndexCRLFCRLFVector(strictSlice)

		if got != l {
			t.Fatalf("length %d ending in \\n\\n: expected %d, got %d", l, l, got)
		}
		if ref != got {
			t.Fatalf("length %d mismatch: ref=%d, got=%d", l, ref, got)
		}

		// Slicing at got: must produce exactly empty slice without panic
		rem := strictSlice[got:]
		if len(rem) != 0 {
			t.Fatalf("length %d: expected remaining slice of len 0, got %d", l, len(rem))
		}
	}

	// 3. Delimiter priority: CRLFCRLF vs LFLF order
	t.Run("CRLFCRLF_before_LFLF", func(t *testing.T) {
		data := []byte("Header: val\r\n\r\nBody\n\nMore")
		ref := simd.IndexCRLFCRLF(data)
		got := simd.IndexCRLFCRLFVector(data)
		assert.Equal(t, 15, ref)
		assert.Equal(t, ref, got)
		assert.Equal(t, "Body\n\nMore", string(data[got:]))
	})

	t.Run("LFLF_before_CRLFCRLF", func(t *testing.T) {
		// IndexCRLFCRLF checks bytes.Index(crlfCrlfBytes) FIRST across the entire buffer!
		// If crlfCrlfBytes exists later, IndexCRLFCRLF returns crlfCrlfBytes position.
		// Vector MUST match this exact specification.
		data := []byte("Header: val\n\nBody\r\n\r\nMore")
		ref := simd.IndexCRLFCRLF(data)
		got := simd.IndexCRLFCRLFVector(data)
		assert.Equal(t, ref, got)
	})
}

// TestChallenge_Adversarial_FullExhaustiveSweep tests every possible position in buffers 0 to 1024.
func TestChallenge_Adversarial_FullExhaustiveSweep(t *testing.T) {
	lengthsToTest := []int{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 15, 16, 17, 31, 32, 33, 34,
		47, 48, 63, 64, 65, 95, 96, 97, 127, 128, 129, 255, 256, 257,
		511, 512, 513, 1023, 1024,
	}

	for _, l := range lengthsToTest {
		// 1. Clean buffer (no delimiters)
		clean := make([]byte, l)
		for i := range clean {
			clean[i] = 'Z'
		}
		refClean := simd.IndexCRLFCRLF(clean[:l:l])
		vecClean := simd.IndexCRLFCRLFVector(clean[:l:l])
		if refClean != vecClean {
			t.Fatalf("clean len %d: ref=%d, vec=%d", l, refClean, vecClean)
		}

		// 2. CRLFCRLF at every single valid index
		if l >= 4 {
			for pos := 0; pos <= l-4; pos++ {
				buf := make([]byte, l)
				for i := range buf {
					buf[i] = '.'
				}
				copy(buf[pos:], "\r\n\r\n")

				ref := simd.IndexCRLFCRLF(buf[:l:l])
				got := simd.IndexCRLFCRLFVector(buf[:l:l])
				if ref != got {
					t.Fatalf("len %d pos %d CRLFCRLF: ref=%d, got=%d", l, pos, ref, got)
				}
				if got != pos+4 {
					t.Fatalf("len %d pos %d CRLFCRLF: expected pos+4=%d, got=%d", l, pos, pos+4, got)
				}
				_ = buf[got:] // must not panic
			}
		}

		// 3. LFLF at every single valid index
		if l >= 2 {
			for pos := 0; pos <= l-2; pos++ {
				buf := make([]byte, l)
				for i := range buf {
					buf[i] = '.'
				}
				copy(buf[pos:], "\n\n")

				ref := simd.IndexCRLFCRLF(buf[:l:l])
				got := simd.IndexCRLFCRLFVector(buf[:l:l])
				if ref != got {
					t.Fatalf("len %d pos %d LFLF: ref=%d, got=%d", l, pos, ref, got)
				}
				if got != pos+2 {
					t.Fatalf("len %d pos %d LFLF: expected pos+2=%d, got=%d", l, pos, pos+2, got)
				}
				_ = buf[got:] // must not panic
			}
		}
	}
}

// TestChallenge_IndexCRLFCRLFVector_HighConcurrency_Race runs concurrent differential fuzzing.
func TestChallenge_IndexCRLFCRLFVector_HighConcurrency_Race(t *testing.T) {
	const numGoroutines = 64
	const itersPerGoroutine = 500

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for g := range numGoroutines {
		go func(gID int) {
			defer wg.Done()
			rng := rand.New(rand.NewPCG(uint64(gID+555), uint64(time.Now().UnixNano())))

			for it := range itersPerGoroutine {
				l := rng.IntN(1025)
				buf := make([]byte, l)

				// Fill buffer with mix of ASCII and delimiters
				patternChoice := rng.IntN(5)
				switch patternChoice {
				case 0:
					// Pure random bytes
					for i := range buf {
						buf[i] = byte(rng.IntN(256))
					}
				case 1:
					// High density of CR and LF
					for i := range buf {
						switch rng.IntN(4) {
						case 0:
							buf[i] = '\r'
						case 1:
							buf[i] = '\n'
						default:
							buf[i] = byte('A' + rng.IntN(26))
						}
					}
				case 2:
					// HTTP style text with occasional CRLFCRLF injected
					for i := range buf {
						buf[i] = 'a'
					}
					if l >= 4 {
						pos := rng.IntN(l - 3)
						copy(buf[pos:], "\r\n\r\n")
					}
				case 3:
					// HTTP style text with legacy \n\n injected
					for i := range buf {
						buf[i] = 'b'
					}
					if l >= 2 {
						pos := rng.IntN(l - 1)
						copy(buf[pos:], "\n\n")
					}
				case 4:
					// Near boundary offsets (28 to 36)
					for i := range buf {
						buf[i] = 'c'
					}
					if l >= 36 {
						offset := 28 + rng.IntN(8)
						if rng.IntN(2) == 0 {
							copy(buf[offset:], "\r\n\r\n")
						} else {
							copy(buf[offset:], "\n\n")
						}
					}
				}

				strictSlice := buf[:l:l]
				expected := simd.IndexCRLFCRLF(strictSlice)
				got := simd.IndexCRLFCRLFVector(strictSlice)

				if expected != got {
					t.Errorf(
						"g=%d it=%d len=%d pattern=%d mismatch: expected=%d got=%d (data=%q)",
						gID,
						it,
						l,
						patternChoice,
						expected,
						got,
						strictSlice,
					)
					return
				}

				if got >= 0 {
					rem := strictSlice[got:]
					_ = rem // ensure valid slice
				}
			}
		}(g)
	}

	wg.Wait()
}
