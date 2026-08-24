// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows

package sysnet

import "golang.org/x/sys/windows"

const (
	// udpSendMsgSize specifies UDP_SEND_MSG_SIZE socket option on Windows (IPPROTO_UDP, 2).
	udpSendMsgSize = 2
	// udpRecvMaxCoalescedSize specifies UDP_RECV_MAX_COALESCED_SIZE option on Windows (IPPROTO_UDP, 3).
	udpRecvMaxCoalescedSize = 3
)

func setUDPSegmentFD(fd uintptr, segmentSize int) error {
	return windows.SetsockoptInt(windows.Handle(fd), windows.IPPROTO_UDP, udpSendMsgSize, segmentSize)
}

func setUDPGROFD(fd uintptr, enable bool) error {
	val := 0
	if enable {
		val = 1
	}

	return windows.SetsockoptInt(windows.Handle(fd), windows.IPPROTO_UDP, udpRecvMaxCoalescedSize, val)
}
