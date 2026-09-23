// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package quic

import (
	"errors"
	"net"
	"testing"
)

type benchAddrSender struct {
	local  net.Addr
	remote net.Addr
}

func (m *benchAddrSender) onHasConnectionData()                                       {}
func (m *benchAddrSender) onHasStreamData(StreamID, *SendStream)                      {}
func (m *benchAddrSender) onHasStreamRetransmission(StreamID, *SendStream)            {}
func (m *benchAddrSender) onHasStreamControlFrame(StreamID, streamControlFrameGetter) {}
func (m *benchAddrSender) updateStreamPriority(StreamID)                              {}
func (m *benchAddrSender) onStreamCompleted(StreamID)                                 {}
func (m *benchAddrSender) LocalAddr() net.Addr                                        { return m.local }

func (m *benchAddrSender) RemoteAddr() net.Addr { return m.remote }

func BenchmarkSupportedVersionsSeq(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		for v := range SupportedVersionsSeq() {
			_ = v
		}
	}
}

func BenchmarkStreamNetConn_LocalRemoteAddr(b *testing.B) {
	sender := &benchAddrSender{
		local:  &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 1234},
		remote: &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 5678},
	}
	str := &Stream{sender: sender}
	b.ReportAllocs()
	for b.Loop() {
		_ = str.LocalAddr()
		_ = str.RemoteAddr()
	}
}

func BenchmarkErrorsIs_StreamReset(b *testing.B) {
	err := &StreamError{StreamID: 1, ErrorCode: 42, Remote: true}
	b.ReportAllocs()
	for b.Loop() {
		if !errors.Is(err, ErrStreamReset) {
			b.Fatal("expected match")
		}
	}
}

func BenchmarkErrorsIs_StreamLimitReached(b *testing.B) {
	err := StreamLimitReachedError{}
	b.ReportAllocs()
	for b.Loop() {
		if !errors.Is(err, ErrStreamLimitReached) {
			b.Fatal("expected match")
		}
	}
}

func BenchmarkErrorsIs_Timeout(b *testing.B) {
	err := &IdleTimeoutError{}
	b.ReportAllocs()
	for b.Loop() {
		if !errors.Is(err, ErrTimeout) {
			b.Fatal("expected match")
		}
	}
}

func BenchmarkErrorsIs_ConnectionClosed(b *testing.B) {
	err := &ApplicationError{ErrorCode: 0}
	b.ReportAllocs()
	for b.Loop() {
		if !errors.Is(err, ErrConnectionClosed) {
			b.Fatal("expected match")
		}
	}
}
