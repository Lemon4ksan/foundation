// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows

package fskit

import (
	"fmt"
	"io"
	"os"
	"syscall"
	"unsafe"
)

type windowsMmap struct {
	f      *os.File
	hMap   syscall.Handle
	addr   uintptr
	data   []byte
	length int64
}

// OpenMmap maps the specified file path directly into the process virtual memory address space.
func OpenMmap(path string) (MmapFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	size := fi.Size()
	if size == 0 {
		return &windowsMmap{f: f, data: []byte{}}, nil
	}

	hMap, err := syscall.CreateFileMapping(syscall.Handle(f.Fd()), nil, syscall.PAGE_READONLY, 0, 0, nil)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("CreateFileMapping: %w", err)
	}

	addr, err := syscall.MapViewOfFile(hMap, syscall.FILE_MAP_READ, 0, 0, uintptr(size))
	if err != nil {
		_ = syscall.CloseHandle(hMap)
		_ = f.Close()
		return nil, fmt.Errorf("MapViewOfFile: %w", err)
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
	return m.data
}

func (m *windowsMmap) Len() int64 {
	return m.length
}

func (m *windowsMmap) ReadAt(p []byte, off int64) (int, error) {
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
