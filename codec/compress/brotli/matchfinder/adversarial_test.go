// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package matchfinder

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"math"
	"testing"
)

// verifyMatchesCumulative verifies that matches correctly reconstruct data when streaming across chunks.
func verifyMatchesCumulative(t *testing.T, cumulative []byte, chunkStart int, chunk []byte, matches []Match, name string) {
	t.Helper()
	if len(chunk) == 0 {
		return
	}
	reconstructed := make([]byte, chunkStart, len(cumulative))
	copy(reconstructed, cumulative[:chunkStart])
	pos := chunkStart
	for idx, m := range matches {
		if m.Unmatched > 0 {
			if pos+m.Unmatched > len(cumulative) {
				t.Fatalf("%s: match %d unmatched %d exceeds cumulative len %d at pos %d", name, idx, m.Unmatched, len(cumulative), pos)
			}
			reconstructed = append(reconstructed, cumulative[pos:pos+m.Unmatched]...)
			pos += m.Unmatched
		}
		if m.Length > 0 {
			if m.Distance <= 0 {
				t.Fatalf("%s: match %d invalid distance %d", name, idx, m.Distance)
			}
			start := len(reconstructed) - m.Distance
			if start < 0 {
				t.Fatalf("%s: match %d distance %d exceeds reconstructed len %d", name, idx, m.Distance, len(reconstructed))
			}
			for i := 0; i < m.Length; i++ {
				reconstructed = append(reconstructed, reconstructed[start+i])
			}
			pos += m.Length
		}
	}
	if pos != len(cumulative) {
		t.Fatalf("%s: total matched bytes %d != cumulative len %d", name, pos, len(cumulative))
	}
	if !bytes.Equal(cumulative, reconstructed) {
		t.Fatalf("%s: cumulative reconstructed data mismatch", name)
	}
}

// TestAdversarialIdenticalByteRuns tests all matchfinders with large runs of identical bytes.
func TestAdversarialIdenticalByteRuns(t *testing.T) {
	finders := map[string]func() MatchFinder{
		"M0":         func() MatchFinder { return &M0{} },
		"M0_Lazy":    func() MatchFinder { return &M0{Lazy: true} },
		"M4":         func() MatchFinder { return &M4{} },
		"M4_Chained": func() MatchFinder { return &M4{ChainLength: 16, DistanceBitCost: 32} },
		"ZFast":      func() MatchFinder { return &ZFast{} },
		"ZDFast":     func() MatchFinder { return &ZDFast{} },
		"ZM":         func() MatchFinder { return &ZM{} },
		"Trio":       func() MatchFinder { return &Trio{} },
		"Bargain1":   func() MatchFinder { return &Bargain1{} },
		"Bargain2":   func() MatchFinder { return &Bargain2{} },
		"Bargain3":   func() MatchFinder { return &Bargain3{} },
		"Pathfinder": func() MatchFinder { return &Pathfinder{} },
	}

	byteRuns := []struct {
		name string
		val  byte
		size int
	}{
		{"Zeros_16KB", 0x00, 16 * 1024},
		{"Zeros_64KB", 0x00, 64 * 1024},
		{"Ones_64KB", 0xFF, 64 * 1024},
		{"LetterA_64KB", 'A', 64 * 1024},
		{"Zeros_128KB", 0x00, 128 * 1024},
		{"LetterB_256KB", 'B', 256 * 1024},
	}

	for fname, newFinder := range finders {
		t.Run(fname, func(t *testing.T) {
			for _, tc := range byteRuns {
				if (fname == "M0" || fname == "M0_Lazy") && tc.size > 65536 {
					continue // M0 max block size is 65536 by design
				}
				t.Run(tc.name, func(t *testing.T) {
					data := bytes.Repeat([]byte{tc.val}, tc.size)
					mf := newFinder()
					mf.Reset()
					var matches []Match
					matches = mf.FindMatches(matches, data)
					verifyMatches(t, data, matches, fname+"_"+tc.name)
				})
			}
		})
	}
}

// TestAdversarialWorstCaseHashing tests pathological repeating and collision patterns.
func TestAdversarialWorstCaseHashing(t *testing.T) {
	finders := map[string]func() MatchFinder{
		"M0":         func() MatchFinder { return &M0{} },
		"M4":         func() MatchFinder { return &M4{} },
		"ZFast":      func() MatchFinder { return &ZFast{} },
		"ZDFast":     func() MatchFinder { return &ZDFast{} },
		"ZM":         func() MatchFinder { return &ZM{} },
		"Trio":       func() MatchFinder { return &Trio{} },
		"Bargain1":   func() MatchFinder { return &Bargain1{} },
		"Pathfinder": func() MatchFinder { return &Pathfinder{} },
	}

	patterns := []struct {
		name string
		gen  func() []byte
	}{
		{"2BytePeriod", func() []byte {
			return bytes.Repeat([]byte{0xAA, 0x55}, 16384)
		}},
		{"3BytePeriod", func() []byte {
			return bytes.Repeat([]byte{0x12, 0x34, 0x56}, 10000)
		}},
		{"4BytePeriod", func() []byte {
			return bytes.Repeat([]byte{0xDE, 0xAD, 0xBE, 0xEF}, 8192)
		}},
		{"FibonacciHashCollision", func() []byte {
			// Multiples of Fibonacci hash golden ratio constant
			buf := make([]byte, 32768)
			for i := 0; i < len(buf)/4; i++ {
				v := uint32(i) * 2654435761
				buf[i*4] = byte(v)
				buf[i*4+1] = byte(v >> 8)
				buf[i*4+2] = byte(v >> 16)
				buf[i*4+3] = byte(v >> 24)
			}
			return buf
		}},
	}

	for fname, newFinder := range finders {
		t.Run(fname, func(t *testing.T) {
			for _, pat := range patterns {
				t.Run(pat.name, func(t *testing.T) {
					data := pat.gen()
					mf := newFinder()
					mf.Reset()
					var matches []Match
					matches = mf.FindMatches(matches, data)
					verifyMatches(t, data, matches, fname+"_"+pat.name)
				})
			}
		})
	}
}

// TestAdversarialSlidingWindowAndHistoryTrim tests multi-chunk streams exceeding MaxDistance.
func TestAdversarialSlidingWindowAndHistoryTrim(t *testing.T) {
	finders := map[string]func() MatchFinder{
		"M4":         func() MatchFinder { return &M4{MaxDistance: 16384} },
		"ZFast":      func() MatchFinder { return &ZFast{MaxDistance: 16384} },
		"ZDFast":     func() MatchFinder { return &ZDFast{MaxDistance: 16384} },
		"ZM":         func() MatchFinder { return &ZM{MaxDistance: 16384} },
		"Trio":       func() MatchFinder { return &Trio{MaxDistance: 16384} },
		"Bargain1":   func() MatchFinder { return &Bargain1{MaxDistance: 16384} },
		"Bargain2":   func() MatchFinder { return &Bargain2{MaxDistance: 16384} },
		"Bargain3":   func() MatchFinder { return &Bargain3{MaxDistance: 16384} },
		"Pathfinder": func() MatchFinder { return &Pathfinder{MaxDistance: 16384} },
	}

	chunkSizes := []int{16, 128, 1024, 8192, 16384, 32768, 65536}

	for fname, newFinder := range finders {
		t.Run(fname, func(t *testing.T) {
			mf := newFinder()
			mf.Reset()

			var cumulative []byte
			for _, sz := range chunkSizes {
				chunk := make([]byte, sz)
				for i := range chunk {
					chunk[i] = byte((i * 31) ^ (i % 7))
				}
				chunkStart := len(cumulative)
				cumulative = append(cumulative, chunk...)

				var matches []Match
				matches = mf.FindMatches(matches, chunk)
				verifyMatchesCumulative(t, cumulative, chunkStart, chunk, matches, fname)
			}
		})
	}
}

// TestAdversarialRapidHistoryResets tests rapid alternating resets and varied block feeding.
func TestAdversarialRapidHistoryResets(t *testing.T) {
	finders := map[string]func() MatchFinder{
		"M0":         func() MatchFinder { return &M0{} },
		"M4":         func() MatchFinder { return &M4{} },
		"ZFast":      func() MatchFinder { return &ZFast{} },
		"ZDFast":     func() MatchFinder { return &ZDFast{} },
		"ZM":         func() MatchFinder { return &ZM{} },
		"Trio":       func() MatchFinder { return &Trio{} },
		"Bargain1":   func() MatchFinder { return &Bargain1{} },
		"Pathfinder": func() MatchFinder { return &Pathfinder{} },
	}

	for fname, newFinder := range finders {
		t.Run(fname, func(t *testing.T) {
			mf := newFinder()
			for iter := 0; iter < 50; iter++ {
				mf.Reset()
				sz := 10 + (iter * 17) % 5000
				data := make([]byte, sz)
				for i := range data {
					data[i] = byte(i + iter)
				}
				var matches []Match
				matches = mf.FindMatches(matches, data)
				verifyMatches(t, data, matches, fname)
			}
		})
	}
}

// TestAdversarialCryptoRandomData tests random incompressible noise through all matchfinders.
func TestAdversarialCryptoRandomData(t *testing.T) {
	finders := map[string]func() MatchFinder{
		"M0":         func() MatchFinder { return &M0{} },
		"M4":         func() MatchFinder { return &M4{} },
		"ZFast":      func() MatchFinder { return &ZFast{} },
		"ZDFast":     func() MatchFinder { return &ZDFast{} },
		"ZM":         func() MatchFinder { return &ZM{} },
		"Trio":       func() MatchFinder { return &Trio{} },
		"Bargain1":   func() MatchFinder { return &Bargain1{} },
		"Pathfinder": func() MatchFinder { return &Pathfinder{} },
	}

	data := make([]byte, 32768)
	_, err := rand.Read(data)
	if err != nil {
		t.Fatal(err)
	}
	for fname, newFinder := range finders {
		t.Run(fname, func(t *testing.T) {
			mf := newFinder()
			mf.Reset()
			var matches []Match
			matches = mf.FindMatches(matches, data)
			verifyMatches(t, data, matches, fname+"_Random")
		})
	}
}

// TestAdversarialM4SliceOutOfBoundsPanic demonstrates the panic when pre-populated dst has Distance > i+1.
func TestAdversarialM4SliceOutOfBoundsPanic(t *testing.T) {
	m4 := &M4{}
	dst := []Match{{Unmatched: 0, Length: 4, Distance: 50}}
	src := []byte("hello world this is a test string for m4 panic reproducer")
	matches := m4.FindMatches(dst, src)
	if len(matches) < 1 {
		t.Fatalf("expected at least original match")
	}
	verifyMatches(t, src, matches[1:], "M4_PrePopulated")
}

// TestAdversarialM4PrePopulatedLiteralZeroDistance demonstrates invalid 0-distance match emission in M4.
func TestAdversarialM4PrePopulatedLiteralZeroDistance(t *testing.T) {
	m4 := &M4{}
	dst := []Match{{Unmatched: 5, Length: 0, Distance: 0}}
	src := bytes.Repeat([]byte("test pattern with repeated tokens for m4"), 10)
	matches := m4.FindMatches(dst, src)
	for i, m := range matches[1:] {
		if m.Length > 0 && m.Distance <= 0 {
			t.Fatalf("M4 emitted invalid match at idx %d with distance %d (length %d)", i, m.Distance, m.Length)
		}
	}
	verifyMatches(t, src, matches[1:], "M4_LiteralPrePopulated")
}

// TestAdversarialZFastZeroBytesDistance demonstrates ZFast emitting invalid match with Distance 0 on zero bytes.
func TestAdversarialZFastZeroBytesDistance(t *testing.T) {
	zf := &ZFast{}
	zf.Reset()
	data := make([]byte, 1024)
	matches := zf.FindMatches(nil, data)
	for i, m := range matches {
		if m.Length > 0 && m.Distance <= 0 {
			t.Fatalf("ZFast emitted invalid match at idx %d with distance %d (length %d)", i, m.Distance, m.Length)
		}
	}
	verifyMatches(t, data, matches, "ZFast_ZeroBytes")
}

// TestAdversarialZDFastZeroBytesDistance demonstrates ZDFast emitting invalid match with Distance 0 on zero bytes.
func TestAdversarialZDFastZeroBytesDistance(t *testing.T) {
	zdf := &ZDFast{}
	zdf.Reset()
	data := make([]byte, 1024)
	matches := zdf.FindMatches(nil, data)
	for i, m := range matches {
		if m.Length > 0 && m.Distance <= 0 {
			t.Fatalf("ZDFast emitted invalid match at idx %d with distance %d (length %d)", i, m.Distance, m.Length)
		}
	}
	verifyMatches(t, data, matches, "ZDFast_ZeroBytes")
}

// TestAdversarialM4ExhaustivePrePopulatedDst tests M4 against a combinatorial space of pre-populated dst slices.
func TestAdversarialM4ExhaustivePrePopulatedDst(t *testing.T) {
	distances := []int{
		math.MinInt32, -1000000, -65536, -100, -1,
		0,
		1, 2, 3, 4, 5, 6, 7, 8, 15, 16, 50, 100, 1000, 65535, 65536, 100000, math.MaxInt32,
	}
	lengths := []int{0, 1, 4, 10, 100}
	unmatched := []int{0, 1, 10}

	testInputs := [][]byte{
		{},
		{0x42},
		{0x00, 0x00, 0x00},
		{0x01, 0x02, 0x03, 0x04},
		{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11}, // 7 bytes (< 8)
		bytes.Repeat([]byte("abcdefgh"), 16),
		bytes.Repeat([]byte{0x00}, 512),
		bytes.Repeat([]byte("test repetition in brotli matchfinder M4 dst stress "), 20),
	}

	for _, d := range distances {
		for _, l := range lengths {
			for _, u := range unmatched {
				preDst := []Match{{Unmatched: u, Length: l, Distance: d}}
				for inIdx, src := range testInputs {
					m4 := &M4{}
					var matches []Match
					func() {
						defer func() {
							if r := recover(); r != nil {
								t.Fatalf("M4 panicked with pre-populated Match{Unmatched:%d, Length:%d, Distance:%d} on input %d: %v",
									u, l, d, inIdx, r)
							}
						}()
						matches = m4.FindMatches(append([]Match(nil), preDst...), src)
					}()

					newMatches := matches[len(preDst):]
					for mIdx, m := range newMatches {
						if m.Length > 0 && m.Distance <= 0 {
							t.Fatalf("M4 emitted non-positive distance %d (length %d) at idx %d (pre-populated d=%d)",
								m.Distance, m.Length, mIdx, d)
						}
					}
					if len(src) >= 8 {
						verifyMatches(t, src, newMatches, "M4_ExhaustivePreDst")
					}
				}
			}
		}
	}
}

// TestAdversarialM4ChainedDstStress tests passing the output of FindMatches back into FindMatches across multiple blocks.
func TestAdversarialM4ChainedDstStress(t *testing.T) {
	m4 := &M4{}
	var dst []Match
	// Start with arbitrary dummy matches
	dst = append(dst, Match{Unmatched: 5, Length: 0, Distance: 0})
	dst = append(dst, Match{Unmatched: 0, Length: 8, Distance: 100})
	dst = append(dst, Match{Unmatched: 2, Length: 4, Distance: -50})

	blocks := [][]byte{
		[]byte("block number one with some repeats block number one"),
		[]byte("second block continues the stream with repeats second block"),
		make([]byte, 256),
		bytes.Repeat([]byte{0x55}, 128),
		[]byte("short"),
		[]byte("another slightly longer block with repetitive patterns repetitive patterns"),
	}

	var cumulative []byte
	for bIdx, block := range blocks {
		chunkStart := len(cumulative)
		cumulative = append(cumulative, block...)

		prevLen := len(dst)
		dst = m4.FindMatches(dst, block)
		newMatches := dst[prevLen:]
		for i, m := range newMatches {
			if m.Length > 0 && m.Distance <= 0 {
				t.Fatalf("block %d: M4 emitted invalid distance %d at index %d", bIdx, m.Distance, i)
			}
		}
		verifyMatchesCumulative(t, cumulative, chunkStart, block, newMatches, fmt.Sprintf("M4_Chained_Block_%d", bIdx))
	}
}

// TestAdversarialZFastZDFastZeroBytesStress tests ZFast and ZDFast on extensive zero-byte lengths.
func TestAdversarialZFastZDFastZeroBytesStress(t *testing.T) {
	sizes := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 15, 16, 31, 32, 63, 64, 127, 128, 255, 256, 512, 1024, 4096, 65536}

	finders := map[string]func() MatchFinder{
		"ZFast":  func() MatchFinder { return &ZFast{} },
		"ZDFast": func() MatchFinder { return &ZDFast{} },
	}

	for name, newFinder := range finders {
		t.Run(name, func(t *testing.T) {
			for _, sz := range sizes {
				data := make([]byte, sz)
				mf := newFinder()
				mf.Reset()
				matches := mf.FindMatches(nil, data)
				for i, m := range matches {
					if m.Length > 0 && m.Distance <= 0 {
						t.Fatalf("%s on %d zero bytes emitted match %d with invalid distance %d (len %d)",
							name, sz, i, m.Distance, m.Length)
					}
				}
				verifyMatches(t, data, matches, fmt.Sprintf("%s_Zeros_%d", name, sz))
			}
		})
	}
}

// TestAdversarialZFastZDFastFibonacciCollisions tests ZFast and ZDFast on various Fibonacci collision block sizes.
func TestAdversarialZFastZDFastFibonacciCollisions(t *testing.T) {
	sizes := []int{16, 32, 64, 128, 256, 512, 1024, 4096, 16384, 32768, 65536}

	finders := map[string]func() MatchFinder{
		"ZFast":  func() MatchFinder { return &ZFast{} },
		"ZDFast": func() MatchFinder { return &ZDFast{} },
	}

	for name, newFinder := range finders {
		t.Run(name, func(t *testing.T) {
			for _, sz := range sizes {
				buf := make([]byte, sz)
				for i := 0; i < len(buf)/4; i++ {
					v := uint32(i) * 2654435761
					buf[i*4] = byte(v)
					buf[i*4+1] = byte(v >> 8)
					buf[i*4+2] = byte(v >> 16)
					buf[i*4+3] = byte(v >> 24)
				}
				mf := newFinder()
				mf.Reset()
				matches := mf.FindMatches(nil, buf)
				for i, m := range matches {
					if m.Length > 0 && m.Distance <= 0 {
						t.Fatalf("%s on %d fibonacci bytes emitted match %d with invalid distance %d (len %d)",
							name, sz, i, m.Distance, m.Length)
					}
				}
				verifyMatches(t, buf, matches, fmt.Sprintf("%s_Fib_%d", name, sz))
			}
		})
	}
}

// TestAdversarialZFastZDFastLookaheadCollisions tests lookahead collisions where candidate values match.
func TestAdversarialZFastZDFastLookaheadCollisions(t *testing.T) {
	finders := map[string]func() MatchFinder{
		"ZFast":  func() MatchFinder { return &ZFast{} },
		"ZDFast": func() MatchFinder { return &ZDFast{} },
	}

	for name, newFinder := range finders {
		t.Run(name, func(t *testing.T) {
			for prefixZeros := 0; prefixZeros <= 8; prefixZeros++ {
				data := make([]byte, prefixZeros)
				token := []byte("LOOKAHEAD_COLLISION_PATTERN_12345678")
				for k := 0; k < 20; k++ {
					data = append(data, token...)
				}

				mf := newFinder()
				mf.Reset()
				matches := mf.FindMatches(nil, data)
				for i, m := range matches {
					if m.Length > 0 && m.Distance <= 0 {
						t.Fatalf("%s with prefix zeros %d emitted match %d with invalid distance %d",
							name, prefixZeros, i, m.Distance)
					}
				}
				verifyMatches(t, data, matches, fmt.Sprintf("%s_PrefixZeros_%d", name, prefixZeros))
			}
		})
	}
}

// TestAdversarialAllFindersPrePopulatedDst tests all matchfinders with pre-populated dst containing valid and literal entries.
func TestAdversarialAllFindersPrePopulatedDst(t *testing.T) {
	finders := map[string]func() MatchFinder{
		"M0":         func() MatchFinder { return &M0{} },
		"M0_Lazy":    func() MatchFinder { return &M0{Lazy: true} },
		"M4":         func() MatchFinder { return &M4{} },
		"M4_Chained": func() MatchFinder { return &M4{ChainLength: 16, DistanceBitCost: 32} },
		"ZFast":      func() MatchFinder { return &ZFast{} },
		"ZDFast":     func() MatchFinder { return &ZDFast{} },
		"ZM":         func() MatchFinder { return &ZM{} },
		"Trio":       func() MatchFinder { return &Trio{} },
		"Bargain1":   func() MatchFinder { return &Bargain1{} },
		"Bargain2":   func() MatchFinder { return &Bargain2{} },
		"Bargain3":   func() MatchFinder { return &Bargain3{} },
		"Pathfinder": func() MatchFinder { return &Pathfinder{} },
	}

	prePopulations := [][]Match{
		{{Unmatched: 0, Length: 4, Distance: 50}},
		{{Unmatched: 10, Length: 0, Distance: 0}},
		{{Unmatched: 0, Length: 8, Distance: 1000}, {Unmatched: 5, Length: 0, Distance: 0}},
	}

	src := bytes.Repeat([]byte("stress testing all matchfinders with pre-populated dst entries "), 10)

	for fname, newFinder := range finders {
		t.Run(fname, func(t *testing.T) {
			for pIdx, preDst := range prePopulations {
				mf := newFinder()
				mf.Reset()
				var matches []Match
				func() {
					defer func() {
						if r := recover(); r != nil {
							t.Fatalf("%s panicked with pre-populated dst[%d]: %v", fname, pIdx, r)
						}
					}()
					matches = mf.FindMatches(append([]Match(nil), preDst...), src)
				}()

				newMatches := matches[len(preDst):]
				for i, m := range newMatches {
					if m.Length > 0 && m.Distance <= 0 {
						t.Fatalf("%s emitted invalid distance %d at index %d", fname, m.Distance, i)
					}
				}
				verifyMatches(t, src, newMatches, fmt.Sprintf("%s_PreDst_%d", fname, pIdx))
			}
		})
	}
}



