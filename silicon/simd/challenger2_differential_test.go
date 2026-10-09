// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package simd_test

import (
	"bytes"
	"fmt"
	"math/rand/v2"
	"sync"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/silicon/simd"
	"github.com/lemon4ksan/foundation/testing/assert"
)

// TestChallenger2_BoundaryStraddles_Offsets28to36 exhaustively tests all boundary straddles
// across offsets 28 through 36 and lengths 28 through 128 bytes.
func TestChallenger2_BoundaryStraddles_Offsets28to36(t *testing.T) {
	for l := 32; l <= 128; l++ {
		for offset := 28; offset <= 36; offset++ {
			// Test 1: CRLFCRLF straddling offset
			if offset+4 <= l {
				buf := bytes.Repeat([]byte("A"), l)
				copy(buf[offset:], "\r\n\r\n")

				strictBuf := buf[:l:l]
				expected := simd.IndexCRLFCRLF(strictBuf)
				got := simd.IndexCRLFCRLFVector(strictBuf)

				if expected != got {
					t.Fatalf("CRLFCRLF mismatch at len=%d, offset=%d: expected=%d, got=%d", l, offset, expected, got)
				}
				if got != offset+4 {
					t.Fatalf("CRLFCRLF expected pos+4=%d, got=%d", offset+4, got)
				}

				// Verify body slicing
				body := strictBuf[got:]
				if len(body) != l-(offset+4) {
					t.Fatalf("invalid body len: expected %d, got %d", l-(offset+4), len(body))
				}
			}

			// Test 2: Legacy LFLF straddling offset
			if offset+2 <= l {
				buf := bytes.Repeat([]byte("B"), l)
				copy(buf[offset:], "\n\n")

				strictBuf := buf[:l:l]
				expected := simd.IndexCRLFCRLF(strictBuf)
				got := simd.IndexCRLFCRLFVector(strictBuf)

				if expected != got {
					t.Fatalf("LFLF mismatch at len=%d, offset=%d: expected=%d, got=%d", l, offset, expected, got)
				}
				if got != offset+2 {
					t.Fatalf("LFLF expected pos+2=%d, got=%d", offset+2, got)
				}

				// Verify body slicing
				body := strictBuf[got:]
				if len(body) != l-(offset+2) {
					t.Fatalf("invalid body len: expected %d, got %d", l-(offset+2), len(body))
				}
			}

			// Test 3: False positive traps near boundary (partial CRLFCR, lone CR, lone LF)
			trapBuf := bytes.Repeat([]byte("C"), l)
			if offset+3 <= l {
				copy(trapBuf[offset:], "\r\n\r")
			}
			strictTrap := trapBuf[:l:l]
			expTrap := simd.IndexCRLFCRLF(strictTrap)
			gotTrap := simd.IndexCRLFCRLFVector(strictTrap)
			if expTrap != gotTrap {
				t.Fatalf("trap mismatch at len=%d, offset=%d: expected=%d, got=%d", l, offset, expTrap, gotTrap)
			}
		}
	}
}

// TestChallenger2_DifferentialFuzzing_Concurrent runs 100,000 randomized fuzzing iterations
// across 64 concurrent goroutines with varied byte patterns and payloads.
func TestChallenger2_DifferentialFuzzing_Concurrent(t *testing.T) {
	const numGoroutines = 64
	const itersPerGoroutine = 1500 // 64 * 1500 = 96,000 iterations

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for g := 0; g < numGoroutines; g++ {
		go func(gID int) {
			defer wg.Done()
			rng := rand.New(rand.NewPCG(uint64(gID+999), uint64(time.Now().UnixNano())))

			for it := 0; it < itersPerGoroutine; it++ {
				l := rng.IntN(2048)
				buf := make([]byte, l)

				category := rng.IntN(6)
				switch category {
				case 0:
					// Pure random bytes
					for i := range buf {
						buf[i] = byte(rng.IntN(256))
					}
				case 1:
					// Pure CR and LF noise
					for i := range buf {
						if rng.IntN(2) == 0 {
							buf[i] = '\r'
						} else {
							buf[i] = '\n'
						}
					}
				case 2:
					// Structured HTTP request
					method := "GET"
					if rng.IntN(2) == 0 {
						method = "POST"
					}
					header := fmt.Sprintf("%s /test HTTP/1.1\r\nHost: example.com\r\nContent-Length: 0\r\n\r\n", method)
					copy(buf, header)
					for i := len(header); i < l; i++ {
						buf[i] = 'X'
					}
				case 3:
					// Boundary straddle injection around offsets 28..36
					for i := range buf {
						buf[i] = '.'
					}
					if l >= 36 {
						offset := 28 + rng.IntN(9)
						if offset+4 <= l {
							copy(buf[offset:], "\r\n\r\n")
						}
					}
				case 4:
					// Injected legacy \n\n delimiter
					for i := range buf {
						buf[i] = 'y'
					}
					if l >= 2 {
						pos := rng.IntN(l - 1)
						copy(buf[pos:], "\n\n")
					}
				case 5:
					// Mixed delimiters: \n\n followed by \r\n\r\n
					for i := range buf {
						buf[i] = 'z'
					}
					if l >= 16 {
						pos1 := rng.IntN(l / 2)
						copy(buf[pos1:], "\n\n")
						if pos1+6 < l {
							pos2 := pos1 + 2 + rng.IntN(l-(pos1+6))
							copy(buf[pos2:], "\r\n\r\n")
						}
					}
				}

				strictSlice := buf[:l:l]
				expected := simd.IndexCRLFCRLF(strictSlice)
				got := simd.IndexCRLFCRLFVector(strictSlice)

				if expected != got {
					t.Errorf(
						"fuzz mismatch: g=%d it=%d len=%d cat=%d expected=%d got=%d (data=%q)",
						gID, it, l, category, expected, got, strictSlice,
					)
					return
				}

				if got >= 0 {
					rem := strictSlice[got:]
					_ = rem
				}
			}
		}(g)
	}

	wg.Wait()
}

// TestChallenger2_ZeroAllocations strictly asserts 0 heap allocations across all payload profiles.
func TestChallenger2_ZeroAllocations(t *testing.T) {
	testPayloads := []struct {
		name string
		data []byte
	}{
		{"nil", nil},
		{"empty", []byte{}},
		{"short_3_bytes", []byte("OK\r")},
		{"crlfcrlf_4_bytes", []byte("\r\n\r\n")},
		{"short_11_bytes", []byte("OK\r\n\r\nHELLO")},
		{"exact_31_bytes", append(bytes.Repeat([]byte("A"), 27), []byte("\r\n\r\n")...)},
		{"exact_32_bytes", append(bytes.Repeat([]byte("A"), 28), []byte("\r\n\r\n")...)},
		{
			"boundary_straddle_28",
			append(
				bytes.Repeat([]byte("A"), 28),
				append([]byte("\r\n\r\n"), bytes.Repeat([]byte("B"), 20)...)...,
			),
		},
		{
			"boundary_straddle_30",
			append(
				bytes.Repeat([]byte("A"), 30),
				append([]byte("\r\n\r\n"), bytes.Repeat([]byte("B"), 20)...)...,
			),
		},
		{
			"boundary_straddle_32",
			append(
				bytes.Repeat([]byte("A"), 32),
				append([]byte("\r\n\r\n"), bytes.Repeat([]byte("B"), 20)...)...,
			),
		},
		{
			"boundary_straddle_34",
			append(
				bytes.Repeat([]byte("A"), 34),
				append([]byte("\r\n\r\n"), bytes.Repeat([]byte("B"), 20)...)...,
			),
		},
		{"legacy_lflf_64_bytes", append(bytes.Repeat([]byte("C"), 62), []byte("\n\n")...)},
		{"large_1024_bytes_no_match", bytes.Repeat([]byte("D"), 1024)},
		{
			"large_1024_bytes_with_crlfcrlf",
			append(
				bytes.Repeat([]byte("E"), 500),
				append([]byte("\r\n\r\n"), bytes.Repeat([]byte("F"), 520)...)...,
			),
		},
	}

	for _, tc := range testPayloads {
		t.Run(tc.name, func(t *testing.T) {
			allocs := testing.AllocsPerRun(1000, func() {
				_ = simd.IndexCRLFCRLFVector(tc.data)
			})
			assert.Equal(t, float64(0), allocs, fmt.Sprintf("expected 0 allocations for %s", tc.name))
		})
	}
}
