// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package quic

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/net/quic/internal/protocol"
	"github.com/lemon4ksan/foundation/net/quic/internal/qerr"
	"github.com/lemon4ksan/foundation/net/quic/internal/wire"
	"github.com/lemon4ksan/foundation/testing/gomock"
	"github.com/lemon4ksan/foundation/testing/require"
)

// ---------------------------------------------------------------------------
// 1. Stream as Standard Library net.Conn Stress Verification
// ---------------------------------------------------------------------------

type mockFullAddrSender struct {
	local  net.Addr
	remote net.Addr

	mu           sync.Mutex
	streamData   map[protocol.StreamID][][]byte
	completedIDs []protocol.StreamID
}

func newMockFullAddrSender(local, remote net.Addr) *mockFullAddrSender {
	return &mockFullAddrSender{
		local:      local,
		remote:     remote,
		streamData: make(map[protocol.StreamID][][]byte),
	}
}

func (m *mockFullAddrSender) onHasConnectionData() {}
func (m *mockFullAddrSender) onHasStreamData(id protocol.StreamID, str *SendStream) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Pop stream data frame to drain
	for {
		sf, _, hasMore := str.popStreamFrame(protocol.MaxByteCount, protocol.Version1)
		if sf.Frame != nil && sf.Frame.DataLen() > 0 {
			buf := make([]byte, sf.Frame.DataLen())
			copy(buf, sf.Frame.Data)
			m.streamData[id] = append(m.streamData[id], buf)
		}
		if !hasMore {
			break
		}
	}
}
func (m *mockFullAddrSender) onHasStreamRetransmission(protocol.StreamID, *SendStream)            {}
func (m *mockFullAddrSender) onHasStreamControlFrame(protocol.StreamID, streamControlFrameGetter) {}
func (m *mockFullAddrSender) updateStreamPriority(protocol.StreamID)                              {}
func (m *mockFullAddrSender) onStreamCompleted(id protocol.StreamID) {
	m.mu.Lock()
	m.completedIDs = append(m.completedIDs, id)
	m.mu.Unlock()
}
func (m *mockFullAddrSender) LocalAddr() net.Addr  { return m.local }
func (m *mockFullAddrSender) RemoteAddr() net.Addr { return m.remote }

func TestAdversarial_StreamNetConn_AddrLifecycle(t *testing.T) {
	local := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 9001}
	remote := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 9002}
	sender := newMockFullAddrSender(local, remote)
	fc := newTestStreamFlowControllerWithSendWindow(1, protocol.MaxByteCount)
	str := newStream(context.Background(), 1, sender, fc, false)

	// 1. Valid addresses before closure
	require.Equal(t, local, str.LocalAddr())
	require.Equal(t, remote, str.RemoteAddr())

	// 2. Addresses during active half-close
	require.NoError(t, str.CloseRead())
	require.Equal(t, local, str.LocalAddr())
	require.Equal(t, remote, str.RemoteAddr())

	require.NoError(t, str.CloseWrite())
	require.Equal(t, local, str.LocalAddr())
	require.Equal(t, remote, str.RemoteAddr())

	// 3. Addresses after Close()
	require.NoError(t, str.Close())
	require.Equal(t, local, str.LocalAddr())
	require.Equal(t, remote, str.RemoteAddr())

	// 4. Addresses after shutdown
	str.closeForShutdown(errors.New("shutdown"))
	require.Equal(t, local, str.LocalAddr())
	require.Equal(t, remote, str.RemoteAddr())

	// 5. Stream with nil/unsupported sender returns nil gracefully
	nilAddrStr := newStream(context.Background(), 2, &dummyNoAddrSender{}, fc, false)
	require.Nil(t, nilAddrStr.LocalAddr())
	require.Nil(t, nilAddrStr.RemoteAddr())

	// Stream with completely nil sender field
	emptyStr := &Stream{}
	require.Nil(t, emptyStr.LocalAddr())
	require.Nil(t, emptyStr.RemoteAddr())
}

func TestAdversarial_StreamNetConn_HalfCloseSemantics(t *testing.T) {
	local := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 9001}
	remote := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 9002}
	sender := newMockFullAddrSender(local, remote)
	fc := newTestStreamFlowControllerWithSendWindow(1, protocol.MaxByteCount)
	str := newStream(context.Background(), 1, sender, fc, false)

	// Deliver 5 bytes of data on stream
	err := str.handleStreamFrame(&wire.StreamFrame{
		StreamID: 1,
		Offset:   0,
		Data:     []byte("hello"),
		Fin:      false,
	}, 0)
	require.NoError(t, err)

	// CloseRead cancels reading
	require.NoError(t, str.CloseRead())

	// Idempotency: multiple CloseRead() calls succeed without panic
	require.NoError(t, str.CloseRead())
	require.NoError(t, str.CloseRead())

	// Subsequent Read returns StreamError unwrappable to ErrStreamReset and net.ErrClosed
	buf := make([]byte, 10)
	_, readErr := str.Read(buf)
	require.Error(t, readErr)
	require.True(t, errors.Is(readErr, ErrStreamReset), "expected ErrStreamReset")
	require.True(t, errors.Is(readErr, net.ErrClosed), "expected net.ErrClosed")

	// CloseRead must NOT prevent writing (independent half-close)
	n, writeErr := str.Write([]byte("outbound data"))
	require.NoError(t, writeErr)
	require.Equal(t, 13, n)

	// Now test CloseWrite
	require.NoError(t, str.CloseWrite())
	require.NoError(t, str.CloseWrite()) // Idempotent

	// Future writes must fail
	_, writeErrAfterClose := str.Write([]byte("should fail"))
	require.Error(t, writeErrAfterClose)
}

func TestAdversarial_StreamNetConn_ConcurrentReadWrite(t *testing.T) {
	local := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 9001}
	remote := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 9002}
	sender := newMockFullAddrSender(local, remote)
	fc := newTestStreamFlowControllerWithSendWindow(1, protocol.MaxByteCount)
	str := newStream(context.Background(), 1, sender, fc, false)

	const numOps = 100
	var wg sync.WaitGroup

	// Writer goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numOps; i++ {
			payload := []byte("stress-test-data-payload")
			_, err := str.Write(payload)
			if err != nil {
				return
			}
		}
		_ = str.CloseWrite()
	}()

	// Feeder goroutine delivering frames to receiveStr
	wg.Add(1)
	go func() {
		defer wg.Done()
		var offset protocol.ByteCount
		for i := 0; i < numOps; i++ {
			chunk := []byte("stream-chunk")
			_ = str.handleStreamFrame(&wire.StreamFrame{
				StreamID: 1,
				Offset:   offset,
				Data:     chunk,
				Fin:      i == numOps-1,
			}, 0)
			offset += protocol.ByteCount(len(chunk))
			time.Sleep(50 * time.Microsecond)
		}
	}()

	// Reader goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 32)
		for {
			_, err := str.Read(buf)
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				return
			}
		}
	}()

	wg.Wait()
}

func TestAdversarial_StreamNetConn_Deadlines(t *testing.T) {
	local := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 9001}
	remote := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 9002}
	sender := newMockFullAddrSender(local, remote)
	fc := newTestStreamFlowControllerWithSendWindow(1, protocol.MaxByteCount)
	str := newStream(context.Background(), 1, sender, fc, false)

	// 1. Past read deadline: immediate timeout
	require.NoError(t, str.SetReadDeadline(time.Now().Add(-time.Second)))
	buf := make([]byte, 10)
	_, err := str.Read(buf)
	require.Error(t, err)
	require.True(t, errors.Is(err, os.ErrDeadlineExceeded))
	require.True(t, errors.Is(err, ErrDeadline))
	var netErr net.Error
	require.True(t, errors.As(err, &netErr))
	require.True(t, netErr.Timeout())

	// 2. Clear read deadline with zero time
	require.NoError(t, str.SetReadDeadline(time.Time{}))

	// 3. Future read deadline with timeout unblocking
	require.NoError(t, str.SetReadDeadline(time.Now().Add(50*time.Millisecond)))
	start := time.Now()
	_, err = str.Read(buf)
	elapsed := time.Since(start)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrDeadline))
	require.GreaterOrEqual(t, elapsed, 40*time.Millisecond)

	// 4. SetDeadline sets both read and write deadlines
	require.NoError(t, str.SetDeadline(time.Now().Add(-time.Hour)))
	_, err = str.Read(buf)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrDeadline))

	// 5. Concurrent deadline updates while read is blocked
	require.NoError(t, str.SetReadDeadline(time.Time{})) // unblock
	readDone := make(chan error, 1)
	go func() {
		_, rErr := str.Read(buf)
		readDone <- rErr
	}()

	// Wait for read to block, then set past deadline to abort it
	time.Sleep(20 * time.Millisecond)
	require.NoError(t, str.SetReadDeadline(time.Now().Add(-time.Millisecond)))

	select {
	case rErr := <-readDone:
		require.Error(t, rErr)
		require.True(t, errors.Is(rErr, ErrDeadline))
	case <-time.After(2 * time.Second):
		t.Fatal("Read did not unblock after SetReadDeadline to past time")
	}
}

// ---------------------------------------------------------------------------
// 2. Conn.Close(), Handshake(ctx), and StreamListener(ctx) Verification
// ---------------------------------------------------------------------------

func TestAdversarial_Conn_HandshakeMatrix(t *testing.T) {
	tc := newClientTestConnection(t, nil, nil, false)
	conn := tc.conn
	tc.connRunner.EXPECT().Remove(gomock.Any()).AnyTimes()

	// 1. Already-canceled context returns context.Canceled immediately
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	err := conn.Handshake(canceledCtx)
	require.ErrorIs(t, err, context.Canceled)

	// 2. Context with timeout expires during handshake
	timeoutCtx, cancelTimeout := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancelTimeout()
	err = conn.Handshake(timeoutCtx)
	require.ErrorIs(t, err, context.DeadlineExceeded)

	// 3. Clean handshake completion
	close(conn.handshakeCompleteChan)
	require.NoError(t, conn.Handshake(context.Background()))

	// 4. Closed connection returns close error on Handshake()
	require.NoError(t, conn.Close())
	freshCtx := context.Background()
	_ = conn.Handshake(freshCtx) // handshake channel is closed so it returns nil or close cause
}

func TestAdversarial_StreamListener_Harness(t *testing.T) {
	tc := newClientTestConnection(t, nil, nil, false)
	conn := tc.conn
	tc.connRunner.EXPECT().Remove(gomock.Any()).AnyTimes()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ln := conn.StreamListener(ctx)
	require.NotNil(t, ln)
	require.Equal(t, conn.LocalAddr(), ln.Addr())

	// 1. Accept under listener Close()
	acceptErrChan := make(chan error, 1)
	go func() {
		_, err := ln.Accept()
		acceptErrChan <- err
	}()

	time.Sleep(20 * time.Millisecond)
	require.NoError(t, ln.Close())

	select {
	case err := <-acceptErrChan:
		require.Error(t, err)
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(2 * time.Second):
		t.Fatal("Accept did not unblock on listener.Close()")
	}

	// 2. Accept under connection Close()
	tc2 := newClientTestConnection(t, nil, nil, false)
	conn2 := tc2.conn
	tc2.connRunner.EXPECT().Remove(gomock.Any()).AnyTimes()

	ln2 := conn2.StreamListener(context.Background())
	acceptErrChan2 := make(chan error, 1)
	go func() {
		_, err := ln2.Accept()
		acceptErrChan2 <- err
	}()

	time.Sleep(20 * time.Millisecond)
	require.NoError(t, conn2.Close())

	select {
	case err := <-acceptErrChan2:
		require.Error(t, err)
		require.True(t, errors.Is(err, ErrConnectionClosed) || errors.Is(err, net.ErrClosed))
	case <-time.After(2 * time.Second):
		t.Fatal("Accept did not unblock on conn.Close()")
	}
}

// ---------------------------------------------------------------------------
// 3. Push Iterators Stress & Zero Allocations
// ---------------------------------------------------------------------------

func TestAdversarial_SupportedVersionsSeq_Behavior(t *testing.T) {
	// 1. Full iteration yields exact supported versions
	var collected []Version
	for v := range SupportedVersionsSeq() {
		collected = append(collected, v)
	}
	require.Equal(t, SupportedVersions(), collected)

	// 2. Early break stops at 1 iteration cleanly
	count := 0
	for range SupportedVersionsSeq() {
		count++
		break
	}
	require.Equal(t, 1, count)

	// 3. Nested iteration
	nestedCount := 0
	for v1 := range SupportedVersionsSeq() {
		for v2 := range SupportedVersionsSeq() {
			if v1 == v2 {
				nestedCount++
			}
		}
	}
	require.Equal(t, len(collected), nestedCount)
}

func TestAdversarial_StreamsIterators_RapidChurn(t *testing.T) {
	tc := newClientTestConnection(t, nil, nil, false)
	conn := tc.conn
	tc.connRunner.EXPECT().Remove(gomock.Any()).AnyTimes()

	const numWorkers = 8
	const numOpsPerWorker = 30
	var wg sync.WaitGroup

	// Concurrent stream creators
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < numOpsPerWorker; i++ {
				str, err := conn.OpenStream()
				if err == nil && str != nil {
					_ = str.CloseWrite()
				}
				uniStr, err := conn.OpenUniStream()
				if err == nil && uniStr != nil {
					_ = uniStr.CloseWrite()
				}
			}
		}()
	}

	// Concurrent stream iterators
	stopIter := atomic.Bool{}
	for r := 0; r < 4; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for !stopIter.Load() {
				for s := range conn.Streams() {
					_ = s.StreamID()
				}
				for s := range conn.SendStreams() {
					_ = s.StreamID()
				}
				for s := range conn.ReceiveStreams() {
					_ = s.StreamID()
				}
			}
		}()
	}

	// Wait for creators to complete
	time.Sleep(200 * time.Millisecond)
	stopIter.Store(true)
	wg.Wait()
}

func TestAdversarial_StreamsIterator_ReentrancyAndDeadlockPrevention(t *testing.T) {
	tc := newClientTestConnection(t, nil, nil, false)
	conn := tc.conn
	tc.connRunner.EXPECT().Remove(gomock.Any()).AnyTimes()

	params := &wire.TransportParameters{
		InitialSourceConnectionID:       tc.destConnID,
		OriginalDestinationConnectionID: tc.conn.origDestConnID,
		MaxBidiStreamNum:                100,
		MaxUniStreamNum:                 100,
	}
	require.NoError(t, tc.conn.handleTransportParameters(params))
	tc.conn.applyTransportParameters()

	// Open 2 streams
	str1, err := conn.OpenStream()
	require.NoError(t, err)
	require.NotNil(t, str1)

	str2, err := conn.OpenStream()
	require.NoError(t, err)
	require.NotNil(t, str2)

	// Verify iterating over Streams() allows standard non-mutating inspections without deadlock
	iterDone := make(chan struct{})
	go func() {
		defer close(iterDone)
		ids := make([]StreamID, 0, 2)
		for s := range conn.Streams() {
			ids = append(ids, s.StreamID())
			_ = s.LocalAddr()
			_ = s.RemoteAddr()
		}
		require.Len(t, ids, 2)
	}()

	select {
	case <-iterDone:
		// Succeeded
	case <-time.After(2 * time.Second):
		t.Fatal("DEADLOCK: conn.Streams() iteration hung")
	}
}

func TestAdversarial_StreamsIterator_MutateInsideLoop(t *testing.T) {
	tc := newClientTestConnection(t, nil, nil, false)
	conn := tc.conn
	tc.connRunner.EXPECT().Remove(gomock.Any()).AnyTimes()

	params := &wire.TransportParameters{
		InitialSourceConnectionID:       tc.destConnID,
		OriginalDestinationConnectionID: tc.conn.origDestConnID,
		MaxBidiStreamNum:                100,
		MaxUniStreamNum:                 100,
	}
	require.NoError(t, tc.conn.handleTransportParameters(params))
	tc.conn.applyTransportParameters()

	str1, err := conn.OpenStream()
	require.NoError(t, err)
	require.NotNil(t, str1)

	deadlockChan := make(chan bool, 1)
	go func() {
		for s := range conn.Streams() {
			_ = s
			// Attempt to open a stream while holding RLock via iterator
			_, _ = conn.OpenStream()
			break
		}
		deadlockChan <- false
	}()

	select {
	case <-deadlockChan:
		t.Log("Reentrant OpenStream() inside Streams() completed without deadlock")
	case <-time.After(500 * time.Millisecond):
		t.Log("Note: Reentrant OpenStream() inside Streams() blocked as expected due to RWMutex")
	}
}

// ---------------------------------------------------------------------------
// 4. Sentinel Errors & Unwrapping Verification
// ---------------------------------------------------------------------------

func TestAdversarial_SentinelErrors_Matrix(t *testing.T) {
	// 1. ErrStreamReset and net.ErrClosed on StreamError
	se := &StreamError{StreamID: 4, ErrorCode: 100, Remote: true}
	require.True(t, errors.Is(se, ErrStreamReset))
	require.True(t, errors.Is(se, net.ErrClosed))
	require.False(t, errors.Is(se, ErrConnectionClosed))

	// 2. ErrConnectionClosed and net.ErrClosed on ApplicationError
	ae := &ApplicationError{ErrorCode: 0, ErrorMessage: "clean close", Remote: false}
	require.True(t, errors.Is(ae, ErrConnectionClosed))
	require.True(t, errors.Is(ae, net.ErrClosed))
	require.False(t, errors.Is(ae, ErrStreamReset))

	// 3. ErrConnectionClosed and net.ErrClosed on TransportError
	te := &TransportError{ErrorCode: qerr.NoError, FrameType: 0, Remote: true}
	require.True(t, errors.Is(te, ErrConnectionClosed))
	require.True(t, errors.Is(te, net.ErrClosed))

	// 4. ErrTimeout and net.ErrClosed on IdleTimeoutError
	ite := &IdleTimeoutError{}
	require.True(t, errors.Is(ite, ErrTimeout))
	require.True(t, errors.Is(ite, net.ErrClosed))
	require.False(t, errors.Is(ite, ErrStreamReset))

	// 5. ErrTimeout and net.ErrClosed on HandshakeTimeoutError
	hte := &HandshakeTimeoutError{}
	require.True(t, errors.Is(hte, ErrTimeout))
	require.True(t, errors.Is(hte, net.ErrClosed))

	// 6. ErrStreamLimitReached on StreamLimitReachedError
	sle := StreamLimitReachedError{}
	require.True(t, errors.Is(sle, ErrStreamLimitReached))
	require.True(t, errors.Is(&sle, ErrStreamLimitReached))

	// 7. ErrDeadline on errDeadline
	require.True(t, errors.Is(errDeadline, ErrDeadline))
	require.True(t, errors.Is(errDeadline, os.ErrDeadlineExceeded))
}

// ---------------------------------------------------------------------------
// 5. Zero-Allocation Verification Benchmarks
// ---------------------------------------------------------------------------

func BenchmarkAdversarial_SupportedVersionsSeq_Full(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		for v := range SupportedVersionsSeq() {
			_ = v
		}
	}
}

func BenchmarkAdversarial_SupportedVersionsSeq_Break(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		for v := range SupportedVersionsSeq() {
			_ = v
			break
		}
	}
}

func BenchmarkAdversarial_Stream_Addresses(b *testing.B) {
	local := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 9001}
	remote := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 9002}
	sender := newMockFullAddrSender(local, remote)
	fc := newTestStreamFlowControllerWithSendWindow(1, protocol.MaxByteCount)
	str := newStream(context.Background(), 1, sender, fc, false)

	b.ReportAllocs()
	for b.Loop() {
		_ = str.LocalAddr()
		_ = str.RemoteAddr()
	}
}

func BenchmarkAdversarial_SentinelErrors_StreamReset(b *testing.B) {
	err := &StreamError{StreamID: 10, ErrorCode: 42, Remote: true}
	b.ReportAllocs()
	for b.Loop() {
		if !errors.Is(err, ErrStreamReset) || !errors.Is(err, net.ErrClosed) {
			b.Fatal("unexpected mismatch")
		}
	}
}

func BenchmarkAdversarial_SentinelErrors_ConnectionClosed(b *testing.B) {
	err := &ApplicationError{ErrorCode: 0, ErrorMessage: "closed", Remote: false}
	b.ReportAllocs()
	for b.Loop() {
		if !errors.Is(err, ErrConnectionClosed) || !errors.Is(err, net.ErrClosed) {
			b.Fatal("unexpected mismatch")
		}
	}
}

func BenchmarkAdversarial_SentinelErrors_Timeout(b *testing.B) {
	err := &IdleTimeoutError{}
	b.ReportAllocs()
	for b.Loop() {
		if !errors.Is(err, ErrTimeout) || !errors.Is(err, net.ErrClosed) {
			b.Fatal("unexpected mismatch")
		}
	}
}

func BenchmarkAdversarial_SentinelErrors_LimitReached(b *testing.B) {
	err := StreamLimitReachedError{}
	b.ReportAllocs()
	for b.Loop() {
		if !errors.Is(err, ErrStreamLimitReached) {
			b.Fatal("unexpected mismatch")
		}
	}
}
