// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sysnet_test

import (
	"net"
	"testing"

	"github.com/lemon4ksan/foundation/silicon/sysnet"
)

func BenchmarkTuneSocketConn_Nil(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		sysnet.TuneSocketConn(nil)
	}
}

func BenchmarkTuneSocketConnWithFlags_Nil(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		sysnet.TuneSocketConnWithFlags(nil, 1)
	}
}

func BenchmarkWriteVectorBuffers_Nil(b *testing.B) {
	buffers := [][]byte{[]byte("header"), []byte("payload")}
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, _ = sysnet.WriteVectorBuffers(nil, buffers)
	}
}

func BenchmarkWriteVectorBuffers_Empty(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, _ = sysnet.WriteVectorBuffers(nil, nil)
	}
}

func BenchmarkBatchUDPConn_Nil(b *testing.B) {
	var nilBatch *sysnet.BatchUDPConn
	buffers := [][]byte{[]byte("msg1"), []byte("msg2")}
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, _ = nilBatch.WriteVector(buffers)
	}
}

func BenchmarkBatchUDPConn_SetBuffers(b *testing.B) {
	uconn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		b.Fatal(err)
	}
	defer uconn.Close()

	batch := sysnet.NewBatchUDPConn(uconn)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = batch.SetReadBuffer(64 * 1024)
		_ = batch.SetWriteBuffer(64 * 1024)
	}
}

func BenchmarkIsRIOSupported(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = sysnet.IsRIOSupported()
	}
}
