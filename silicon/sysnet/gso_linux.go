// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build linux

package sysnet

import "golang.org/x/sys/unix"

const (
	// optUDPSegment represents UDP_SEGMENT (103) socket option for UDP Generic Segmentation Offload (GSO).
	optUDPSegment = 103
	// optUDPGRO represents UDP_GRO (104) socket option for UDP Generic Receive Offload (GRO).
	optUDPGRO = 104
)

func setUDPSegmentFD(fd uintptr, segmentSize int) error {
	return unix.SetsockoptInt(int(fd), unix.SOL_UDP, optUDPSegment, segmentSize)
}

func setUDPGROFD(fd uintptr, enable bool) error {
	val := 0
	if enable {
		val = 1
	}

	return unix.SetsockoptInt(int(fd), unix.SOL_UDP, optUDPGRO, val)
}
