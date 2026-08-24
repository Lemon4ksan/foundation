// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !linux && !windows

package sysnet

func setUDPSegmentFD(fd uintptr, segmentSize int) error {
	return nil
}

func setUDPGROFD(fd uintptr, enable bool) error {
	return nil
}
