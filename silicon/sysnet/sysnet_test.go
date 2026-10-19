// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sysnet_test

import (
	"net"
	"testing"

	"github.com/lemon4ksan/foundation/silicon/sysnet"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

func TestTuneSocketConn_And_Flags(t *testing.T) {
	t.Parallel()

	// 1. Nil conn
	sysnet.TuneSocketConn(nil)
	sysnet.TuneSocketConnWithFlags(nil, 1)

	// 2. Real TCP listener & dialer
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, errAccept := ln.Accept()
		if errAccept == nil {
			sysnet.TuneSocketConn(conn)
			sysnet.TuneSocketConnWithFlags(conn, 1)
			if tcpConn, ok := conn.(*net.TCPConn); ok {
				if raw, errRaw := tcpConn.SyscallConn(); errRaw == nil {
					_ = raw.Control(func(fd uintptr) {
						sysnet.SetTCPMaxSeg(fd, 1400)
					})
				}
			}
			conn.Close()
		}
	}()

	client, errDial := net.Dial("tcp", ln.Addr().String())
	require.NoError(t, errDial)

	sysnet.TuneSocketConn(client)
	sysnet.TuneSocketConnWithFlags(client, 2)
	sysnet.SetTCPMaxSeg(0, 1400)

	// WriteVectorBuffers over TCP
	bufs := [][]byte{[]byte("GET / HTTP/1.1\r\n"), []byte("Host: localhost\r\n\r\n")}
	n, err := sysnet.WriteVectorBuffers(client, bufs)
	require.NoError(t, err)
	assert.Greater(t, n, int64(0))

	// Nil / empty buffers edge cases
	n0, err0 := sysnet.WriteVectorBuffers(nil, bufs)
	assert.NoError(t, err0)
	assert.Equal(t, int64(0), n0)

	nEmpty, errEmpty := sysnet.WriteVectorBuffers(client, nil)
	assert.NoError(t, errEmpty)
	assert.Equal(t, int64(0), nEmpty)

	client.Close()
	<-done
}

func TestBatchUDPConn_AllMethods(t *testing.T) {
	t.Parallel()

	uconn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	require.NoError(t, err)
	defer uconn.Close()

	batch := sysnet.NewBatchUDPConn(uconn)
	require.NotNil(t, batch)

	_ = batch.SetGSO(1200)
	_ = batch.SetGRO(true)
	_ = batch.SetReadBuffer(64 * 1024)
	_ = batch.SetWriteBuffer(64 * 1024)

	raw, errRaw := batch.SyscallConn()
	_ = raw
	_ = errRaw

	bufs := [][]byte{
		[]byte("hello"),
		[]byte("world"),
	}

	// WriteVectorTo with explicit destination address
	n, err := batch.WriteVectorTo(bufs, uconn.LocalAddr())
	require.NoError(t, err)
	assert.Equal(t, int64(10), n)

	// WriteVectorTo with empty slice element
	nWithEmpty, errWithEmpty := batch.WriteVectorTo([][]byte{nil, []byte("abc")}, uconn.LocalAddr())
	require.NoError(t, errWithEmpty)
	assert.Equal(t, int64(3), nWithEmpty)

	// Connected UDP socket for WriteVector (addr == nil -> writeVectorBuffersLoop)
	dialConn, errDial := net.DialUDP("udp", nil, uconn.LocalAddr().(*net.UDPAddr))
	require.NoError(t, errDial)
	defer dialConn.Close()

	dialBatch := sysnet.NewBatchUDPConn(dialConn)
	nVec, errVec := dialBatch.WriteVector(bufs)
	require.NoError(t, errVec)
	assert.Equal(t, int64(10), nVec)

	// Single buffer and leading empty buffer in WriteVector
	nSingle, errSingle := dialBatch.WriteVector([][]byte{[]byte("single")})
	require.NoError(t, errSingle)
	assert.Equal(t, int64(6), nSingle)

	nLeadEmpty, errLeadEmpty := dialBatch.WriteVector([][]byte{nil, []byte("lead")})
	require.NoError(t, errLeadEmpty)
	assert.Equal(t, int64(4), nLeadEmpty)

	nAllEmpty, errAllEmpty := dialBatch.WriteVector([][]byte{nil, {}})
	require.NoError(t, errAllEmpty)
	assert.Equal(t, int64(0), nAllEmpty)

	// >16 buffers to exercise heap-allocated WSABuf slice path
	manyBufs := make([][]byte, 20)
	for i := range manyBufs {
		manyBufs[i] = []byte("x")
	}
	nMany, errMany := dialBatch.WriteVector(manyBufs)
	require.NoError(t, errMany)
	assert.Equal(t, int64(20), nMany)

	// Nil / empty cases
	var nilBatch *sysnet.BatchUDPConn
	nNil, errNil := nilBatch.WriteVectorTo(bufs, uconn.LocalAddr())
	assert.NoError(t, errNil)
	assert.Equal(t, int64(0), nNil)

	nEmpty, errEmpty := batch.WriteVectorTo(nil, uconn.LocalAddr())
	assert.NoError(t, errEmpty)
	assert.Equal(t, int64(0), nEmpty)

	// WriteVector nil/empty
	nVecNil, errVecNil := nilBatch.WriteVector(bufs)
	assert.NoError(t, errVecNil)
	assert.Equal(t, int64(0), nVecNil)

	nVecEmpty, errVecEmpty := batch.WriteVector(nil)
	assert.NoError(t, errVecEmpty)
	assert.Equal(t, int64(0), nVecEmpty)
}

func TestRIO_BufferRegistration(t *testing.T) {
	t.Parallel()

	if !sysnet.IsRIOSupported() {
		reg, err := sysnet.RegisterBuffer(nil)
		assert.ErrorIs(t, err, sysnet.ErrRIONotSupported)
		assert.Nil(t, reg)
		return
	}

	// Empty data
	regEmpty, err := sysnet.RegisterBuffer(nil)
	assert.NoError(t, err)
	assert.Nil(t, regEmpty)

	// Buffer registration
	data := make([]byte, 4096)
	reg, err := sysnet.RegisterBuffer(data)
	if err == nil && reg != nil {
		reg.Deregister()
	}
}

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
