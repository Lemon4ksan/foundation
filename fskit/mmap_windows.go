// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows

package fskit

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"syscall"
	"unsafe"
)

type windowsMmap struct {
	mu     sync.RWMutex
	f      *os.File
	hMap   syscall.Handle
	addr   uintptr
	data   []byte
	length int64
	closed bool
}

// OpenMmap maps the specified file path directly into the process virtual memory address space.
//
// It does not panic.
//
// Complexity: O(1) time.
func OpenMmap(path string) (MmapFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	fi, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	size := fi.Size()
	if size < 0 {
		_ = f.Close()
		return nil, errors.New("fskit: negative file size")
	}
	if size == 0 {
		return &windowsMmap{f: f, data: []byte{}, length: 0}, nil
	}

	hMap, err := syscall.CreateFileMapping(syscall.Handle(f.Fd()), nil, syscall.PAGE_READONLY, 0, 0, nil)
	if err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("fskit: CreateFileMapping: %w", err)
	}

	addr, err := syscall.MapViewOfFile(hMap, syscall.FILE_MAP_READ, 0, 0, uintptr(size))
	if err != nil {
		_ = syscall.CloseHandle(hMap)
		_ = f.Close()
		return nil, fmt.Errorf("fskit: MapViewOfFile: %w", err)
	}
	//nolint:govet // addr is an OS memory-mapped address outside the Go heap
	data := unsafe.Slice((*byte)(unsafe.Pointer(addr)), int(size))

	return &windowsMmap{
		f:      f,
		hMap:   hMap,
		addr:   addr,
		data:   data,
		length: size,
	}, nil
}

func (m *windowsMmap) Bytes() []byte {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.data
}

func (m *windowsMmap) Len() int64 {
	return m.length
}

func (m *windowsMmap) ReadAt(p []byte, off int64) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.closed {
		return 0, os.ErrClosed
	}
	if off < 0 || off >= m.length {
		return 0, io.EOF
	}
	n := copy(p, m.data[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

func (m *windowsMmap) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil
	}
	m.closed = true

	var firstErr error
	if m.addr != 0 {
		if err := syscall.UnmapViewOfFile(m.addr); err != nil && firstErr == nil {
			firstErr = err
		}
		m.addr = 0
	}
	if m.hMap != 0 {
		if err := syscall.CloseHandle(m.hMap); err != nil && firstErr == nil {
			firstErr = err
		}
		m.hMap = 0
	}
	if m.f != nil {
		if err := m.f.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		m.f = nil
	}
	m.data = nil
	return firstErr
}
