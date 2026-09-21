// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vfs_test

import (
	"bytes"
	"io"
	"io/fs"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/vfs"
)

type trackedReadCloser struct {
	io.Reader
	closed atomic.Bool
}

func (t *trackedReadCloser) Close() error {
	t.closed.Store(true)
	return nil
}

type mockVFSEntry struct {
	name    string
	isDir   bool
	content []byte
	stream  *trackedReadCloser
}

func (m *mockVFSEntry) EntryName() string { return m.name }
func (m *mockVFSEntry) EntrySize() int64  { return int64(len(m.content)) }
func (m *mockVFSEntry) EntryMode() fs.FileMode {
	if m.isDir {
		return fs.ModeDir | 0o755
	}
	return 0o644
}
func (m *mockVFSEntry) EntryModTime() time.Time { return time.Unix(1000, 0) }
func (m *mockVFSEntry) EntryIsDir() bool        { return m.isDir }
func (m *mockVFSEntry) OpenStream() (io.ReadCloser, error) {
	rc := &trackedReadCloser{Reader: bytes.NewReader(m.content)}
	m.stream = rc
	return rc, nil
}

type mockProvider struct {
	entries []vfs.VFSEntry
}

func (p *mockProvider) GetEntries() []vfs.VFSEntry {
	return p.entries
}

func (p *mockProvider) FindEntry(name string) (vfs.VFSEntry, bool) {
	for _, e := range p.entries {
		if e.EntryName() == name {
			return e, true
		}
	}
	return nil, false
}

func setupMockVFS() (*vfs.FS, []*mockVFSEntry) {
	items := []*mockVFSEntry{
		{name: "dir1", isDir: true},
		{name: "file1.txt", isDir: false, content: []byte("content 1")},
		{name: "file2.txt", isDir: false, content: []byte("content 2")},
		{name: "file3.log", isDir: false, content: []byte("content 3")},
		{name: "file4.txt", isDir: false, content: []byte("content 4")},
	}
	entries := make([]vfs.VFSEntry, len(items))
	for i, it := range items {
		entries[i] = it
	}
	p := &mockProvider{entries: entries}
	return vfs.NewFS(p), items
}

func TestEmpirical_VFS_EntriesSeq_EarlyTermination(t *testing.T) {
	fsys, items := setupMockVFS()

	// Nil receiver safety
	var nilFS *vfs.FS
	for range nilFS.EntriesSeq() {
		t.Fatal("expected no entries for nil FS")
	}

	// 1. Break after 0 iterations (immediate break)
	count := 0
	for range fsys.EntriesSeq() {
		count++
		break
	}
	if count != 1 {
		t.Fatalf("expected count 1 on immediate break, got %d", count)
	}

	// 2. Break after 2 iterations
	count = 0
	for range fsys.EntriesSeq() {
		count++
		if count == 2 {
			break
		}
	}
	if count != 2 {
		t.Fatalf("expected count 2, got %d", count)
	}

	// 3. Full traversal
	count = 0
	for range fsys.EntriesSeq() {
		count++
	}
	if count != len(items) {
		t.Fatalf("expected full count %d, got %d", len(items), count)
	}
}

func TestEmpirical_VFS_OpenSeq_EarlyTerminationAndNoLeak(t *testing.T) {
	fsys, items := setupMockVFS()

	// 1. Break after first yield: check that OpenSeq automatically closes the file on !yield
	opened := 0
	for name, f := range fsys.OpenSeq("*.txt") {
		_ = name
		_ = f
		opened++
		break // yield returns false
	}
	if opened != 1 {
		t.Fatalf("expected 1 file opened before break, got %d", opened)
	}

	// Check file1.txt stream was closed automatically
	if items[1].stream == nil || !items[1].stream.closed.Load() {
		t.Fatalf("expected first opened file stream to be closed after early break")
	}
	// Check subsequent files were never opened
	if items[2].stream != nil {
		t.Fatalf("expected file2.txt not to be opened")
	}

	// 2. Break mid-iteration: open 2 files, close them, break on second
	opened = 0
	for _, f := range fsys.OpenSeq("*.txt") {
		opened++
		if opened == 2 {
			break // this yields false, so OpenSeq will close this second file
		}
		_ = f.Close() // caller closes the first file
	}
	if opened != 2 {
		t.Fatalf("expected opened 2, got %d", opened)
	}
	if items[2].stream == nil || !items[2].stream.closed.Load() {
		t.Fatalf("expected second file stream to be closed")
	}
	if items[4].stream != nil {
		t.Fatalf("expected file4.txt not to be opened")
	}
}

func BenchmarkEntriesSeq_Traverse(b *testing.B) {
	fsys, _ := setupMockVFS()
	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		for e := range fsys.EntriesSeq() {
			_ = e
		}
	}
}
