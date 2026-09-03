// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !windows

package fskit

import (
	"fmt"
	"io"
	"os"
	"syscall"
)

type unixMmap struct {
	f      *os.File
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
		return &unixMmap{f: f, data: []byte{}}, nil
	}

	data, err := syscall.Mmap(int(f.Fd()), 0, int(size), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("mmap: %w", err)
	}

	return &unixMmap{
		f:      f,
		data:   data,
		length: size,
	}, nil
}

func (m *unixMmap) Bytes() []byte {
	return m.data
}

func (m *unixMmap) Len() int64 {
	return m.length
}

func (m *unixMmap) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 || off >= m.length {
		return 0, io.EOF
	}
	n := copy(p, m.data[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

func (m *unixMmap) Close() error {
	var firstErr error
	if m.data != nil {
		if err := syscall.Munmap(m.data); err != nil && firstErr == nil {
			firstErr = err
		}
		m.data = nil
	}
	if m.f != nil {
		if err := m.f.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		m.f = nil
	}
	return firstErr
}
