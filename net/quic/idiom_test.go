// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package quic

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/net/quic/internal/protocol"
	"github.com/lemon4ksan/foundation/net/quic/internal/qerr"
	"github.com/lemon4ksan/foundation/net/quic/internal/wire"
	"github.com/lemon4ksan/foundation/testing/gomock"
	"github.com/lemon4ksan/foundation/testing/require"
)

// Compile-time interface conformance assertions.
var (
	_ net.Conn           = (*Stream)(nil)
	_ io.ReadWriteCloser = (*Stream)(nil)
	_ io.ReadCloser      = (*ReceiveStream)(nil)
	_ io.WriteCloser     = (*SendStream)(nil)
	_ io.Closer          = (*Conn)(nil)
	_ net.Listener       = (*streamListener)(nil)
	_ net.Listener       = (*quicNetListener)(nil)
)

type dummyNoAddrSender struct{}

func (d *dummyNoAddrSender) onHasConnectionData()                                       {}
func (d *dummyNoAddrSender) onHasStreamData(StreamID, *SendStream)                      {}
func (d *dummyNoAddrSender) onHasStreamRetransmission(StreamID, *SendStream)            {}
func (d *dummyNoAddrSender) onHasStreamControlFrame(StreamID, streamControlFrameGetter) {}
func (d *dummyNoAddrSender) updateStreamPriority(StreamID)                              {}
func (d *dummyNoAddrSender) onStreamCompleted(StreamID)                                 {}

func TestStream_NetConnInterface(t *testing.T) {
	localAddr := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 1234}
	remoteAddr := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 5678}

	addrSender := &benchAddrSender{local: localAddr, remote: remoteAddr}
	fc := newTestStreamFlowControllerWithSendWindow(1, protocol.MaxByteCount)
	str := newStream(context.Background(), 1, addrSender, fc, false)

	// Addresses
	require.Equal(t, localAddr, str.LocalAddr())
	require.Equal(t, remoteAddr, str.RemoteAddr())

	// Without address methods on sender
	noAddrStr := newStream(context.Background(), 2, &dummyNoAddrSender{}, fc, false)
	require.Nil(t, noAddrStr.LocalAddr())
	require.Nil(t, noAddrStr.RemoteAddr())

	// FinalSize before completion
	fs, ok := str.FinalSize()
	require.False(t, ok)
	require.Zero(t, fs)

	// CloseWrite and CloseRead
	require.NoError(t, str.CloseWrite())
	require.NoError(t, str.CloseRead())

	// Deadlines
	require.NoError(t, str.SetDeadline(time.Now().Add(time.Hour)))
	require.NoError(t, str.SetReadDeadline(time.Now().Add(time.Hour)))
	require.NoError(t, str.SetWriteDeadline(time.Now().Add(time.Hour)))
}

func TestReceiveStream_Enhancements(t *testing.T) {
	fc := newTestStreamFlowControllerWithSendWindow(3, protocol.MaxByteCount)
	rs := newReceiveStream(3, &dummyNoAddrSender{}, fc)

	// Context should be non-nil and not yet canceled
	ctx := rs.Context()
	require.NotNil(t, ctx)
	require.NoError(t, ctx.Err())

	// FinalSize before FIN
	fs, ok := rs.FinalSize()
	require.False(t, ok)
	require.Zero(t, fs)

	// Close cancels read with code 0
	require.NoError(t, rs.Close())

	// StreamID
	require.Equal(t, protocol.StreamID(3), rs.StreamID())
}

func TestSendStream_Enhancements(t *testing.T) {
	fc := newTestStreamFlowControllerWithSendWindow(5, protocol.MaxByteCount)
	ss := newSendStream(context.Background(), 5, &dummyNoAddrSender{}, fc, false)

	// CloseWrite is an alias for Close
	require.NoError(t, ss.CloseWrite())
}

func TestConn_CloserAndHandshake(t *testing.T) {
	tc := newClientTestConnection(t, nil, nil, false)
	conn := tc.conn
	tc.connRunner.EXPECT().Remove(gomock.Any()).AnyTimes()

	// Handshake with canceled context
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	err := conn.Handshake(canceledCtx)
	require.ErrorIs(t, err, context.Canceled)

	// Handshake when handshake completes
	tc.conn.handshakeCompleteChan = make(chan struct{})
	close(tc.conn.handshakeCompleteChan)
	require.NoError(t, conn.Handshake(context.Background()))

	// Close() error via io.Closer
	require.NoError(t, conn.Close())
}

func TestConn_Iterators(t *testing.T) {
	tc := newClientTestConnection(t, nil, nil, false)
	conn := tc.conn

	// Active stream iterators
	count := 0
	for range conn.Streams() {
		count++
	}
	require.Zero(t, count)

	for range conn.SendStreams() {
		count++
	}
	require.Zero(t, count)

	for range conn.ReceiveStreams() {
		count++
	}
	require.Zero(t, count)

	// Event-loop push iterators with canceled context
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	for str, err := range conn.IncomingStreams(canceledCtx) {
		require.Nil(t, str)
		require.ErrorIs(t, err, context.Canceled)
	}

	for str, err := range conn.IncomingUniStreams(canceledCtx) {
		require.Nil(t, str)
		require.ErrorIs(t, err, context.Canceled)
	}

	for data, err := range conn.Datagrams(canceledCtx) {
		require.Nil(t, data)
		require.Error(t, err)
	}
}

func TestConn_ContextAliases(t *testing.T) {
	tc := newClientTestConnection(t, nil, nil, false)
	conn := tc.conn

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	// OpenStreamContext blocks on canceled context
	_, err := conn.OpenStreamContext(canceledCtx)
	require.ErrorIs(t, err, context.Canceled)

	// OpenUniStreamContext blocks on canceled context
	_, err = conn.OpenUniStreamContext(canceledCtx)
	require.ErrorIs(t, err, context.Canceled)
}

func TestConn_StreamListener(t *testing.T) {
	tc := newClientTestConnection(t, nil, nil, false)
	conn := tc.conn

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ln := conn.StreamListener(ctx)
	require.NotNil(t, ln)
	require.Equal(t, conn.LocalAddr(), ln.Addr())

	// Close listener cancels its internal context
	require.NoError(t, ln.Close())

	_, err := ln.Accept()
	require.ErrorIs(t, err, context.Canceled)
}

func TestListener_IncomingAndNetListener(t *testing.T) {
	udpConn := newUDPConnLocalhost(t)
	tr := &Transport{Conn: udpConn}
	require.NoError(t, tr.init(false))
	defer tr.Close()

	tlsConf := &tls.Config{ServerName: "localhost"}
	ln, err := tr.Listen(tlsConf)
	require.NoError(t, err)
	defer ln.Close()

	// Incoming iterator can be constructed
	incomingSeq := ln.Incoming(context.Background())
	require.NotNil(t, incomingSeq)

	// NetListener
	netLn := ln.NetListener(context.Background())
	require.NotNil(t, netLn)
	require.Equal(t, ln.Addr(), netLn.Addr())

	// Close netListener
	require.NoError(t, netLn.Close())
	_, err = netLn.Accept()
	require.Error(t, err)
}

func TestInterface_AliasesAndIter(t *testing.T) {
	// Type aliases
	var (
		_ *Connection      = (*Conn)(nil)
		_ *Session         = (*Conn)(nil)
		_ *EarlyConnection = (*Conn)(nil)
	)

	// SupportedVersionsSeq yields RFC 9000 (Version1) and RFC 9369 (Version2)
	var versions []Version
	for v := range SupportedVersionsSeq() {
		versions = append(versions, v)
	}
	require.Equal(t, []Version{Version1, Version2}, versions)

	// Early break in SupportedVersionsSeq
	count := 0
	for range SupportedVersionsSeq() {
		count++
		break
	}
	require.Equal(t, 1, count)
}

func TestErrors_WrappingAndIs(t *testing.T) {
	// Sentinels exist and are non-nil
	require.NotNil(t, ErrConnectionClosed)
	require.NotNil(t, ErrStreamReset)
	require.NotNil(t, ErrTimeout)
	require.NotNil(t, ErrStreamLimitReached)
	require.NotNil(t, ErrDeadline)

	// StreamError unwrapping and Is
	sErr := &StreamError{StreamID: 10, ErrorCode: 42, Remote: true}
	require.True(t, errors.Is(sErr, ErrStreamReset))
	require.True(t, errors.Is(sErr, net.ErrClosed))
	require.True(t, errors.Is(sErr, &StreamError{}))
	require.True(t, errors.Is(sErr, &StreamError{StreamID: 10, ErrorCode: 42, Remote: true}))
	require.False(t, errors.Is(sErr, &StreamError{StreamID: 99, ErrorCode: 42, Remote: true}))

	unwrapped := sErr.Unwrap()
	require.Len(t, unwrapped, 2)
	require.Equal(t, ErrStreamReset, unwrapped[0])
	require.Equal(t, net.ErrClosed, unwrapped[1])

	var targetSErr *StreamError
	require.True(t, errors.As(sErr, &targetSErr))
	require.Equal(t, protocol.StreamID(10), targetSErr.StreamID)
	require.Equal(t, qerr.StreamErrorCode(42), targetSErr.ErrorCode)
	require.True(t, targetSErr.Remote)

	// StreamLimitReachedError Is
	limitErr := StreamLimitReachedError{}
	require.True(t, errors.Is(limitErr, ErrStreamLimitReached))
	require.True(t, errors.Is(&limitErr, ErrStreamLimitReached))
	require.True(t, errors.Is(limitErr, StreamLimitReachedError{}))
	require.True(t, errors.Is(limitErr, &StreamLimitReachedError{}))
	require.True(t, errors.Is(ErrStreamLimitReached, StreamLimitReachedError{}))
	require.True(t, errors.Is(ErrStreamLimitReached, &StreamLimitReachedError{}))

	// ApplicationError
	appErr := &ApplicationError{ErrorCode: 0x42, ErrorMessage: "app error"}
	require.True(t, errors.Is(appErr, ErrConnectionClosed))
	require.True(t, errors.Is(appErr, net.ErrClosed))

	// TransportError
	transErr := &TransportError{ErrorCode: FlowControlError}
	require.True(t, errors.Is(transErr, ErrConnectionClosed))
	require.True(t, errors.Is(transErr, net.ErrClosed))

	// IdleTimeoutError
	idleErr := &IdleTimeoutError{}
	require.True(t, errors.Is(idleErr, ErrTimeout))
	require.True(t, errors.Is(idleErr, net.ErrClosed))

	// HandshakeTimeoutError
	hsErr := &HandshakeTimeoutError{}
	require.True(t, errors.Is(hsErr, ErrTimeout))
	require.True(t, errors.Is(hsErr, net.ErrClosed))
}

type dummyStream struct {
	id protocol.StreamID
}

func (d *dummyStream) updateSendWindow(protocol.ByteCount) {}
func (d *dummyStream) enableResetStreamAt()                {}
func (d *dummyStream) closeForShutdown(error)              {}

func TestStreamsMatrix_Iterators(t *testing.T) {
	m := newStreamsMatrix[*dummyStream]()

	s1 := &dummyStream{id: 1}
	s5 := &dummyStream{id: 5}
	s9 := &dummyStream{id: 9}

	m.set(1, incomingStreamEntry[*dummyStream]{stream: s1})
	m.set(5, incomingStreamEntry[*dummyStream]{stream: s5})
	m.set(9, incomingStreamEntry[*dummyStream]{stream: s9})

	// All() iterator
	var allEntries []incomingStreamEntry[*dummyStream]
	for entry := range m.All() {
		allEntries = append(allEntries, entry)
	}
	require.Len(t, allEntries, 3)

	// Streams() iterator
	var active []*dummyStream
	for str := range m.Streams() {
		active = append(active, str)
	}
	require.Len(t, active, 3)

	// Break early in Streams()
	count := 0
	for range m.Streams() {
		count++
		break
	}
	require.Equal(t, 1, count)

	// Delete an entry and verify Streams skips it
	m.del(5)
	var remaining []*dummyStream
	for str := range m.Streams() {
		remaining = append(remaining, str)
	}
	require.Len(t, remaining, 2)
	require.Contains(t, remaining, s1)
	require.Contains(t, remaining, s9)
}

func TestOutgoingStreamsMap_Iterators(t *testing.T) {
	m := newOutgoingStreamsMap[*dummyStream](
		protocol.StreamTypeUni,
		func(id protocol.StreamID) *dummyStream { return &dummyStream{id: id} },
		func(wire.Frame) {},
		protocol.PerspectiveClient,
	)

	m.streams[1] = &dummyStream{id: 1}
	m.streams[5] = &dummyStream{id: 5}

	var collected []protocol.StreamID
	for str := range m.Streams() {
		collected = append(collected, str.id)
	}
	require.Len(t, collected, 2)

	// Early break
	count := 0
	for range m.Streams() {
		count++
		break
	}
	require.Equal(t, 1, count)
}

func TestTransport_PureGoCoverage(t *testing.T) {
	udpConn := newUDPConnLocalhost(t)
	tr := &Transport{Conn: udpConn}
	require.NoError(t, tr.init(false))
	defer tr.Close()

	// 1. Reset tokens
	token := protocol.StatelessResetToken{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	mockHandler := &mockPacketHandler{}
	(*packetHandlerMap)(tr).AddResetToken(token, mockHandler)
	require.NotNil(t, tr.resetTokens[token])

	(*packetHandlerMap)(tr).RemoveResetToken(token)
	require.Nil(t, tr.resetTokens[token])

	// 2. AddWithConnID
	c1 := protocol.ParseConnectionID([]byte{1, 2, 3, 4})
	c2 := protocol.ParseConnectionID([]byte{5, 6, 7, 8})
	require.True(t, (*packetHandlerMap)(tr).AddWithConnID(c1, c2, mockHandler))
	require.False(t, (*packetHandlerMap)(tr).AddWithConnID(c1, c2, mockHandler))

	// 3. ReplaceWithClosed
	closePkt := []byte("conn-close")
	(*packetHandlerMap)(tr).ReplaceWithClosed([]protocol.ConnectionID{c1}, closePkt, time.Hour)
	handler, ok := (*packetHandlerMap)(tr).Get(c1)
	require.True(t, ok)
	require.NotNil(t, handler)

	// Remote closed (connClosePacket is nil)
	(*packetHandlerMap)(tr).ReplaceWithClosed([]protocol.ConnectionID{c2}, nil, time.Hour)
	handler2, ok := (*packetHandlerMap)(tr).Get(c2)
	require.True(t, ok)
	require.NotNil(t, handler2)

	// 4. Non-QUIC packet handling
	p := receivedPacket{
		remoteAddr: &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 9999},
		data:       []byte("non-quic-data"),
	}

	// Reading disabled -> dropped
	tr.readingNonQUICPackets.Store(false)
	tr.handleNonQUICPacket(p)

	// Reading enabled
	tr.nonQUICPackets = make(chan receivedPacket, 10)
	tr.readingNonQUICPackets.Store(true)
	tr.handleNonQUICPacket(p)

	ctx := context.Background()
	buf := make([]byte, 64)
	n, addr, err := tr.ReadNonQUICPacket(ctx, buf)
	require.NoError(t, err)
	require.Equal(t, len("non-quic-data"), n)
	require.Equal(t, "non-quic-data", string(buf[:n]))
	require.Equal(t, p.remoteAddr, addr)

	// ReadNonQUICPacket with canceled context
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err = tr.ReadNonQUICPacket(canceledCtx, buf)
	require.ErrorIs(t, err, context.Canceled)

	// 5. ListenAddr invalid address error
	_, err = ListenAddr("invalid:::port", nil)
	require.Error(t, err)
}
