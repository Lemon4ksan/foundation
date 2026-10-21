// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package iouring_test

import (
	"context"
	"errors"
	"io"
	"net"
	"runtime"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"github.com/lemon4ksan/foundation/silicon/iouring"
)

func TestSQEAndCQELayout(t *testing.T) {
	if got := unsafe.Sizeof(iouring.SQE{}); got != 64 {
		t.Fatalf("sizeof(SQE) = %d, want 64", got)
	}
	if got := unsafe.Sizeof(iouring.CQE{}); got != 16 {
		t.Fatalf("sizeof(CQE) = %d, want 16", got)
	}

	var sqe iouring.SQE
	buf := []byte("hello")

	sqe.PrepNop(42)
	if sqe.Opcode != iouring.OpNop || sqe.Fd != -1 || sqe.UserData != 42 {
		t.Fatalf("PrepNop mismatch: %+v", sqe)
	}

	sqe.PrepRead(7, buf, 128, 100)
	if sqe.Opcode != iouring.OpRead || sqe.Fd != 7 || sqe.Off != 128 || sqe.Len != 5 || sqe.UserData != 100 {
		t.Fatalf("PrepRead mismatch: %+v", sqe)
	}

	sqe.PrepWrite(8, buf, 256, 101)
	if sqe.Opcode != iouring.OpWrite || sqe.Fd != 8 || sqe.Off != 256 || sqe.Len != 5 || sqe.UserData != 101 {
		t.Fatalf("PrepWrite mismatch: %+v", sqe)
	}

	sqe.PrepRecv(9, buf, 0x40, 102)
	if sqe.Opcode != iouring.OpRecv || sqe.Fd != 9 || sqe.OpFlags != 0x40 || sqe.Len != 5 || sqe.UserData != 102 {
		t.Fatalf("PrepRecv mismatch: %+v", sqe)
	}

	sqe.PrepSend(10, buf, 0x80, 103)
	if sqe.Opcode != iouring.OpSend || sqe.Fd != 10 || sqe.OpFlags != 0x80 || sqe.Len != 5 || sqe.UserData != 103 {
		t.Fatalf("PrepSend mismatch: %+v", sqe)
	}

	sqe.PrepFsync(11, 1, 104)
	if sqe.Opcode != iouring.OpFsync || sqe.Fd != 11 || sqe.OpFlags != 1 || sqe.UserData != 104 {
		t.Fatalf("PrepFsync mismatch: %+v", sqe)
	}

	sqe.PrepClose(12, 105)
	if sqe.Opcode != iouring.OpClose || sqe.Fd != 12 || sqe.UserData != 105 {
		t.Fatalf("PrepClose mismatch: %+v", sqe)
	}

	cqeOK := iouring.CQE{UserData: 1, Res: 64}
	if err := cqeOK.Err(); err != nil {
		t.Fatalf("expected nil CQE error, got %v", err)
	}

	cqeErr := iouring.CQE{UserData: 2, Res: -int32(syscall.EINVAL)}
	if !errors.Is(cqeErr.Err(), syscall.EINVAL) {
		t.Fatalf("expected EINVAL from CQE.Err(), got %v", cqeErr.Err())
	}
}

func TestIOUring_NonLinuxFallback(t *testing.T) {
	if runtime.GOOS == "linux" {
		t.Skip("skipping non-linux test on linux")
	}

	_, err := iouring.New(32)
	if !errors.Is(err, iouring.ErrIOUringUnsupported) {
		t.Fatalf("expected ErrIOUringUnsupported on %s, got %v", runtime.GOOS, err)
	}

	_, err = iouring.DialIOUring(context.Background(), "tcp", "127.0.0.1:80")
	if !errors.Is(err, iouring.ErrIOUringUnsupported) {
		t.Fatalf("expected ErrIOUringUnsupported on %s, got %v", runtime.GOOS, err)
	}
}

func TestIOUring_EchoServer_Linux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("skipping io_uring socket test on non-linux OS")
	}

	// 1. Direct Ring lifecycle, SQE/CQE operations, and error branches.
	ring, err := iouring.New(0)
	if err != nil {
		t.Fatalf("iouring.New(0) failed: %v", err)
	}
	if ring.FD() <= 0 || ring.Entries() == 0 {
		t.Fatalf("unexpected ring FD=%d Entries=%d", ring.FD(), ring.Entries())
	}
	if _, ok := ring.PeekCQE(); ok {
		t.Fatal("expected empty CQ before submission")
	}

	sqe, err := ring.GetSQE()
	if err != nil {
		t.Fatalf("GetSQE failed: %v", err)
	}
	sqe.PrepNop(777)
	if _, err := ring.SubmitAndWait(1); err != nil {
		t.Fatalf("SubmitAndWait(1) failed: %v", err)
	}
	cqe, ok := ring.PeekCQE()
	if !ok || cqe.UserData != 777 {
		t.Fatalf("PeekCQE = (%+v, %v), want UserData=777", cqe, ok)
	}

	// Fill SQ to capacity to test ErrRingFull.
	for i := uint32(0); i < ring.Entries(); i++ {
		s, err := ring.GetSQE()
		if err != nil {
			t.Fatalf("GetSQE(%d) failed: %v", i, err)
		}
		s.PrepNop(uint64(i))
	}
	if _, err := ring.GetSQE(); !errors.Is(err, iouring.ErrRingFull) {
		t.Fatalf("expected ErrRingFull when SQ is full, got %v", err)
	}
	if _, err := ring.Submit(); err != nil {
		t.Fatalf("Submit full batch failed: %v", err)
	}
	WaitCQE, err := ring.WaitCQE()
	if err != nil || WaitCQE.Err() != nil {
		t.Fatalf("WaitCQE failed: %v, cqe=%+v", err, WaitCQE)
	}

	if err := ring.Close(); err != nil {
		t.Fatalf("ring.Close() failed: %v", err)
	}
	if err := ring.Close(); err != nil {
		t.Fatalf("second ring.Close() failed: %v", err)
	}
	if _, err := ring.GetSQE(); !errors.Is(err, iouring.ErrRingClosed) {
		t.Fatalf("expected ErrRingClosed from GetSQE, got %v", err)
	}
	if _, err := ring.SubmitAndWait(1); !errors.Is(err, iouring.ErrRingClosed) {
		t.Fatalf("expected ErrRingClosed from SubmitAndWait, got %v", err)
	}
	if _, ok := ring.PeekCQE(); ok {
		t.Fatal("expected false from PeekCQE on closed ring")
	}
	if _, err := ring.WaitCQE(); !errors.Is(err, iouring.ErrRingClosed) {
		t.Fatalf("expected ErrRingClosed from WaitCQE, got %v", err)
	}

	// 2. DialIOUring error validation.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if _, err := iouring.DialIOUring(ctx, "tcp", "missing-port"); err == nil {
		t.Fatal("expected error for invalid host:port")
	}
	if _, err := iouring.DialIOUring(ctx, "tcp", "127.0.0.1:notaport"); err == nil {
		t.Fatal("expected error for invalid port")
	}

	// 3. Echo server end-to-end test.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start listener: %v", err)
	}
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		buf := make([]byte, 1024)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				return
			}

			_, _ = conn.Write(buf[:n])
		}
	}()

	conn, err := iouring.DialIOUring(ctx, "tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("failed to dial io_uring socket: %v", err)
	}
	defer conn.Close()

	_ = conn.LocalAddr()
	_ = conn.RemoteAddr()
	_ = conn.SetDeadline(time.Now().Add(time.Second))
	_ = conn.SetReadDeadline(time.Now().Add(time.Second))
	_ = conn.SetWriteDeadline(time.Now().Add(time.Second))

	if n, err := conn.Read(nil); n != 0 || err != nil {
		t.Fatalf("Read(nil) = (%d, %v), want (0, nil)", n, err)
	}
	if n, err := conn.Write(nil); n != 0 || err != nil {
		t.Fatalf("Write(nil) = (%d, %v), want (0, nil)", n, err)
	}

	payload := []byte("HELLO_IO_URING_SILICON_SPEED")

	n, err := conn.Write(payload)
	if err != nil || n != len(payload) {
		t.Fatalf("write failed: %v (wrote %d)", err, n)
	}

	readBuf := make([]byte, len(payload))
	n, err = io.ReadFull(conn, readBuf)
	if err != nil || n != len(payload) {
		t.Fatalf("read failed: %v (read %d)", err, n)
	}

	if string(readBuf) != string(payload) {
		t.Fatalf("mismatched payload: got %q, want %q", string(readBuf), string(payload))
	}

	_ = conn.Close()
	_ = conn.Close()
	if _, err := conn.Read(readBuf); !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF on Read after Close, got %v", err)
	}
	if _, err := conn.Write(payload); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("expected ErrClosedPipe on Write after Close, got %v", err)
	}
}
