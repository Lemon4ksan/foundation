// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vfs

import (
	"errors"
	"io"
	"io/fs"
	"iter"
	"path"
	"sort"
	"strings"
	"time"
)

var (
	_ fs.FS         = (*FS)(nil)
	_ fs.StatFS     = (*FS)(nil)
	_ fs.ReadFileFS = (*FS)(nil)
	_ fs.ReadDirFS  = (*FS)(nil)
)

// VFSEntry represents a generic virtual file or directory within the VFS engine.
type VFSEntry interface {
	EntryName() string
	EntrySize() int64
	EntryMode() fs.FileMode
	EntryModTime() time.Time
	EntryIsDir() bool
	OpenStream() (io.ReadCloser, error)
}

// EntryProvider defines the contract for querying VFS entries.
type EntryProvider interface {
	GetEntries() []VFSEntry
	FindEntry(name string) (VFSEntry, bool)
}

// FS provides a standard [fs.FS] virtual filesystem implementation atop any archive or entry provider.
type FS struct {
	provider EntryProvider
}

// NewFS constructs a virtual filesystem from an entry provider.
func NewFS(provider EntryProvider) *FS {
	return &FS{provider: provider}
}

// EntriesSeq returns an iterator yielding all [VFSEntry] elements registered with the VFS provider.
// Traversal stops immediately if yield returns false.
//
// Concurrency & Zero-Allocation Semantics:
// Iteration executes sequentially in the calling goroutine and performs zero heap allocations.
func (v *FS) EntriesSeq() iter.Seq[VFSEntry] {
	return func(yield func(VFSEntry) bool) {
		if v == nil || v.provider == nil {
			return
		}
		for _, e := range v.provider.GetEntries() {
			if !yield(e) {
				return
			}
		}
	}
}

// OpenSeq returns an iterator yielding (path, file) pairs for non-directory files matching pattern.
// If pattern is empty or "*", all files are matched.
// The caller is responsible for closing each yielded [fs.File]. If iteration breaks early,
// the current file that caused early break is automatically closed if yield returned false.
func (v *FS) OpenSeq(pattern string) iter.Seq2[string, fs.File] {
	return func(yield func(string, fs.File) bool) {
		if v == nil || v.provider == nil {
			return
		}
		for _, e := range v.provider.GetEntries() {
			if e.EntryIsDir() {
				continue
			}
			clean := CleanPath(e.EntryName())
			if pattern != "" && pattern != "*" {
				matched, err := path.Match(pattern, clean)
				if err != nil || !matched {
					matchedBase, err2 := path.Match(pattern, path.Base(clean))
					if err2 != nil || !matchedBase {
						continue
					}
				}
			}
			f, err := v.Open(clean)
			if err != nil {
				continue
			}
			if !yield(clean, f) {
				_ = f.Close()
				return
			}
		}
	}
}

// Open opens the named virtual file conforming to [fs.FS].
func (v *FS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}

	clean := CleanPath(name)
	if clean == "" || clean == "." {
		return &vfsDir{name: ".", isRoot: true, fs: v}, nil
	}

	if entry, ok := v.provider.FindEntry(clean); ok {
		if entry.EntryIsDir() {
			return &vfsDir{name: clean, entry: entry, fs: v}, nil
		}
		rc, err := entry.OpenStream()
		if err != nil {
			return nil, &fs.PathError{Op: "open", Path: name, Err: err}
		}
		return &vfsFile{entry: entry, rc: rc}, nil
	}

	// Check for synthesized implicit directory
	if v.hasImplicitDir(clean) {
		return &vfsDir{name: clean, fs: v}, nil
	}

	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
}

// ReadFile reads and returns the full contents of the named file according to [fs.ReadFileFS].
func (v *FS) ReadFile(name string) ([]byte, error) {
	f, err := v.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

// Stat returns [fs.FileInfo] describing the named virtual file according to [fs.StatFS].
func (v *FS) Stat(name string) (fs.FileInfo, error) {
	clean := CleanPath(name)
	if clean == "" || clean == "." {
		return &vfsFileInfo{name: ".", isDir: true, mode: 0o755 | fs.ModeDir, modTime: time.Now()}, nil
	}

	if entry, ok := v.provider.FindEntry(clean); ok {
		return &vfsFileInfo{
			name:    path.Base(entry.EntryName()),
			size:    entry.EntrySize(),
			mode:    entry.EntryMode(),
			modTime: entry.EntryModTime(),
			isDir:   entry.EntryIsDir(),
		}, nil
	}

	if v.hasImplicitDir(clean) {
		return &vfsFileInfo{
			name:    path.Base(clean),
			isDir:   true,
			mode:    0o755 | fs.ModeDir,
			modTime: time.Now(),
		}, nil
	}

	return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrNotExist}
}

// ReadDir reads and returns directory entries conforming to [fs.ReadDirFS].
func (v *FS) ReadDir(name string) ([]fs.DirEntry, error) {
	f, err := v.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	rdf, ok := f.(fs.ReadDirFile)
	if !ok {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: errors.New("not a directory")}
	}
	return rdf.ReadDir(-1)
}

func (v *FS) hasImplicitDir(dirPath string) bool {
	prefix := dirPath + "/"
	for _, e := range v.provider.GetEntries() {
		if strings.HasPrefix(e.EntryName(), prefix) {
			return true
		}
	}
	return false
}

type vfsFile struct {
	entry VFSEntry
	rc    io.ReadCloser
}

func (f *vfsFile) Stat() (fs.FileInfo, error) {
	return &vfsFileInfo{
		name:    path.Base(f.entry.EntryName()),
		size:    f.entry.EntrySize(),
		mode:    f.entry.EntryMode(),
		modTime: f.entry.EntryModTime(),
		isDir:   f.entry.EntryIsDir(),
	}, nil
}

func (f *vfsFile) Read(p []byte) (int, error) { return f.rc.Read(p) }
func (f *vfsFile) Close() error               { return f.rc.Close() }

type vfsDir struct {
	name   string
	isRoot bool
	entry  VFSEntry
	fs     *FS
}

func (d *vfsDir) Stat() (fs.FileInfo, error) {
	if d.isRoot {
		return &vfsFileInfo{name: ".", isDir: true, mode: 0o755 | fs.ModeDir, modTime: time.Now()}, nil
	}
	if d.entry != nil {
		return &vfsFileInfo{
			name:    path.Base(d.entry.EntryName()),
			size:    0,
			mode:    d.entry.EntryMode(),
			modTime: d.entry.EntryModTime(),
			isDir:   true,
		}, nil
	}
	return &vfsFileInfo{name: path.Base(d.name), isDir: true, mode: 0o755 | fs.ModeDir, modTime: time.Now()}, nil
}

func (d *vfsDir) Read(p []byte) (int, error) {
	return 0, &fs.PathError{Op: "read", Path: d.name, Err: errors.New("is a directory")}
}

func (d *vfsDir) Close() error { return nil }

func (d *vfsDir) ReadDir(n int) ([]fs.DirEntry, error) {
	dirPrefix := d.name
	if dirPrefix == "." || dirPrefix == "" {
		dirPrefix = ""
	} else if !strings.HasSuffix(dirPrefix, "/") {
		dirPrefix += "/"
	}

	seen := make(map[string]bool)
	var result []fs.DirEntry

	for _, e := range d.fs.provider.GetEntries() {
		clean := CleanPath(e.EntryName())
		if !strings.HasPrefix(clean, dirPrefix) {
			continue
		}

		rel := strings.TrimPrefix(clean, dirPrefix)
		if rel == "" {
			continue
		}

		parts := strings.SplitN(rel, "/", 2)
		childName := parts[0]

		if seen[childName] {
			continue
		}
		seen[childName] = true

		isDir := len(parts) > 1 || e.EntryIsDir()
		result = append(result, &vfsDirEntry{
			name:    childName,
			isDir:   isDir,
			size:    e.EntrySize(),
			mode:    e.EntryMode(),
			modTime: e.EntryModTime(),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name() < result[j].Name()
	})

	if n <= 0 || n >= len(result) {
		return result, nil
	}
	return result[:n], nil
}

type vfsFileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	isDir   bool
}

func (fi *vfsFileInfo) Name() string       { return fi.name }
func (fi *vfsFileInfo) Size() int64        { return fi.size }
func (fi *vfsFileInfo) Mode() fs.FileMode  { return fi.mode }
func (fi *vfsFileInfo) ModTime() time.Time { return fi.modTime }
func (fi *vfsFileInfo) IsDir() bool        { return fi.isDir }
func (fi *vfsFileInfo) Sys() any           { return nil }

type vfsDirEntry struct {
	name    string
	isDir   bool
	size    int64
	mode    fs.FileMode
	modTime time.Time
}

func (de *vfsDirEntry) Name() string      { return de.name }
func (de *vfsDirEntry) IsDir() bool       { return de.isDir }
func (de *vfsDirEntry) Type() fs.FileMode { return de.mode.Type() }
func (de *vfsDirEntry) Info() (fs.FileInfo, error) {
	return &vfsFileInfo{
		name:    de.name,
		size:    de.size,
		mode:    de.mode,
		modTime: de.modTime,
		isDir:   de.isDir,
	}, nil
}
