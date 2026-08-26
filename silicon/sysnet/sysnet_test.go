// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sysnet_test

import (
	"net"
	"testing"

	"github.com/lemon4ksan/foundation/testkit/assert"

	"github.com/lemon4ksan/foundation/silicon/sysnet"
)

func TestTuneSocketConn(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)

	defer ln.Close()

	go func() {
		conn, errAccept := ln.Accept()
		if errAccept == nil {
			sysnet.TuneSocketConn(conn)
			conn.Close()
		}
	}()

	client, errDial := net.Dial("tcp", ln.Addr().String())
	assert.NoError(t, errDial)

	sysnet.TuneSocketConn(client)
	client.Close()
}

func TestBatchUDPConn_GSO(t *testing.T) {
	uconn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	assert.NoError(t, err)
	defer uconn.Close()

	batch := sysnet.NewBatchUDPConn(uconn)
	assert.NotNil(t, batch)

	_ = batch.SetGSO(1200)
	_ = batch.SetGRO(true)

	bufs := [][]byte{
		[]byte("hello"),
		[]byte("world"),
	}

	n, err := batch.WriteVectorTo(bufs, uconn.LocalAddr())
	assert.NoError(t, err)
	assert.Equal(t, int64(10), n)
}
