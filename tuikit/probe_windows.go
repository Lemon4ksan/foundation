// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows

package tuikit

import (
	"strings"
	"syscall"
	"unsafe"
)

var (
	modkernel32                      = syscall.NewLazyDLL("kernel32.dll")
	procGetFileInformationByHandleEx = modkernel32.NewProc("GetFileInformationByHandleEx")
)

const (
	fileNameInfoClass = 2
)

// probeFd tests whether fd is a valid console or pseudo-terminal on Windows.
func probeFd(fd uintptr) bool {
	h := syscall.Handle(fd)

	// 1. Standard Windows Console check via native console mode syscall
	var mode uint32
	err := syscall.GetConsoleMode(h, &mode)
	if err == nil {
		return true
	}

	// 2. Check for Cygwin / MSYS / Git-Bash mintty pseudo-terminal (PTY)
	// These terminals connect stdio to named pipes with names matching "msys-*-pty*" or "cygwin-*-pty*".
	return isCygwinTerminal(h)
}

func isCygwinTerminal(h syscall.Handle) bool {
	fileType, err := syscall.GetFileType(h)
	if err != nil || fileType != syscall.FILE_TYPE_PIPE {
		return false
	}

	if procGetFileInformationByHandleEx.Find() != nil {
		return false
	}

	// Allocate buffer for FILE_NAME_INFO structure:
	// DWORD FileNameLength + WCHAR FileName[1]
	var buf [1024]byte
	r1, _, _ := procGetFileInformationByHandleEx.Call(
		uintptr(h),
		uintptr(fileNameInfoClass),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
	)
	if r1 == 0 {
		return false
	}

	fileNameLen := *(*uint32)(unsafe.Pointer(&buf[0]))
	if fileNameLen == 0 || fileNameLen > uint32(len(buf)-4) {
		return false
	}

	nameChars := (*[512]uint16)(unsafe.Pointer(&buf[4]))[:fileNameLen/2]
	name := syscall.UTF16ToString(nameChars)

	// Check if pipe name corresponds to a pty master/slave device
	lower := strings.ToLower(name)
	return strings.Contains(lower, "pty") || strings.Contains(lower, "msys") || strings.Contains(lower, "cygwin")
}
