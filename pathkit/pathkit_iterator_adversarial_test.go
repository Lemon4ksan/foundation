// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pathkit_test

import (
	"slices"
	"testing"

	"github.com/lemon4ksan/foundation/pathkit"
)

func TestEmpirical_Pathkit_SegmentsSeq_EarlyTermination(t *testing.T) {
	path := "/usr/local/bin/foundation/exec"

	// 1. Break immediately after 0 iterations (first item returns false)
	count := 0
	for seg := range pathkit.SegmentsSeq(path) {
		_ = seg
		count++
		break
	}
	if count != 1 {
		t.Fatalf("expected count 1 on immediate break, got %d", count)
	}

	// 2. Break after 2 segments
	count = 0
	for seg := range pathkit.SegmentsSeq(path) {
		_ = seg
		count++
		if count == 2 {
			break
		}
	}
	if count != 2 {
		t.Fatalf("expected count 2, got %d", count)
	}

	// 3. Full traversal
	var segments []string
	for seg := range pathkit.SegmentsSeq(path) {
		segments = append(segments, seg)
	}
	expected := []string{"usr", "local", "bin", "foundation", "exec"}
	if !slices.Equal(segments, expected) {
		t.Fatalf("expected segments %v, got %v", expected, segments)
	}

	// 4. Break mid-iteration
	mid := len(expected) / 2
	visited := 0
	for range pathkit.SegmentsSeq(path) {
		visited++
		if visited == mid {
			break
		}
	}
	if visited != mid {
		t.Fatalf("expected visited %d, got %d", mid, visited)
	}
}

func TestEmpirical_Pathkit_SegmentsSeq_EdgeCases(t *testing.T) {
	// Empty string
	for range pathkit.SegmentsSeq("") {
		t.Fatal("expected no segments for empty string")
	}

	// Slashes only
	for range pathkit.SegmentsSeq("////") {
		t.Fatal("expected no segments for slashes only")
	}

	// URL format
	urlPath := "https://api.github.com/v1/repos/foundation"
	var urlSegs []string
	for seg := range pathkit.SegmentsSeq(urlPath) {
		urlSegs = append(urlSegs, seg)
	}
	expectedURL := []string{"v1", "repos", "foundation"}
	if !slices.Equal(urlSegs, expectedURL) {
		t.Fatalf("expected %v, got %v", expectedURL, urlSegs)
	}

	// Path object methods
	p := pathkit.New("/var/log/syslog")
	var pSegs []string
	for seg := range p.SegmentsSeq() {
		pSegs = append(pSegs, seg)
	}
	if !slices.Equal(pSegs, []string{"var", "log", "syslog"}) {
		t.Fatalf("expected [var log syslog], got %v", pSegs)
	}
}

func BenchmarkSegmentsSeq_ZeroAlloc(b *testing.B) {
	path := "/usr/local/bin/foundation/exec"

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		for seg := range pathkit.SegmentsSeq(path) {
			_ = seg
		}
	}
}
