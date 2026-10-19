// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !windows && !linux

package sysnet

import "syscall"

func writeVectorBuffersFD(_ uintptr, buffers [][]byte) (int64, error) {
	if len(buffers) == 0 {
		return 0, nil
	}

	return 0, syscall.ENOSYS
}
