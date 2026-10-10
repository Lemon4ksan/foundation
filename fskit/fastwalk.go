// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fskit

import (
	"context"
	"errors"
	"io/fs"
	"iter"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Entry describes a filesystem entry discovered during concurrent traversal.
//
// Its zero value is valid.
type Entry struct {
	Path     string
	RelPath  string
	IsDir    bool
	FileMode fs.FileMode
	Size     int64
	ModTime  time.Time
}

// Name returns the base filename of the entry.
func (e *Entry) Name() string {
	if idx := strings.LastIndexAny(e.Path, "/\\"); idx >= 0 {
		return e.Path[idx+1:]
	}
	return e.Path
}

// Size returns the length in bytes for a regular file.
func (e *Entry) SizeBytes() int64 {
	return e.Size
}

// Mode returns the file mode bits.
func (e *Entry) Mode() fs.FileMode {
	if e.FileMode != 0 {
		return e.FileMode
	}
	if e.IsDir {
		return 0o755 | fs.ModeDir
	}
	return 0o644
}

// ModTime returns the modification time.
func (e *Entry) ModTimeUTC() time.Time {
	return e.ModTime
}

// Sys returns underlying data source (always nil).
func (e *Entry) Sys() any {
	return nil
}

// Info returns an [fs.FileInfo] view of the [Entry].
func (e *Entry) Info() fs.FileInfo {
	return &entryFileInfo{e: e}
}

// Type returns type bits for [fs.DirEntry].
func (e *Entry) Type() fs.FileMode {
	return e.Mode().Type()
}

type entryFileInfo struct {
	e *Entry
}

func (fi *entryFileInfo) Name() string       { return fi.e.Name() }
func (fi *entryFileInfo) Size() int64        { return fi.e.Size }
func (fi *entryFileInfo) Mode() fs.FileMode  { return fi.e.Mode() }
func (fi *entryFileInfo) ModTime() time.Time { return fi.e.ModTime }
func (fi *entryFileInfo) IsDir() bool        { return fi.e.IsDir }
func (fi *entryFileInfo) Sys() any           { return nil }

type entryDirEntry struct {
	e Entry
}

func (de *entryDirEntry) Name() string               { return de.e.Name() }
func (de *entryDirEntry) IsDir() bool                { return de.e.IsDir }
func (de *entryDirEntry) Type() fs.FileMode          { return de.e.Type() }
func (de *entryDirEntry) Info() (fs.FileInfo, error) { return de.e.Info(), nil }
func (de *entryDirEntry) Entry() Entry               { return de.e }

type queue struct {
	mu     sync.Mutex
	cond   *sync.Cond
	dirs   []string
	active int
	closed bool
}

func newQueue() *queue {
	q := &queue{}
	q.cond = sync.NewCond(&q.mu)
	return q
}

func (q *queue) push(d string) {
	q.mu.Lock()
	if !q.closed {
		q.dirs = append(q.dirs, d)
		q.cond.Signal()
	}
	q.mu.Unlock()
}

func (q *queue) pop() (string, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for len(q.dirs) == 0 && !q.closed {
		if q.active == 0 {
			q.closed = true
			q.cond.Broadcast()
			return "", false
		}
		q.cond.Wait()
	}

	if q.closed || len(q.dirs) == 0 {
		return "", false
	}

	d := q.dirs[len(q.dirs)-1]
	q.dirs = q.dirs[:len(q.dirs)-1]
	q.active++
	return d, true
}

func (q *queue) done() {
	q.mu.Lock()
	q.active--
	if q.active == 0 && len(q.dirs) == 0 {
		q.closed = true
		q.cond.Broadcast()
	}
	q.mu.Unlock()
}

func (q *queue) cancel() {
	q.mu.Lock()
	q.closed = true
	q.dirs = nil
	q.cond.Broadcast()
	q.mu.Unlock()
}

// WalkConfig holds configuration options for directory traversal.
//
// Its zero value is valid.
type WalkConfig struct {
	NumWorkers     int
	BufferSize     int
	FollowSymlinks bool
	Filter         func(path string, isDir bool) bool
	skipMap        *sync.Map
}

// WalkOption configures directory traversal behaviour.
type WalkOption func(*WalkConfig)

// WithWorkers sets the number of concurrent worker goroutines.
func WithWorkers(n int) WalkOption {
	return func(c *WalkConfig) {
		if n > 0 {
			c.NumWorkers = n
		}
	}
}

// WithBufferSize sets the buffer capacity of the entry channel.
func WithBufferSize(n int) WalkOption {
	return func(c *WalkConfig) {
		if n > 0 {
			c.BufferSize = n
		}
	}
}

// WithFollowSymlinks configures whether symbolic links should be followed.
func WithFollowSymlinks(follow bool) WalkOption {
	return func(c *WalkConfig) {
		c.FollowSymlinks = follow
	}
}

// WithFilter sets a callback to filter encountered filesystem paths.
// If the callback returns false, the file or directory is ignored and not entered.
func WithFilter(filter func(path string, isDir bool) bool) WalkOption {
	return func(c *WalkConfig) {
		c.Filter = filter
	}
}

func joinPath(dir, name string) string {
	if len(dir) == 0 {
		return name
	}
	last := dir[len(dir)-1]
	if last == '/' || last == filepath.Separator {
		return dir + name
	}
	return dir + string(filepath.Separator) + name
}

func fastRelPath(basePrefix, fullPath string) string {
	if basePrefix == "." {
		return filepath.ToSlash(fullPath)
	}
	prefixLen := len(basePrefix)
	if len(fullPath) > prefixLen && strings.HasPrefix(fullPath, basePrefix) {
		c := fullPath[prefixLen]
		if c == '/' || c == '\\' {
			return filepath.ToSlash(fullPath[prefixLen+1:])
		}
	}
	rel, err := filepath.Rel(basePrefix, fullPath)
	if err != nil {
		return filepath.ToSlash(fullPath)
	}
	return filepath.ToSlash(rel)
}

// WalkContext traverses targetRoot concurrently using a worker pool and emits entries to outChan,
// closing outChan when traversal completes or when ctx is cancelled.
// WalkContext panics if ctx is nil.
func WalkContext(ctx context.Context, targetRoot string, outChan chan<- Entry, opts ...WalkOption) {
	cfg := WalkConfig{
		NumWorkers:     max(runtime.GOMAXPROCS(0)*2, 8),
		BufferSize:     2048,
		FollowSymlinks: false,
		Filter:         nil,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	cleanRoot := filepath.Clean(targetRoot)
	basePrefix := filepath.Dir(cleanRoot)

	fi, err := os.Stat(cleanRoot)
	if err != nil {
		close(outChan)
		return
	}

	if !fi.IsDir() {
		rel := filepath.Base(cleanRoot)
		select {
		case <-ctx.Done():
		case outChan <- Entry{
			Path:     cleanRoot,
			RelPath:  filepath.ToSlash(rel),
			IsDir:    false,
			FileMode: fi.Mode(),
			Size:     fi.Size(),
			ModTime:  fi.ModTime(),
		}:
		}
		close(outChan)
		return
	}

	q := newQueue()
	q.push(cleanRoot)

	stopCh := make(chan struct{})
	var stopOnce sync.Once
	stop := func() {
		stopOnce.Do(func() {
			close(stopCh)
			q.cancel()
		})
	}

	go func() {
		select {
		case <-ctx.Done():
			stop()
		case <-stopCh:
		}
	}()

	var wg sync.WaitGroup
	for range cfg.NumWorkers {
		wg.Go(func() {
			for {
				dir, ok := q.pop()
				if !ok {
					return
				}

				if cfg.skipMap != nil {
					// Check if this exact dir, or any of its parents, are in skipMap
					// For performance, we can just check the exact dir if we assume
					// parents were already skipped, but since concurrency allows
					// parents to be marked later, let's just check the exact dir 
					// and rely on consumer filtering for the rest.
					if _, skipped := cfg.skipMap.Load(dir); skipped {
						q.done()
						continue
					}
				}

				entries, err := os.ReadDir(dir)
				if err == nil {
					for _, d := range entries {
						select {
						case <-stopCh:
							q.done()
							return
						default:
						}

						dName := d.Name()
						fullPath := joinPath(dir, dName)
						isDir := d.IsDir()

						if cfg.Filter != nil && !cfg.Filter(fullPath, isDir) {
							continue
						}

						rel := fastRelPath(basePrefix, fullPath)

						if isDir {
							entry := Entry{
								Path:     fullPath,
								RelPath:  rel + "/",
								IsDir:    true,
								FileMode: d.Type(),
							}
							select {
							case <-stopCh:
								q.done()
								return
							case outChan <- entry:
							}
							q.push(fullPath)
						} else {
							info, err := d.Info()
							if err == nil {
								entry := Entry{
									Path:     fullPath,
									RelPath:  rel,
									IsDir:    false,
									FileMode: info.Mode(),
									Size:     info.Size(),
									ModTime:  info.ModTime(),
								}
								select {
								case <-stopCh:
									q.done()
									return
								case outChan <- entry:
								}
							}
						}
					}
				}
				q.done()
			}
		})
	}

	wg.Wait()
	stop()
	close(outChan)
}

// Walk traverses targetRoot concurrently using a worker pool and emits entries to outChan,
// closing outChan when traversal completes.
func Walk(targetRoot string, outChan chan<- Entry) {
	WalkContext(context.Background(), targetRoot, outChan)
}

// FastWalk traverses the directory tree rooted at root using a worker pool, calling walkFn for each entry.
// If walkFn returns [filepath.SkipAll], it terminates traversal immediately and returns nil;
// any other non-nil error terminates traversal and is returned.
func FastWalk(root string, walkFn func(path string, d fs.DirEntry, err error) error, opts ...WalkOption) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	skipMap := &sync.Map{}
	cfg := WalkConfig{
		BufferSize: 2048,
		skipMap:    skipMap,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	ch := make(chan Entry, cfg.BufferSize)
	done := make(chan struct{})
	go func() {
		defer close(done)
		// We pass a custom option to ensure skipMap is passed to WalkContext
		optsWithSkip := append(opts, func(c *WalkConfig) {
			c.skipMap = skipMap
		})
		WalkContext(ctx, root, ch, optsWithSkip...)
	}()

	var walkErr error
	var skippedPrefixes []string
	
	for e := range ch {
		if walkErr != nil {
			continue
		}

		// Skip consumer-side yielding for children of skipped directories.
		shouldSkip := false
		for _, p := range skippedPrefixes {
			if strings.HasPrefix(e.Path, p) && (len(e.Path) == len(p) || os.IsPathSeparator(e.Path[len(p)])) {
				shouldSkip = true
				break
			}
		}
		if shouldSkip {
			continue
		}

		de := &entryDirEntry{e: e}
		if err := walkFn(e.Path, de, nil); err != nil {
			if errors.Is(err, filepath.SkipAll) {
				walkErr = nil
				cancel()
				continue
			}
			if errors.Is(err, filepath.SkipDir) {
				if e.IsDir {
					skippedPrefixes = append(skippedPrefixes, e.Path)
					skipMap.Store(e.Path, true)
				}
				continue
			}
			walkErr = err
			cancel()
		}
	}
	<-done
	return walkErr
}

// FastWalkSeq returns an iterator over the directory tree rooted at root using concurrent workers.
// Traversal ceases immediately if yield returns false.
func FastWalkSeq(root string, opts ...WalkOption) iter.Seq2[string, fs.DirEntry] {
	return func(yield func(string, fs.DirEntry) bool) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		cfg := WalkConfig{
			BufferSize: 256,
		}
		for _, opt := range opts {
			if opt != nil {
				opt(&cfg)
			}
		}

		ch := make(chan Entry, cfg.BufferSize)
		done := make(chan struct{})
		go func() {
			defer close(done)
			WalkContext(ctx, root, ch, opts...)
		}()

		for e := range ch {
			de := &entryDirEntry{e: e}
			if !yield(e.Path, de) {
				cancel()
				for range ch {
				}
				break
			}
		}
		<-done
	}
}

// WalkSeq returns an iterator yielding paths and directory entries sequentially via [filepath.WalkDir].
// Traversal ceases immediately if yield returns false.
func WalkSeq(root string) iter.Seq2[string, os.DirEntry] {
	return func(yield func(string, os.DirEntry) bool) {
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil //nolint:nilerr // Ignore file access errors to continue directory traversal
			}
			if !yield(path, d) {
				return filepath.SkipAll
			}
			return nil
		})
	}
}
