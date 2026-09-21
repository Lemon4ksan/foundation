// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fskit_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/lemon4ksan/foundation/fskit"
)

func createTestTree(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "sub1", "sub2"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("1"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "sub1", "file2.txt"), []byte("2"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "sub1", "sub2", "file3.txt"), []byte("3"), 0o644)
	return dir
}

func TestEmpirical_WalkSeq_EarlyTermination(t *testing.T) {
	dir := createTestTree(t)

	// Case 1: Break immediately at 0 iterations (first item returns false)
	count := 0
	for path, de := range fskit.WalkSeq(dir) {
		_ = path
		_ = de
		count++
		break
	}
	if count != 1 {
		t.Fatalf("expected count 1 on immediate break, got %d", count)
	}

	// Case 2: Break after exactly 2 items
	count = 0
	for path, de := range fskit.WalkSeq(dir) {
		_ = path
		_ = de
		count++
		if count == 2 {
			break
		}
	}
	if count != 2 {
		t.Fatalf("expected count 2 on early break, got %d", count)
	}

	// Case 3: Complete traversal count
	totalCount := 0
	for path, de := range fskit.WalkSeq(dir) {
		_ = path
		_ = de
		totalCount++
	}
	// Root + file1.txt + sub1 + sub1/file2.txt + sub1/sub2 + sub1/sub2/file3.txt = 6 entries
	if totalCount < 4 {
		t.Fatalf("expected at least 4 entries, got %d", totalCount)
	}

	// Case 4: Break mid-iteration (totalCount / 2)
	mid := totalCount / 2
	visited := 0
	for path, de := range fskit.WalkSeq(dir) {
		_ = path
		_ = de
		visited++
		if visited == mid {
			break
		}
	}
	if visited != mid {
		t.Fatalf("expected visited %d, got %d", mid, visited)
	}
}

func TestEmpirical_WalkSeq_NoGoroutineLeak(t *testing.T) {
	dir := createTestTree(t)
	before := runtime.NumGoroutine()

	for range 100 {
		for range fskit.WalkSeq(dir) {
			break // immediate termination
		}
	}

	runtime.GC()
	after := runtime.NumGoroutine()
	if after > before+2 {
		t.Fatalf("potential goroutine leak: before=%d, after=%d", before, after)
	}
}

func BenchmarkWalkSeq_Traverse(b *testing.B) {
	dir := b.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "f1.txt"), []byte("a"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "f2.txt"), []byte("b"), 0o644)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		for p, de := range fskit.WalkSeq(dir) {
			_ = p
			_ = de
		}
	}
}
