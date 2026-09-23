// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package matchfinder

import (
	"bytes"
	"errors"
	"math"
	"testing"
)

// verifyMatches checks that a slice of Match accurately reconstructs the input.
func verifyMatches(t *testing.T, src []byte, matches []Match, name string) {
	t.Helper()
	if len(src) == 0 {
		return
	}
	pos := 0
	reconstructed := make([]byte, 0, len(src))
	for idx, m := range matches {
		if m.Unmatched > 0 {
			if pos+m.Unmatched > len(src) {
				t.Fatalf("%s: match %d unmatched %d exceeds src len %d at pos %d", name, idx, m.Unmatched, len(src), pos)
			}
			reconstructed = append(reconstructed, src[pos:pos+m.Unmatched]...)
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
	if pos != len(src) {
		t.Fatalf("%s: total matched bytes %d != len(src) %d", name, pos, len(src))
	}
	if !bytes.Equal(src, reconstructed) {
		t.Fatalf("%s: reconstructed data mismatch", name)
	}
}

// TestAllMatchFinders verifies all 10 MatchFinders against varied patterns.
func TestAllMatchFinders(t *testing.T) {
	finders := map[string]func() MatchFinder{
		"M0":         func() MatchFinder { return &M0{} },
		"M0_Lazy":    func() MatchFinder { return &M0{Lazy: true} },
		"M4":         func() MatchFinder { return &M4{} },
		"M4_Chained": func() MatchFinder { return &M4{ChainLength: 8, DistanceBitCost: 32} },
		"ZFast":      func() MatchFinder { return &ZFast{} },
		"ZDFast":     func() MatchFinder { return &ZDFast{} },
		"ZM":         func() MatchFinder { return &ZM{} },
		"Trio":       func() MatchFinder { return &Trio{} },
		"Bargain1":   func() MatchFinder { return &Bargain1{} },
		"Bargain1_S": func() MatchFinder { return &Bargain1{Skip: true} },
		"Bargain2":   func() MatchFinder { return &Bargain2{} },
		"Bargain2_S": func() MatchFinder { return &Bargain2{Skip: true} },
		"Bargain3":   func() MatchFinder { return &Bargain3{} },
		"Bargain3_S": func() MatchFinder { return &Bargain3{Skip: true} },
		"Pathfinder": func() MatchFinder { return &Pathfinder{} },
		"Path_Chain": func() MatchFinder { return &Pathfinder{ChainLength: 4} },
	}

	testCases := []struct {
		name string
		data []byte
	}{
		{"Empty", []byte{}},
		{"SingleByte", []byte("X")},
		{"ShortNonMatch", []byte("abcdefghijkl")},
		{"RepeatedSmall", bytes.Repeat([]byte("hello world! "), 50)},
		{"RepeatedStructured", bytes.Repeat([]byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"), 100)},
		{"AllSameByte", bytes.Repeat([]byte("A"), 1024)},
	}

	for fname, newFinder := range finders {
		t.Run(fname, func(t *testing.T) {
			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					mf := newFinder()
					mf.Reset()
					var matches []Match
					matches = mf.FindMatches(matches, tc.data)
					verifyMatches(t, tc.data, matches, fname+"_"+tc.name)
				})
			}
		})
	}
}

// TestSlidingWindowHistory verifies sliding window and history trimming across blocks.
func TestSlidingWindowHistory(t *testing.T) {
	finders := map[string]func() MatchFinder{
		"M4":         func() MatchFinder { return &M4{MaxDistance: 512} },
		"ZFast":      func() MatchFinder { return &ZFast{MaxDistance: 512} },
		"ZDFast":     func() MatchFinder { return &ZDFast{MaxDistance: 512} },
		"ZM":         func() MatchFinder { return &ZM{MaxDistance: 512} },
		"Trio":       func() MatchFinder { return &Trio{MaxDistance: 512} },
		"Bargain1":   func() MatchFinder { return &Bargain1{MaxDistance: 512} },
		"Bargain2":   func() MatchFinder { return &Bargain2{MaxDistance: 512} },
		"Bargain3":   func() MatchFinder { return &Bargain3{MaxDistance: 512} },
		"Pathfinder": func() MatchFinder { return &Pathfinder{MaxDistance: 512} },
	}

	block := bytes.Repeat([]byte("0123456789abcdefghijklmnopqrstuvwxyz"), 16) // 576 bytes
	for fname, newFinder := range finders {
		t.Run(fname, func(t *testing.T) {
			mf := newFinder()
			mf.Reset()
			for i := 0; i < 5; i++ {
				var matches []Match
				matches = mf.FindMatches(matches, block)
				if len(matches) == 0 {
					t.Fatalf("expected matches on iteration %d", i)
				}
			}
		})
	}
}

// TestM0EdgeCases covers M0 specific boundaries (panic on >65536, max distance, max length).
func TestM0EdgeCases(t *testing.T) {
	m := &M0{MaxDistance: 16, MaxLength: 8}
	m.Reset()
	(M0{}).Reset()

	data := bytes.Repeat([]byte("abcdef123456"), 10)
	matches := m.FindMatches(nil, data)
	for _, match := range matches {
		if match.Distance > 16 {
			t.Fatalf("match distance %d > 16", match.Distance)
		}
		if match.Length > 8 {
			t.Fatalf("match length %d > 8", match.Length)
		}
	}

	// Test panic on input > 65536
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic on block > 65536")
		}
	}()
	m.FindMatches(nil, make([]byte, 65537))
}

// TestNoMatchFinder verifies NoMatchFinder.
func TestNoMatchFinder(t *testing.T) {
	nm := NoMatchFinder{}
	nm.Reset()
	src := []byte("hello world")
	matches := nm.FindMatches(nil, src)
	if len(matches) != 1 || matches[0].Unmatched != len(src) || matches[0].Length != 0 {
		t.Fatalf("unexpected matches: %+v", matches)
	}
}

// TestAutoReset verifies that AutoReset calls Reset before each FindMatches call.
func TestAutoReset(t *testing.T) {
	base := &M4{MaxDistance: 1024}
	ar := AutoReset{MatchFinder: base}
	src := bytes.Repeat([]byte("test pattern data "), 10)
	m1 := ar.FindMatches(nil, src)
	m2 := ar.FindMatches(nil, src)
	if len(m1) != len(m2) {
		t.Fatalf("AutoReset mismatch between calls: %d vs %d", len(m1), len(m2))
	}
}

// TestTextEncoder verifies human-readable encoding of matches.
func TestTextEncoder(t *testing.T) {
	te := TextEncoder{}
	te.Reset()
	src := []byte("abcdeabcde")
	matches := []Match{
		{Unmatched: 5, Length: 5, Distance: 5},
	}
	out := te.Encode(nil, src, matches, true)
	expected := "abcde<5,5>"
	if string(out) != expected {
		t.Fatalf("got %q, want %q", string(out), expected)
	}
}

// failWriter is an io.Writer that always returns an error.
type failWriter struct{}

func (failWriter) Write(p []byte) (int, error) {
	return 0, errors.New("write error")
}

// TestWriter verifies Writer buffered and unbuffered behavior.
func TestWriter(t *testing.T) {
	// Unbuffered (BlockSize = 0)
	var buf bytes.Buffer
	w := &Writer{
		Dest:        &buf,
		MatchFinder: &M0{},
		Encoder:     TextEncoder{},
		BlockSize:   0,
	}
	n, err := w.Write([]byte("abcdefabcdef"))
	if err != nil || n != 12 {
		t.Fatalf("Write failed: n=%d, err=%v", n, err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Buffered (BlockSize = 32)
	buf.Reset()
	w.Reset(&buf)
	w.BlockSize = 32
	input := bytes.Repeat([]byte("0123456789abcdef"), 4) // 64 bytes
	_, err = w.Write(input[:40])
	if err != nil {
		t.Fatalf("Write 1 failed: %v", err)
	}
	_, err = w.Write(input[40:])
	if err != nil {
		t.Fatalf("Write 2 failed: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Fail writer path
	failW := &Writer{
		Dest:        failWriter{},
		MatchFinder: &M0{},
		Encoder:     TextEncoder{},
	}
	_, err = failW.Write([]byte("fail test"))
	if err == nil {
		t.Fatalf("expected write error")
	}
	// Calling Write after error should return existing error
	_, err2 := failW.Write([]byte("more"))
	if err2 == nil {
		t.Fatalf("expected write error on subsequent write")
	}
}

// TestOverflowAndEdgeBranches exercises internal current overflow and small input branches.
func TestOverflowAndEdgeBranches(t *testing.T) {
	// Small inputs < 16 bytes
	for _, sz := range []int{0, 1, 4, 10, 15} {
		sm := make([]byte, sz)
		_ = (&ZFast{}).FindMatches(nil, sm)
		_ = (&ZDFast{}).FindMatches(nil, sm)
		_ = (&ZM{}).FindMatches(nil, sm)
		_ = (&Trio{}).FindMatches(nil, sm)
	}

	// Overflow protection branches for ZFast, ZDFast, ZM
	data := bytes.Repeat([]byte("0123456789abcdefghijklmnopqrstuvwxyz"), 10)
	zf := &ZFast{
		MaxDistance: 512,
		current:     int32(math.MaxInt32) - 500,
	}
	zf.table[0] = tableEntry{offset: int32(math.MaxInt32) - 100}
	zf.table[1] = tableEntry{offset: 10}
	_ = zf.FindMatches(nil, data)

	zdf := &ZDFast{
		MaxDistance: 512,
		current:     int32(math.MaxInt32) - 500,
	}
	zdf.table[0] = tableEntry{offset: int32(math.MaxInt32) - 100}
	zdf.table[1] = tableEntry{offset: 10}
	zdf.longTable[0] = tableEntry{offset: int32(math.MaxInt32) - 100}
	zdf.longTable[1] = tableEntry{offset: 10}
	_ = zdf.FindMatches(nil, data)

	zm := &ZM{
		MaxDistance: 512,
	}
	zm.table[0] = tableEntry{offset: 100}
	zm.table[1] = tableEntry{offset: 10}
	zm.longTable[0] = tableEntry{offset: 100}
	zm.longTable[1] = tableEntry{offset: 10}
	_ = zm.FindMatches(nil, data)

	// Slide down history branches (cap > 0 and len + len > cap)
	for _, fn := range []func(src []byte){
		func(src []byte) {
			f := &ZFast{MaxDistance: 64, history: make([]byte, 100, 100)}
			_ = f.FindMatches(nil, src)
		},
		func(src []byte) {
			f := &ZDFast{MaxDistance: 64, history: make([]byte, 100, 100)}
			_ = f.FindMatches(nil, src)
		},
		func(src []byte) {
			f := &ZM{MaxDistance: 64, history: make([]byte, 100, 100)}
			_ = f.FindMatches(nil, src)
		},
		func(src []byte) {
			f := &Trio{MaxDistance: 64, history: make([]byte, 100, 100)}
			_ = f.FindMatches(nil, src)
		},
	} {
		fn(make([]byte, 20))
	}

	// Repeated distance matches (offset1, offset2) across all fast matchers
	repPattern := []byte("0123456789abcdef" + "0123456789abcdef" + "0123456789abcdef" +
		"ABCDEFGHIJKLMNOP" + "ABCDEFGHIJKLMNOP" + "0123456789abcdef" +
		"0123456789abcdef" + "ABCDEFGHIJKLMNOP" + "0123456789abcdef" +
		"abcdef0123456789abcdef0123456789abcdef0123456789")
	for _, mf := range []MatchFinder{
		&ZFast{MaxDistance: 1024},
		&ZDFast{MaxDistance: 1024},
		&ZM{MaxDistance: 1024},
		&Trio{MaxDistance: 1024},
	} {
		var m []Match
		m = mf.FindMatches(m, repPattern)
		verifyMatches(t, repPattern, m, "repPattern")
	}

	// Overlapping and chained matches for M4, ZM, Trio, and Pathfinder
	overlapData := []byte("abcdef123456abcdef123456bcdef1234567cdef12345678abcdef123456" +
		"1234567890abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz" +
		"bcdefghijklmnopqrstuvwxyzcdefghijklmnopqrstuvwxyza")
	for _, mf := range []MatchFinder{
		&M4{ChainLength: 16, MaxDistance: 1024},
		&ZM{MaxDistance: 1024},
		&Trio{MaxDistance: 1024},
		&Pathfinder{ChainLength: 16, MaxDistance: 1024},
	} {
		var m []Match
		m = mf.FindMatches(m, overlapData)
		verifyMatches(t, overlapData, m, "overlapData")
	}

	// History trimming with active chain
	m4Trim := &M4{MaxDistance: 64, ChainLength: 4}
	for i := 0; i < 5; i++ {
		_ = m4Trim.FindMatches(nil, bytes.Repeat([]byte("0123456789"), 15))
	}
	pfTrim := &Pathfinder{MaxDistance: 64, ChainLength: 4}
	for i := 0; i < 5; i++ {
		_ = pfTrim.FindMatches(nil, bytes.Repeat([]byte("0123456789"), 15))
	}

	// Reset calls on value receivers
	var m0 M0
	m0.Reset()
	var te TextEncoder
	te.Reset()
	var nm NoMatchFinder
	nm.Reset()
}

// TestMatchFinderAdvancedPatterns tests alternating block patterns and dense overlaps.
func TestMatchFinderAdvancedPatterns(t *testing.T) {
	// Alternating patterns to trigger offset2 reuse
	patternAB := bytes.Repeat([]byte("0123456789abcdef"+"ABCDEFGHIJKLMNO_"), 40)
	for _, mf := range []MatchFinder{
		&ZFast{MaxDistance: 2048},
		&ZDFast{MaxDistance: 2048},
		&ZM{MaxDistance: 2048},
		&Trio{MaxDistance: 2048},
	} {
		var m []Match
		m = mf.FindMatches(m, patternAB)
		verifyMatches(t, patternAB, m, "patternAB")
	}

	// Dense overlapping and chained patterns for M4 overlap cases
	denseOverlap := bytes.Repeat([]byte("12345678123456782345678934567890abcdefgh45678901"), 30)
	for _, mf := range []MatchFinder{
		&M4{ChainLength: 16, MaxDistance: 2048},
		&M4{ChainLength: 8, MinLength: 4, MaxDistance: 512},
		&Pathfinder{ChainLength: 8, MaxDistance: 512},
	} {
		var m []Match
		m = mf.FindMatches(m, denseOverlap)
		verifyMatches(t, denseOverlap, m, "denseOverlap")
	}

	// Short match at s, long match at s+1
	shortThenLong := []byte("abcd1234567890abcdefXXXXXX" + "abcdZZZZ" + "1234567890abcdef")
	for _, mf := range []MatchFinder{
		&ZFast{MaxDistance: 1024},
		&ZDFast{MaxDistance: 1024},
	} {
		var m []Match
		m = mf.FindMatches(m, shortThenLong)
		verifyMatches(t, shortThenLong, m, "shortThenLong")
	}
}

// TestMatchFinderTargetedCoverage exercises M4 overlap elimination, chain trimming, ZDFast lookahead, and Pathfinder Reset.
func TestMatchFinderTargetedCoverage(t *testing.T) {
	// Pathfinder Reset
	pf := &Pathfinder{MaxDistance: 256, ChainLength: 8}
	_ = pf.FindMatches(nil, bytes.Repeat([]byte("0123456789abcdef"), 10))
	pf.Reset()

	// ZDFast lookahead: short match at s, long 8-byte match at s+1
	// Candidate S matches "Q123", candidate L matches "12345678"
	zdfData := bytes.Repeat([]byte("PREFIX_Q123_MID_12345678_SUFFIX___"), 2)
	zdfData = append(zdfData, []byte("PADDING_BYTES_TO_SEPARATE_BLOCKS_") ...)
	zdfData = append(zdfData, []byte("MID_12345678_Q12345678_END")...)
	zdf := &ZDFast{MaxDistance: 4096}
	m := zdf.FindMatches(nil, zdfData)
	verifyMatches(t, zdfData, m, "zdfData")

	// ZFast candidate2 match at s+1
	zfData := []byte("PRE_0123456789_MID_ABCDEFGHIJK_END_")
	zfData = append(zfData, bytes.Repeat([]byte("X"), 50)...)
	zfData = append(zfData, []byte("!0123456789!ABCDEFGHIJK")...)
	zf := &ZFast{MaxDistance: 4096}
	m2 := zf.FindMatches(nil, zfData)
	verifyMatches(t, zfData, m2, "zfData")

	// M4 overlap scenarios:
	// Construct dictionary of substrings:
	// A: 20 bytes, B: 30 bytes, C: 60 bytes
	// We arrange occurrences so match 2, match 1, and match 0 overlap in ways triggering:
	// - Case 1: matches[0].Start < matches[2].End
	// - Case 2: matches[0].Start < matches[2].End + MinLength
	// - Case 3: matches[2].End > matches[1].Start (shorten match and chain lookup)
	patterns := [][]byte{
		// Pattern with step-overlapping substrings
		bytes.Repeat([]byte("0123456789abcdefghijklmnopqrstuvwxyz"), 10),
		// Tail-heavy overlapping pattern
		[]byte("AAAA_BBBB_CCCC_DDDD_EEEE_012345678901234567890123456789_" +
			"01234567890123456789012345678901234567890123456789_" +
			"AAAA_012345678901234567890123456789012345678901234567890123456789" +
			"BBBB_CCCC_0123456789012345678901234567890123456789"),
		// Cascading overlapping prefixes
		[]byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ_ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
			"BCDEFGHIJKLMNOPQRSTUVWXYZ_CDEFGHIJKLMNOPQRSTUVWXYZ_" +
			"DEFGHIJKLMNOPQRSTUVWXYZ_EFGHIJKLMNOPQRSTUVWXYZ_" +
			"FGHIJKLMNOPQRSTUVWXYZ_GHIJKLMNOPQRSTUVWXYZ"),
	}

	for _, pat := range patterns {
		for _, minLen := range []int{4, 6} {
			for _, chain := range []int{0, 4, 16, 64} {
				for _, distCost := range []int{0, 1, 4} {
					finder := &M4{
						MinLength:       minLen,
						ChainLength:     chain,
						DistanceBitCost: distCost,
						MaxDistance:     4096,
					}
					var matches []Match
					matches = finder.FindMatches(matches, pat)
					verifyMatches(t, pat, matches, "m4 overlap test")
				}
			}
		}
	}

	// Repeated overlapping triggers for Case 1, 2, 3 specifically
	// Match 2: at 100, len 16 (End 116)
	// Match 1: at 108, len 28 (End 136)
	// Match 0: at 112, len 60 (End 172) -> 112 < 116 -> Case 1
	var c1 bytes.Buffer
	c1.WriteString("REF:abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789extra_payload_longer_match!")
	c1.WriteString(string(bytes.Repeat([]byte("_"), 60)))
	// Substrings to match REF
	c1.WriteString("abcdefghijklmnop")                                                                                // 16 bytes
	c1.WriteString("ijklmnopqrstuvwxyz0123456789")                                                                   // overlaps previous
	c1.WriteString("mnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789extra_payload_longer_match!")        // overlaps both, much longer
	m4c1 := &M4{MinLength: 4, ChainLength: 16, DistanceBitCost: 0, MaxDistance: 4096}
	matchesC1 := m4c1.FindMatches(nil, c1.Bytes())
	verifyMatches(t, c1.Bytes(), matchesC1, "m4 case 1")

	// Case 2: matches[0].Start < matches[2].End + MinLength
	var c2 bytes.Buffer
	c2.WriteString("REF:abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789extra_payload_longer_match!")
	c2.WriteString(string(bytes.Repeat([]byte("_"), 60)))
	c2.WriteString("abcdefghijklmnop")                                                                                // 16 bytes
	c2.WriteString("1234")                                                                                            // small gap
	c2.WriteString("mnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789extra_payload_longer_match!")        // starts at End + 2
	m4c2 := &M4{MinLength: 4, ChainLength: 16, DistanceBitCost: 0, MaxDistance: 4096}
	matchesC2 := m4c2.FindMatches(nil, c2.Bytes())
	verifyMatches(t, c2.Bytes(), matchesC2, "m4 case 2")

	// Case 3: matches[2].End > matches[1].Start with chain replacement
	var c3 bytes.Buffer
	c3.WriteString("REPEAT_TOKEN_FOR_CHAINING_ABCDEFGHIJK_REPEAT_TOKEN_FOR_CHAINING_ABCDEFGHIJK_")
	c3.WriteString(string(bytes.Repeat([]byte("Z"), 80)))
	c3.WriteString("REPEAT_TOKEN_FOR_CHAINING_ABCDEFGHIJK")
	c3.WriteString("ABCDEFGHIJK_MORE_TEXT_TO_OVERLAP")
	c3.WriteString(string(bytes.Repeat([]byte("W"), 80)))
	c3.WriteString("FINAL_SEGMENT_FOR_ROOM_BETWEEN_MATCHES_12345678901234567890")
	m4c3 := &M4{MinLength: 4, ChainLength: 32, DistanceBitCost: 1, MaxDistance: 4096}
	matchesC3 := m4c3.FindMatches(nil, c3.Bytes())
	verifyMatches(t, c3.Bytes(), matchesC3, "m4 case 3")
}

// TestSpecificTriggersAndLazyM0 exercises M0 Lazy matching, ZDFast/ZM s+1 lookahead, and overlapping match cascades.
func TestSpecificTriggersAndLazyM0(t *testing.T) {
	// M0 with Lazy matching enabled
	// base has 4-byte match, base+1 has 16-byte match
	m0Data := []byte("PRE_ABCDEF_MID_1234567890123456_END_")
	m0Data = append(m0Data, bytes.Repeat([]byte("X"), 60)...)
	m0Data = append(m0Data, []byte("ABCD1234567890123456_PADDING_AT_END")...)
	m0 := &M0{Lazy: true, MaxDistance: 256, MaxLength: 100}
	m := m0.FindMatches(nil, m0Data)
	verifyMatches(t, m0Data, m, "m0 lazy")

	// ZDFast and ZM lookahead at s+1
	zdfLook := []byte("PREFIX___Q123___MIDDLE___12345678___SUFFIX___")
	zdfLook = append(zdfLook, bytes.Repeat([]byte("Y"), 60)...)
	zdfLook = append(zdfLook, []byte("TARGET___Q12345678___TAIL")...)
	for _, mf := range []MatchFinder{
		&ZDFast{MaxDistance: 2048},
		&ZM{MaxDistance: 2048},
	} {
		out := mf.FindMatches(nil, zdfLook)
		verifyMatches(t, zdfLook, out, "zdfLook")
	}

	// Exact cascade for Case 1 (matches[0].Start < matches[2].End):
	// refA (20 bytes), refB (40 bytes), refC (70 bytes)
	refA := "abcdefghijklmnopqrst"                                         // 20 bytes
	refB := "1234567890123456789012345678901234567890"                         // 40 bytes
	refC := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789____" // 68 bytes
	var cascade1 bytes.Buffer
	cascade1.WriteString("HEADER_")
	cascade1.WriteString(refA)
	cascade1.WriteString("_PAD1_")
	cascade1.WriteString(refB)
	cascade1.WriteString("_PAD2_")
	cascade1.WriteString(refC)
	cascade1.WriteString(string(bytes.Repeat([]byte("!"), 60)))
	// Now target where refA is at s, refB is at s+2, refC is at s+4
	cascade1.WriteString("TARGET_")
	cascade1.WriteString(refA[:2])
	cascade1.WriteString(refB[:2])
	cascade1.WriteString(refC)
	c1Data := cascade1.Bytes()
	for _, mf := range []MatchFinder{
		&M4{MinLength: 4, MaxDistance: 4096},
		&ZM{MaxDistance: 4096},
	} {
		out := mf.FindMatches(nil, c1Data)
		verifyMatches(t, c1Data, out, "cascade1")
	}

	// Exact cascade for Case 3 (matches[2].End > matches[1].Start):
	// refA at s, refB at s+10, refC at s+30
	var cascade3 bytes.Buffer
	cascade3.WriteString("HEADER3_")
	cascade3.WriteString(refA)
	cascade3.WriteString("_PAD3_")
	cascade3.WriteString(refA) // second occurrence of refA for chaining
	cascade3.WriteString("_PAD4_")
	cascade3.WriteString(refB)
	cascade3.WriteString("_PAD5_")
	cascade3.WriteString(refC)
	cascade3.WriteString(string(bytes.Repeat([]byte("@"), 60)))
	cascade3.WriteString("TARGET3_")
	cascade3.WriteString(refA[:10])
	cascade3.WriteString(refB[:20])
	cascade3.WriteString(refC)
	c3Data := cascade3.Bytes()
	for _, mf := range []MatchFinder{
		&M4{MinLength: 4, ChainLength: 16, MaxDistance: 4096},
		&ZM{MaxDistance: 4096},
	} {
		out := mf.FindMatches(nil, c3Data)
		verifyMatches(t, c3Data, out, "cascade3")
	}
}

// TestExtendMatch386 tests the 32-bit comparison loop in extendMatch.
func TestExtendMatch386(t *testing.T) {
	orig := matchFinderArch
	defer func() { matchFinderArch = orig }()
	matchFinderArch = "386"

	src := []byte("0123456789abcdef0123456789abcdef")
	k := extendMatch(src, 0, 16)
	if k != len(src) {
		t.Fatalf("extendMatch 386 failed: got %d, want %d", k, len(src))
	}
	// partial mismatch
	src2 := []byte("01234567XXXX01234567YYYY")
	k2 := extendMatch(src2, 0, 12)
	if k2 != 20 {
		t.Fatalf("extendMatch 386 mismatch: got %d, want 20", k2)
	}
}

// TestZDFastAndZFastLookahead tests candidate lookahead branches in ZDFast and ZFast.
func TestZDFastAndZFastLookahead(t *testing.T) {
	// ZDFast lines 147-162:
	var b1 bytes.Buffer
	b1.WriteString("abcdefgh")
	b1.WriteString("Q123")
	b1.WriteString("ijklmnop")
	b1.WriteString("12345678")
	b1.WriteString("qrstuvwx")
	b1.WriteString(string(bytes.Repeat([]byte{0xff}, 32)))
	b1.WriteString("yz!@#$")
	b1.WriteString("Q12345678")
	b1.WriteString("%^&*()")
	zdf := &ZDFast{MaxDistance: 4096}
	m1 := zdf.FindMatches(nil, b1.Bytes())
	verifyMatches(t, b1.Bytes(), m1, "zdfLookahead")

	// ZFast lines 136-141:
	var b2 bytes.Buffer
	b2.WriteString("abcdefgh")
	b2.WriteString("5678")
	b2.WriteString("ijklmnop")
	b2.WriteString(string(bytes.Repeat([]byte{0xfe}, 32)))
	b2.WriteString("qrstuv")
	b2.WriteString("Z5678")
	b2.WriteString("wxyz")
	zf := &ZFast{MaxDistance: 4096}
	m2 := zf.FindMatches(nil, b2.Bytes())
	verifyMatches(t, b2.Bytes(), m2, "zfCandidate2")
}

// TestM4Case1And3Cascade forces M4 overlapping switch cases 1 and 3.
func TestM4Case1And3Cascade(t *testing.T) {
	// For Case 3: Match A (50..89), Match B (85..150), Match C (146..250)
	// We build a reference dictionary with distinct 4-byte hashes
	var ref bytes.Buffer
	// A at 0..39
	for i := 0; i < 40; i++ {
		ref.WriteByte(byte('A' + (i % 26)))
	}
	// duplicate A at 40..79 for chaining
	for i := 0; i < 40; i++ {
		ref.WriteByte(byte('A' + (i % 26)))
	}
	// B at 80..144 (65 bytes)
	for i := 0; i < 65; i++ {
		ref.WriteByte(byte('0' + (i % 10)))
	}
	// C at 145..249 (105 bytes)
	for i := 0; i < 105; i++ {
		ref.WriteByte(byte('a' + (i % 26)))
	}
	refBytes := ref.Bytes()

	var testBuf bytes.Buffer
	testBuf.Write(refBytes)
	testBuf.Write(bytes.Repeat([]byte{0xaa}, 100)) // separation
	// Emit A (starts at 50)
	testBuf.Write(refBytes[:39]) // Match A: ends at current pos
	// Now B is placed 4 bytes before end of A, so B starts at A.End - 4
	testBuf.Truncate(testBuf.Len() - 4)
	testBuf.Write(refBytes[80:145]) // Match B: 65 bytes
	// Now C is placed 4 bytes before end of B
	testBuf.Truncate(testBuf.Len() - 4)
	testBuf.Write(refBytes[145:250]) // Match C: 105 bytes
	testBuf.Write(bytes.Repeat([]byte{0xbb}, 50))

	m4 := &M4{MinLength: 4, ChainLength: 16, HashLen: 6, MaxDistance: 4096}
	m := m4.FindMatches(nil, testBuf.Bytes())
	verifyMatches(t, testBuf.Bytes(), m, "m4 case 3 cascade")

	// Case 1: Match C starts before Match A ends
	// HashLen = 4 so i == End - 2
	var testBufC1 bytes.Buffer
	testBufC1.Write(refBytes)
	testBufC1.Write(bytes.Repeat([]byte{0xcc}, 100))
	testBufC1.Write(refBytes[:20]) // Match A (len 20)
	testBufC1.Truncate(testBufC1.Len() - 2)
	testBufC1.Write(refBytes[80:112]) // Match B (len 32)
	testBufC1.Truncate(testBufC1.Len() - 2)
	testBufC1.Write(refBytes[145:240]) // Match C (len 95)
	testBufC1.Write(bytes.Repeat([]byte{0xdd}, 50))

	m4_c1 := &M4{MinLength: 4, HashLen: 4, MaxDistance: 4096}
	m_c1 := m4_c1.FindMatches(nil, testBufC1.Bytes())
	verifyMatches(t, testBufC1.Bytes(), m_c1, "m4 case 1 cascade")
}

// TestZDFastCandidateLookahead forces ZDFast lines 147-162.
func TestZDFastCandidateLookahead(t *testing.T) {
	var b bytes.Buffer
	b.WriteString("BCDEFGHIJKLMNOPQRSTUVWXYZ")
	b.WriteString("Q1234") // candidateS (hashShort hashes 5 bytes!)
	b.WriteString("abcdefghijklmnopqrstuvwxyz")
	b.WriteString("12345678") // candidateL (hashLong hashes 8 bytes!)
	b.WriteString("!@#$%^&*()_+-=[]{}|;':,./<>?")
	b.WriteString("Q12345678") // s is at Q1234, s+1 is 12345678
	b.WriteString("~`END_OF_TEST_BUFFER_12345")
	zdf := &ZDFast{MaxDistance: 4096}
	m := zdf.FindMatches(nil, b.Bytes())
	verifyMatches(t, b.Bytes(), m, "zdfLookaheadUnique")
}

// TestZFastCandidateLookahead forces ZFast lines 136-141.
func TestZFastCandidateLookahead(t *testing.T) {
	var b bytes.Buffer
	b.WriteString("BCDEFGHIJKLMNOPQRSTUVWXYZ")
	b.WriteString("567890") // candidate2 (hash takes 6 bytes!)
	b.WriteString("abcdefghijklmnopqrstuvwxyz")
	b.WriteString("!@#$%^&*()_+-=[]{}|;':,./<>?")
	b.WriteString("Z567890") // s is at Z (no match), s+1 is 567890 (candidate2 match!)
	b.WriteString("~`END_OF_TEST_BUFFER_12345")
	zf := &ZFast{MaxDistance: 4096}
	m := zf.FindMatches(nil, b.Bytes())
	verifyMatches(t, b.Bytes(), m, "zfCandidate2Unique")
}

// TestTextEncoderTrailing exercises TextEncoder trailing literals and NoMatchFinder.
func TestTextEncoderTrailing(t *testing.T) {
	te := TextEncoder{}
	te.Reset()
	src := []byte("hello world trailing")
	matches := []Match{{Unmatched: 5, Length: 0, Distance: 0}}
	enc := te.Encode(nil, src, matches, true)
	if !bytes.Equal(enc, src) {
		t.Fatalf("mismatch: got %q, want %q", enc, src)
	}
	nm := NoMatchFinder{}
	nm.Reset()
	m := nm.FindMatches(nil, src)
	if len(m) != 1 || m[0].Unmatched != len(src) {
		t.Fatalf("unexpected matches from NoMatchFinder")
	}
}

// TestOffset2MatchDirect exercises repeated match offset2 in ZFast and ZDFast.
func TestOffset2MatchDirect(t *testing.T) {
	tokenA := []byte("ABCDEFGH12345678")
	tokenB := []byte("IJKLMNOP87654321")
	pad16 := []byte("!@#$%^&*()_+~`<>")
	pad52 := append(bytes.Repeat([]byte("0123456789"), 5), []byte("xy")...)
	pad84 := append(bytes.Repeat([]byte("abcdefghij"), 8), []byte("1234")...)
	pad30 := bytes.Repeat([]byte("END_OF_TEST_"), 3)

	var buf bytes.Buffer
	buf.Write(tokenA) // 0..16
	buf.Write(pad16)  // 16..32
	buf.Write(tokenA) // 32..48
	buf.Write(pad52)  // 48..100
	buf.Write(tokenB) // 100..116
	buf.Write(pad84)  // 116..200
	buf.Write(tokenA) // 200..216 -> matches pos 0 (offset1 = 200)
	buf.Write(tokenB) // 216..232 -> matches pos 100 (offset2 = 200, offset1 = 116)
	buf.Write(tokenA) // 232..248 -> o2 = 232 - 200 = 32 (matches pos 32 via offset2!)
	buf.Write(pad30)

	data := buf.Bytes()
	for _, mf := range []MatchFinder{
		&ZFast{MaxDistance: 4096},
		&ZDFast{MaxDistance: 4096},
	} {
		m := mf.FindMatches(nil, data)
		verifyMatches(t, data, m, "offset2 test")
	}
}

// TestRepeatedOffsetSeparated tests repeated offset2 matches using non-colliding separators.
func TestRepeatedOffsetSeparated(t *testing.T) {
	tokA := "0123456789abcdef"
	tokB := "GHIJKLMNOPQRSTUV"
	var buf bytes.Buffer
	buf.WriteString("PREFIX_" + tokA + "_MID_" + tokB + "_SUF_")
	buf.WriteString(string(bytes.Repeat([]byte("~"), 60)))
	puncs := []string{"!", "@", "#", "$", "%", "^", "&", "*", "(", ")"}
	for _, p := range puncs {
		buf.WriteString(tokA + p + tokB + "_")
	}
	data := buf.Bytes()
	for _, mf := range []MatchFinder{
		&ZFast{MaxDistance: 4096},
		&ZDFast{MaxDistance: 4096},
		&ZM{MaxDistance: 4096},
		&Trio{MaxDistance: 4096},
	} {
		m := mf.FindMatches(nil, data)
		verifyMatches(t, data, m, "separated offset2")
	}
}

// TestPathfinderEdgeBranches exercises candidate distance limits and sequential fallback in pathfinder.
func TestPathfinderEdgeBranches(t *testing.T) {
	pfSmall := &Pathfinder{MaxDistance: 16, MinLength: 4}
	data1 := bytes.Repeat([]byte("0123456789abcdef"), 4)
	_ = pfSmall.FindMatches(nil, data1)

	var b bytes.Buffer
	token := []byte("0123456789ABCDEFGHIJKLMNOPQRST") // 30 bytes
	b.Write(token)
	b.WriteString("_MIDDLE_UNIQUE_PADDING_TEXT_")
	b.Write(token)
	b.WriteString("_TRAILING_UNIQUE_BYTES_12345_")
	pf := &Pathfinder{MaxDistance: 4096, MinLength: 4, HashLen: 6}
	m := pf.FindMatches(nil, b.Bytes())
	verifyMatches(t, b.Bytes(), m, "pathfinder edge")
}








