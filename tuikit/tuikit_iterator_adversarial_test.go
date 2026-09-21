// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tuikit_test

import (
	"slices"
	"testing"

	"github.com/lemon4ksan/foundation/tuikit"
)

func TestEmpirical_Tuikit_LinesSeq_EarlyTermination(t *testing.T) {
	text := "first line\r\nsecond line\nthird line\r\nfourth line\nfifth line"

	// 1. Break immediately after 0 iterations (first item returns false)
	count := 0
	for line := range tuikit.LinesSeq(text) {
		_ = line
		count++
		break
	}
	if count != 1 {
		t.Fatalf("expected count 1 on immediate break, got %d", count)
	}

	// 2. Break after 2 lines
	count = 0
	for line := range tuikit.LinesSeq(text) {
		_ = line
		count++
		if count == 2 {
			break
		}
	}
	if count != 2 {
		t.Fatalf("expected count 2, got %d", count)
	}

	// 3. Full traversal
	var lines []string
	for line := range tuikit.LinesSeq(text) {
		lines = append(lines, line)
	}
	expected := []string{"first line", "second line", "third line", "fourth line", "fifth line"}
	if !slices.Equal(lines, expected) {
		t.Fatalf("expected lines %v, got %v", expected, lines)
	}

	// 4. Break mid-iteration
	mid := len(expected) / 2
	visited := 0
	for range tuikit.LinesSeq(text) {
		visited++
		if visited == mid {
			break
		}
	}
	if visited != mid {
		t.Fatalf("expected visited %d, got %d", mid, visited)
	}
}

func TestEmpirical_Tuikit_LinesSeq_EdgeCases(t *testing.T) {
	// Empty string
	for range tuikit.LinesSeq("") {
		t.Fatal("expected no lines for empty string")
	}

	// Single line without newline
	single := "single line without newline"
	var singleLines []string
	for l := range tuikit.LinesSeq(single) {
		singleLines = append(singleLines, l)
	}
	if len(singleLines) != 1 || singleLines[0] != single {
		t.Fatalf("expected [single line without newline], got %v", singleLines)
	}

	// Only CRLF
	crlf := "\r\n\r\n\r\n"
	var crlfLines []string
	for l := range tuikit.LinesSeq(crlf) {
		crlfLines = append(crlfLines, l)
	}
	// 3 newlines yield 3 empty lines
	if len(crlfLines) != 3 {
		t.Fatalf("expected 3 empty lines, got %d: %v", len(crlfLines), crlfLines)
	}
	for i, l := range crlfLines {
		if l != "" {
			t.Errorf("expected line %d to be empty, got %q", i, l)
		}
	}
}

func BenchmarkLinesSeq_ZeroAlloc(b *testing.B) {
	text := "header\r\nservice: worker\r\nthroughput: 100k\r\nstatus: ok\r\nfooter"

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		for line := range tuikit.LinesSeq(text) {
			_ = line
		}
	}
}
