// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build linux

package tuikit

import (
	"syscall"
	"unsafe"
)

const ioctlTCGETS = 0x5401

func probeFd(fd uintptr) bool {
	var termios syscall.Termios
	_, _, err := syscall.Syscall6(syscall.SYS_IOCTL, fd, uintptr(ioctlTCGETS), uintptr(unsafe.Pointer(&termios)), 0, 0, 0)
	return err == 0
}
