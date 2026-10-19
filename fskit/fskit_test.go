// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fskit_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/lemon4ksan/foundation/fskit"
)

func createTestTree(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "sub1", "sub2"), 0o755); err != nil {
		t.Fatalf("failed to create directories: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("1"), 0o644); err != nil {
		t.Fatalf("failed to write file1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub1", "file2.txt"), []byte("22"), 0o644); err != nil {
		t.Fatalf("failed to write file2: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub1", "sub2", "file3.txt"), []byte("333"), 0o644); err != nil {
		t.Fatalf("failed to write file3: %v", err)
	}
	return dir
}

// -----------------------------------------------------------------------------
// Directory Traversal Tests
// -----------------------------------------------------------------------------

func TestFastWalk_Basic(t *testing.T) {
	tempDir := createTestTree(t)

	found := make(map[string]bool)
	err := fskit.FastWalk(tempDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			rel, _ := filepath.Rel(tempDir, path)
			found[filepath.ToSlash(rel)] = true

			fi, err := d.Info()
			if err != nil {
				t.Errorf("d.Info() failed: %v", err)
			}
			if fi.Size() < 0 {
				t.Errorf("invalid size %d", fi.Size())
			}
			if fi.ModTime().IsZero() {
				t.Errorf("modtime is zero")
			}
			if fi.Name() != d.Name() {
				t.Errorf("expected name %s, got %s", d.Name(), fi.Name())
			}
			if fi.IsDir() != d.IsDir() {
				t.Errorf("mismatch in IsDir: %v vs %v", fi.IsDir(), d.IsDir())
			}
			if d.Type() != fi.Mode().Type() {
				t.Errorf("mismatch in Type: %v vs %v", d.Type(), fi.Mode().Type())
			}
			if fi.Sys() != nil {
				t.Errorf("expected nil Sys, got %v", fi.Sys())
			}
			if unwrap, ok := d.(interface{ Entry() fskit.Entry }); ok {
				ent := unwrap.Entry()
				if ent.Path != path {
					t.Errorf("entry unwrap path mismatch: %s vs %s", ent.Path, path)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("FastWalk failed: %v", err)
	}

	expected := []string{"file1.txt", "sub1/file2.txt", "sub1/sub2/file3.txt"}
	for _, exp := range expected {
		if !found[exp] {
			t.Errorf("expected to find %s, but missing", exp)
		}
	}
}

func TestFastWalk_SingleFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "single.txt")
	if err := os.WriteFile(filePath, []byte("single"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	var visited []string
	err := fskit.FastWalk(filePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		visited = append(visited, path)
		return nil
	})
	if err != nil {
		t.Fatalf("FastWalk on single file failed: %v", err)
	}
	if len(visited) != 1 || visited[0] != filePath {
		t.Fatalf("expected visited [single.txt], got %v", visited)
	}
}

func TestFastWalk_NonExistent(t *testing.T) {
	err := fskit.FastWalk(filepath.Join(t.TempDir(), "nonexistent"), func(path string, d fs.DirEntry, err error) error {
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil error on nonexistent path, got %v", err)
	}
}

func TestFastWalk_OptionsAndFilter(t *testing.T) {
	dir := createTestTree(t)

	var visited []string
	err := fskit.FastWalk(dir, func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			visited = append(visited, filepath.Base(path))
		}
		return nil
	},
		fskit.WithWorkers(2),
		fskit.WithBufferSize(16),
		fskit.WithFollowSymlinks(false),
		fskit.WithFilter(func(path string, isDir bool) bool {
			// Skip sub2 directory
			return !strings.Contains(path, "sub2")
		}),
	)
	if err != nil {
		t.Fatalf("FastWalk with filter failed: %v", err)
	}

	for _, v := range visited {
		if v == "file3.txt" {
			t.Fatalf("file3.txt inside sub2 should have been filtered out")
		}
	}
}

func TestFastWalk_EarlyErrorAndSkip(t *testing.T) {
	dir := createTestTree(t)

	// Test 1: walkFn returns error
	sentinelErr := errors.New("abort walk")
	err := fskit.FastWalk(dir, func(path string, d fs.DirEntry, err error) error {
		return sentinelErr
	})
	if !errors.Is(err, sentinelErr) {
		t.Fatalf("expected sentinelErr, got %v", err)
	}

	// Test 2: walkFn returns filepath.SkipAll (should terminate with nil error)
	err = fskit.FastWalk(dir, func(path string, d fs.DirEntry, err error) error {
		return filepath.SkipAll
	})
	if err != nil {
		t.Fatalf("expected nil on SkipAll, got %v", err)
	}

	// Test 3: walkFn returns filepath.SkipDir
	err = fskit.FastWalk(dir, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil on SkipDir, got %v", err)
	}
}

func TestWalkContext_Cancelled(t *testing.T) {
	dir := createTestTree(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancelled

	ch := make(chan fskit.Entry, 10)
	fskit.WalkContext(ctx, dir, ch)

	// Channel must be closed promptly
	count := 0
	for range ch {
		count++
	}
	if count > 10 {
		t.Fatalf("unexpected entry count on cancelled walk: %d", count)
	}
}

func TestWalk_Direct(t *testing.T) {
	dir := createTestTree(t)
	ch := make(chan fskit.Entry, 100)
	fskit.Walk(dir, ch)

	var entries []fskit.Entry
	for e := range ch {
		entries = append(entries, e)
		fi := e.Info()
		if fi.Name() != e.Name() {
			t.Errorf("name mismatch: %s vs %s", fi.Name(), e.Name())
		}
		if e.IsDir && !fi.IsDir() {
			t.Errorf("expected IsDir for %s", e.Path)
		}
		if e.SizeBytes() != e.Size {
			t.Errorf("size mismatch: %d vs %d", e.SizeBytes(), e.Size)
		}
		if e.ModTimeUTC() != e.ModTime.UTC() {
			t.Errorf("modtime mismatch")
		}
		if e.Sys() != nil {
			t.Errorf("expected nil Sys on entry")
		}
	}
	if len(entries) < 3 {
		t.Fatalf("expected at least 3 entries, got %d", len(entries))
	}

	// Test Entry.Mode fallback branches
	dummyDir := fskit.Entry{IsDir: true}
	if dummyDir.Mode() != 0o755|fs.ModeDir {
		t.Errorf("expected 0755 for default dir mode, got %v", dummyDir.Mode())
	}
	dummyFile := fskit.Entry{IsDir: false}
	if dummyFile.Mode() != 0o644 {
		t.Errorf("expected 0644 for default file mode, got %v", dummyFile.Mode())
	}
	dummyCustom := fskit.Entry{FileMode: 0o600}
	if dummyCustom.Mode() != 0o600 {
		t.Errorf("expected 0600, got %v", dummyCustom.Mode())
	}

	// Trailing separator root walk
	trailingRoot := dir + string(filepath.Separator)
	chTrailing := make(chan fskit.Entry, 100)
	fskit.Walk(trailingRoot, chTrailing)
	for range chTrailing {
	}
}

func TestFastWalkSeq(t *testing.T) {
	dir := createTestTree(t)

	// Test full traversal
	count := 0
	for path, de := range fskit.FastWalkSeq(dir) {
		if path == "" || de == nil {
			t.Fatalf("invalid path or dirEntry")
		}
		count++
	}
	if count < 4 {
		t.Fatalf("expected at least 4 entries, got %d", count)
	}

	// Test early break at 1 item
	count = 0
	for range fskit.FastWalkSeq(dir) {
		count++
		break
	}
	if count != 1 {
		t.Fatalf("expected count 1, got %d", count)
	}

	// Test early break at 2 items
	count = 0
	for range fskit.FastWalkSeq(dir) {
		count++
		if count == 2 {
			break
		}
	}
	if count != 2 {
		t.Fatalf("expected count 2, got %d", count)
	}

	// Test goroutine leak
	before := runtime.NumGoroutine()
	for range 20 {
		for range fskit.FastWalkSeq(dir, fskit.WithWorkers(4)) {
			break
		}
	}
	runtime.GC()
	after := runtime.NumGoroutine()
	if after > before+2 {
		t.Fatalf("potential goroutine leak in FastWalkSeq: before=%d, after=%d", before, after)
	}
}

func TestWalkSeq(t *testing.T) {
	dir := createTestTree(t)

	// Case 1: Break immediately at 0 iterations
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
	if totalCount < 4 {
		t.Fatalf("expected at least 4 entries, got %d", totalCount)
	}

	// Case 4: Break mid-iteration
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

	// Case 5: No goroutine leaks
	before := runtime.NumGoroutine()
	for range 50 {
		for range fskit.WalkSeq(dir) {
			break
		}
	}
	runtime.GC()
	after := runtime.NumGoroutine()
	if after > before+2 {
		t.Fatalf("potential goroutine leak: before=%d, after=%d", before, after)
	}
}

// -----------------------------------------------------------------------------
// Memory-Mapped File Tests
// -----------------------------------------------------------------------------

func TestOpenMmap_Regular(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "mmap_test.bin")
	content := []byte("foundation-mmap-payload-data-1234567890")
	if err := os.WriteFile(filePath, content, 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	m, err := fskit.OpenMmap(filePath)
	if err != nil {
		t.Fatalf("OpenMmap failed: %v", err)
	}
	defer m.Close()

	if m.Len() != int64(len(content)) {
		t.Fatalf("expected length %d, got %d", len(content), m.Len())
	}

	data := m.Bytes()
	if !bytes.Equal(data, content) {
		t.Fatalf("data mismatch: expected %q, got %q", content, data)
	}

	// Test ReadAt
	buf := make([]byte, 10)
	n, err := m.ReadAt(buf, 0)
	if err != nil || n != 10 {
		t.Fatalf("ReadAt(0) failed: n=%d, err=%v", n, err)
	}
	if !bytes.Equal(buf, content[:10]) {
		t.Fatalf("buf mismatch: got %q", buf)
	}

	// ReadAt offset
	n, err = m.ReadAt(buf, 11)
	if err != nil || n != 10 {
		t.Fatalf("ReadAt(11) failed: n=%d, err=%v", n, err)
	}
	if !bytes.Equal(buf, content[11:21]) {
		t.Fatalf("buf mismatch: got %q", buf)
	}

	// ReadAt empty slice
	emptyBuf := make([]byte, 0)
	n, err = m.ReadAt(emptyBuf, 5)
	if err != nil || n != 0 {
		t.Fatalf("ReadAt emptyBuf failed: n=%d, err=%v", n, err)
	}

	// ReadAt past EOF
	longBuf := make([]byte, len(content)+10)
	n, err = m.ReadAt(longBuf, 0)
	if !errors.Is(err, io.EOF) || n != len(content) {
		t.Fatalf("expected EOF with partial read: n=%d, err=%v", n, err)
	}

	// ReadAt beyond EOF
	n, err = m.ReadAt(buf, int64(len(content)))
	if !errors.Is(err, io.EOF) || n != 0 {
		t.Fatalf("expected EOF at end: n=%d, err=%v", n, err)
	}

	// ReadAt negative offset
	n, err = m.ReadAt(buf, -1)
	if !errors.Is(err, io.EOF) || n != 0 {
		t.Fatalf("expected EOF on negative offset: n=%d, err=%v", n, err)
	}

	// Test Close and idempotency
	if err := m.Close(); err != nil {
		t.Fatalf("first Close failed: %v", err)
	}
	if err := m.Close(); err != nil {
		t.Fatalf("second Close failed: %v", err)
	}

	// ReadAt after Close
	_, err = m.ReadAt(buf, 0)
	if !errors.Is(err, os.ErrClosed) {
		t.Fatalf("expected os.ErrClosed after Close, got %v", err)
	}
}

func TestOpenMmap_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "empty.bin")
	if err := os.WriteFile(filePath, nil, 0o644); err != nil {
		t.Fatalf("failed to create empty file: %v", err)
	}

	m, err := fskit.OpenMmap(filePath)
	if err != nil {
		t.Fatalf("OpenMmap on empty file failed: %v", err)
	}
	defer m.Close()

	if m.Len() != 0 {
		t.Fatalf("expected length 0, got %d", m.Len())
	}
	if len(m.Bytes()) != 0 {
		t.Fatalf("expected empty bytes, got %d bytes", len(m.Bytes()))
	}

	buf := make([]byte, 10)
	n, err := m.ReadAt(buf, 0)
	if !errors.Is(err, io.EOF) || n != 0 {
		t.Fatalf("expected EOF on empty file read: n=%d, err=%v", n, err)
	}

	if err := m.Close(); err != nil {
		t.Fatalf("Close empty file failed: %v", err)
	}
}

func TestOpenMmap_NonExistent(t *testing.T) {
	_, err := fskit.OpenMmap(filepath.Join(t.TempDir(), "nonexistent.bin"))
	if err == nil {
		t.Fatalf("expected error for nonexistent file, got nil")
	}
}

// -----------------------------------------------------------------------------
// Media Heuristics Tests
// -----------------------------------------------------------------------------

func TestIsIncompressibleExtension(t *testing.T) {
	compressedExts := []string{
		".zip", ".7z", ".rar", ".gz", ".tgz",
		".bz2", ".xz", ".zst", ".eon",
		".mp4", ".mkv", ".avi", ".mov", ".webm",
		".mp3", ".aac", ".ogg", ".flac",
		".jpg", ".jpeg", ".png", ".webp", ".gif",
		".pdf", ".docx", ".xlsx", ".pptx",
		".jar", ".apk", ".iso",
	}

	for _, ext := range compressedExts {
		// Test lowercase with dot
		if !fskit.IsIncompressibleExtension("test" + ext) {
			t.Errorf("expected %s to be incompressible", ext)
		}
		// Test uppercase
		if !fskit.IsIncompressibleExtension("test" + strings.ToUpper(ext)) {
			t.Errorf("expected uppercase %s to be incompressible", ext)
		}
		// Test without dot via IsIncompressibleExt
		raw := strings.TrimPrefix(ext, ".")
		if !fskit.IsIncompressibleExt(raw) {
			t.Errorf("expected IsIncompressibleExt(%q) = true", raw)
		}
		if !fskit.IsIncompressibleExt(strings.ToUpper(raw)) {
			t.Errorf("expected IsIncompressibleExt(uppercase %q) = true", raw)
		}
		// Test bytes variants
		if !fskit.IsIncompressibleExtensionBytes([]byte("test" + ext)) {
			t.Errorf("expected bytes %s to be incompressible", ext)
		}
		if !fskit.IsIncompressibleExtBytes([]byte(raw)) {
			t.Errorf("expected IsIncompressibleExtBytes(%q) = true", raw)
		}
	}

	// Test non-incompressible extensions
	compressible := []string{
		"file.txt", "main.go", "index.html", "style.css", "data.json",
		"script.py", "lib.rs", "README", ".gitignore", "", "noext",
		"folder.zip/inside.txt",
	}
	for _, name := range compressible {
		if fskit.IsIncompressibleExtension(name) {
			t.Errorf("expected %q NOT to be incompressible", name)
		}
		if fskit.IsIncompressibleExtensionBytes([]byte(name)) {
			t.Errorf("expected bytes %q NOT to be incompressible", name)
		}
	}

	// Test zero heap allocations
	testPath := "some/path/to/archive.tar.gz"
	allocs := testing.AllocsPerRun(100, func() {
		_ = fskit.IsIncompressibleExtension(testPath)
	})
	if allocs != 0 {
		t.Fatalf("expected 0 allocations for IsIncompressibleExtension, got %f", allocs)
	}

	testExt := "docx"
	allocs = testing.AllocsPerRun(100, func() {
		_ = fskit.IsIncompressibleExt(testExt)
	})
	if allocs != 0 {
		t.Fatalf("expected 0 allocations for IsIncompressibleExt, got %f", allocs)
	}

	testBytes := []byte("photo.JPEG")
	allocs = testing.AllocsPerRun(100, func() {
		_ = fskit.IsIncompressibleExtensionBytes(testBytes)
	})
	if allocs != 0 {
		t.Fatalf("expected 0 allocations for IsIncompressibleExtensionBytes, got %f", allocs)
	}
}

// -----------------------------------------------------------------------------
// File Utilities Tests
// -----------------------------------------------------------------------------

func TestAtomicWriteFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "atomic.txt")
	data1 := []byte("first-write")

	// First atomic write
	if err := fskit.AtomicWriteFile(target, data1, 0o644); err != nil {
		t.Fatalf("first AtomicWriteFile failed: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil || !bytes.Equal(got, data1) {
		t.Fatalf("read mismatch: %q, err=%v", got, err)
	}

	// Overwrite atomically
	data2 := []byte("second-atomic-overwrite-data")
	if err := fskit.AtomicWriteFile(target, data2, 0o644); err != nil {
		t.Fatalf("second AtomicWriteFile failed: %v", err)
	}
	got, err = os.ReadFile(target)
	if err != nil || !bytes.Equal(got, data2) {
		t.Fatalf("read overwrite mismatch: %q, err=%v", got, err)
	}

	// Atomic write into invalid dir
	badTarget := filepath.Join(dir, "missing_dir", "sub", "test.txt")
	if err := fskit.AtomicWriteFile(badTarget, []byte("fail"), 0o644); err == nil {
		t.Fatalf("expected error on invalid directory, got nil")
	}
}

func TestReadFileLimited(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "limited.txt")
	content := []byte("0123456789")
	if err := os.WriteFile(filePath, content, 0o644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Read with ample limit
	data, err := fskit.ReadFileLimited(filePath, 20)
	if err != nil || !bytes.Equal(data, content) {
		t.Fatalf("ReadFileLimited failed: %q, err=%v", data, err)
	}

	// Read with exact limit
	data, err = fskit.ReadFileLimited(filePath, 10)
	if err != nil || !bytes.Equal(data, content) {
		t.Fatalf("ReadFileLimited exact failed: %q, err=%v", data, err)
	}

	// Read with limit smaller than file
	_, err = fskit.ReadFileLimited(filePath, 5)
	if err == nil {
		t.Fatalf("expected error when limit is exceeded, got nil")
	}

	// Negative limit
	_, err = fskit.ReadFileLimited(filePath, -1)
	if err == nil {
		t.Fatalf("expected error on negative limit, got nil")
	}

	// Non-existent file
	_, err = fskit.ReadFileLimited(filepath.Join(dir, "nonexistent.txt"), 100)
	if err == nil {
		t.Fatalf("expected error on nonexistent file, got nil")
	}
}

// -----------------------------------------------------------------------------
// Benchmarks
// -----------------------------------------------------------------------------

func BenchmarkFastWalk(b *testing.B) {
	dir := b.TempDir()
	for i := range 20 {
		_ = os.WriteFile(filepath.Join(dir, "file"+string(rune('a'+i))+".txt"), []byte("data"), 0o644)
	}

	b.ReportAllocs()

	for b.Loop() {
		_ = fskit.FastWalk(dir, func(path string, d fs.DirEntry, err error) error {
			return nil
		})
	}
}

func BenchmarkFastWalkSeq(b *testing.B) {
	dir := b.TempDir()
	for i := range 20 {
		_ = os.WriteFile(filepath.Join(dir, "file"+string(rune('a'+i))+".txt"), []byte("data"), 0o644)
	}

	b.ReportAllocs()

	for b.Loop() {
		for p, de := range fskit.FastWalkSeq(dir) {
			_ = p
			_ = de
		}
	}
}

func BenchmarkWalkSeq(b *testing.B) {
	dir := b.TempDir()
	for i := range 20 {
		_ = os.WriteFile(filepath.Join(dir, "file"+string(rune('a'+i))+".txt"), []byte("data"), 0o644)
	}

	b.ReportAllocs()

	for b.Loop() {
		for p, de := range fskit.WalkSeq(dir) {
			_ = p
			_ = de
		}
	}
}

func BenchmarkIsIncompressibleExtension(b *testing.B) {
	name := "path/to/archive/file.tar.gz"

	b.ReportAllocs()

	for b.Loop() {
		_ = fskit.IsIncompressibleExtension(name)
	}
}

func BenchmarkOpenMmapReadAt(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "bench.bin")
	_ = os.WriteFile(path, bytes.Repeat([]byte("A"), 64*1024), 0o644)

	m, err := fskit.OpenMmap(path)
	if err != nil {
		b.Fatalf("OpenMmap failed: %v", err)
	}
	defer m.Close()

	buf := make([]byte, 1024)
	b.ReportAllocs()

	for b.Loop() {
		_, _ = m.ReadAt(buf, 0)
	}
}
