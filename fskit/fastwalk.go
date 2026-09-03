// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fskit

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// Entry describes a filesystem entry discovered during concurrent traversal.
type Entry struct {
	Path    string
	RelPath string
	IsDir   bool
	Size    int64
	ModTime time.Time
}

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
	q.dirs = append(q.dirs, d)
	q.cond.Signal()
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

// Walk traverses targetRoot concurrently using a worker pool and emits entries to outChan.
// It closes outChan when traversal is complete.
func Walk(targetRoot string, outChan chan<- Entry) {
	numWorkers := runtime.GOMAXPROCS(0) * 2
	if numWorkers < 8 {
		numWorkers = 8
	}

	q := newQueue()
	cleanRoot := filepath.Clean(targetRoot)
	basePrefix := filepath.Dir(cleanRoot)

	q.push(cleanRoot)

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				dir, ok := q.pop()
				if !ok {
					return
				}

				entries, err := os.ReadDir(dir)
				if err == nil {
					for _, d := range entries {
						fullPath := filepath.Join(dir, d.Name())
						rel, err := filepath.Rel(basePrefix, fullPath)
						if err != nil {
							continue
						}

						if d.IsDir() {
							outChan <- Entry{
								Path:    fullPath,
								RelPath: filepath.ToSlash(rel) + "/",
								IsDir:   true,
							}
							q.push(fullPath)
						} else {
							fi, err := d.Info()
							if err == nil {
								outChan <- Entry{
									Path:    fullPath,
									RelPath: filepath.ToSlash(rel),
									IsDir:   false,
									Size:    fi.Size(),
									ModTime: fi.ModTime(),
								}
							}
						}
					}
				}
				q.done()
			}
		}()
	}

	wg.Wait()
	close(outChan)
}

// FastWalk traverses the directory tree rooted at root, calling walkFn for each file or directory.
func FastWalk(root string, walkFn func(path string, d fs.DirEntry, err error) error) error {
	ch := make(chan Entry, 2048)
	go Walk(root, ch)

	for e := range ch {
		de := &entryDirEntry{e: e}
		if err := walkFn(e.Path, de, nil); err != nil {
			return err
		}
	}
	return nil
}

type entryDirEntry struct {
	e Entry
}

func (de *entryDirEntry) Name() string               { return filepath.Base(de.e.Path) }
func (de *entryDirEntry) IsDir() bool                { return de.e.IsDir }
func (de *entryDirEntry) Type() fs.FileMode          { return de.e.Info().Mode().Type() }
func (de *entryDirEntry) Info() (fs.FileInfo, error) { return de.e.Info(), nil }

// Info returns an [fs.FileInfo] view of the Entry.
func (e *Entry) Info() fs.FileInfo {
	var mode fs.FileMode = 0o644
	if e.IsDir {
		mode = 0o755 | fs.ModeDir
	}
	return &entryFileInfo{e: e, mode: mode}
}

type entryFileInfo struct {
	e    *Entry
	mode fs.FileMode
}

func (fi *entryFileInfo) Name() string       { return filepath.Base(fi.e.Path) }
func (fi *entryFileInfo) Size() int64        { return fi.e.Size }
func (fi *entryFileInfo) Mode() fs.FileMode  { return fi.mode }
func (fi *entryFileInfo) ModTime() time.Time { return fi.e.ModTime }
func (fi *entryFileInfo) IsDir() bool        { return fi.e.IsDir }
func (fi *entryFileInfo) Sys() any           { return nil }
