// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package iouring

import (
	"errors"
	"syscall"
	"unsafe"
)

var (
	// ErrIOUringUnsupported is returned on non-Linux platforms where io_uring is unavailable.
	ErrIOUringUnsupported = errors.New("iouring: io_uring is only supported on Linux")

	// ErrRingFull is returned when no submission queue entries are available in the ring.
	ErrRingFull = errors.New("iouring: submission queue is full")

	// ErrRingClosed is returned when operations are attempted on a closed ring.
	ErrRingClosed = errors.New("iouring: ring is closed")
)

// Standard Linux io_uring operation codes (enum io_uring_op).
const (
	// OpNop submits a no-op completion event for ring probing and latency benchmarking.
	OpNop uint8 = 0
	// OpReadv submits a vectored readv operation.
	OpReadv uint8 = 1
	// OpWritev submits a vectored writev operation.
	OpWritev uint8 = 2
	// OpFsync submits a file synchronization operation.
	OpFsync uint8 = 3
	// OpTimeout submits a relative or absolute kernel timeout operation.
	OpTimeout uint8 = 11
	// OpAccept submits an asynchronous socket accept operation.
	OpAccept uint8 = 13
	// OpConnect submits an asynchronous socket connect operation.
	OpConnect uint8 = 16
	// OpClose submits an asynchronous file descriptor close operation.
	OpClose uint8 = 19
	// OpRead submits a pread-style contiguous buffer read operation.
	OpRead uint8 = 22
	// OpWrite submits a pwrite-style contiguous buffer write operation.
	OpWrite uint8 = 23
	// OpSend submits a socket send operation.
	OpSend uint8 = 26
	// OpRecv submits a socket recv operation.
	OpRecv uint8 = 27
	// OpShutdown submits a socket shutdown operation.
	OpShutdown uint8 = 34
	// OpSendZC submits a zero-copy socket send operation (Linux 6.0+).
	OpSendZC uint8 = 47
	// OpSendMsgZC submits a zero-copy sendmsg operation (Linux 6.1+).
	OpSendMsgZC uint8 = 48
)

// Flags for sys_io_uring_enter.
const (
	// IORING_ENTER_GETEVENTS instructs the kernel to wait for completion events.
	IORING_ENTER_GETEVENTS uint32 = 1 << 0
	// IORING_ENTER_SQ_WAKEUP wakes up the kernel SQPOLL thread when parked.
	IORING_ENTER_SQ_WAKEUP uint32 = 1 << 1
	// IORING_ENTER_SQ_WAIT blocks until submission queue space becomes available.
	IORING_ENTER_SQ_WAIT uint32 = 1 << 2
	// IORING_ENTER_EXT_ARG passes an extended argument structure to sys_io_uring_enter.
	IORING_ENTER_EXT_ARG uint32 = 1 << 3
)

// Flags for sys_io_uring_setup.
const (
	// IORING_SETUP_IOPOLL enables busy-waiting for I/O completion rather than IRQ delivery.
	IORING_SETUP_IOPOLL uint32 = 1 << 0
	// IORING_SETUP_SQPOLL spawns a kernel thread to poll the submission queue without syscalls.
	IORING_SETUP_SQPOLL uint32 = 1 << 1
	// IORING_SETUP_SQ_AFF binds the kernel SQPOLL thread to a specific CPU core.
	IORING_SETUP_SQ_AFF uint32 = 1 << 2
	// IORING_SETUP_CQSIZE overrides the completion queue entry count.
	IORING_SETUP_CQSIZE uint32 = 1 << 3
	// IORING_SETUP_CLAMP clamps ring entry counts to system maximums instead of returning EINVAL.
	IORING_SETUP_CLAMP uint32 = 1 << 4
	// IORING_SETUP_ATTACH_WQ shares the asynchronous worker pool of an existing io_uring fd.
	IORING_SETUP_ATTACH_WQ uint32 = 1 << 5
	// IORING_SETUP_R_DISABLED initializes the ring in a disabled state until explicitly enabled.
	IORING_SETUP_R_DISABLED uint32 = 1 << 6
)

// Kernel mmap offsets for io_uring file descriptors.
const (
	// IORING_OFF_SQ_RING is the mmap offset for the submission ring metadata and index array.
	IORING_OFF_SQ_RING int64 = 0
	// IORING_OFF_CQ_RING is the mmap offset for the completion ring metadata and CQE array.
	IORING_OFF_CQ_RING int64 = 0x8000000
	// IORING_OFF_SQES is the mmap offset for the contiguous SQE array.
	IORING_OFF_SQES int64 = 0x10000000
)

// SQE represents a 64-byte Linux io_uring submission queue entry (struct io_uring_sqe).
type SQE struct {
	Opcode      uint8
	Flags       uint8
	IOPrio      uint16
	Fd          int32
	Off         uint64
	Addr        uint64
	Len         uint32
	OpFlags     uint32
	UserData    uint64
	BufIndex    uint16
	Personality uint16
	SpliceFdIn  int32
	Pad2        [2]uint64
}

// PrepNop configures the [SQE] as an [OpNop] operation with the given userData correlation tag.
func (s *SQE) PrepNop(userData uint64) {
	*s = SQE{
		Opcode:   OpNop,
		Fd:       -1,
		UserData: userData,
	}
}

// PrepRead configures the [SQE] as an [OpRead] operation into buf at file offset off.
func (s *SQE) PrepRead(fd int32, buf []byte, off, userData uint64) {
	var addr uint64
	if len(buf) > 0 {
		addr = uint64(uintptr(unsafe.Pointer(&buf[0])))
	}
	*s = SQE{
		Opcode:   OpRead,
		Fd:       fd,
		Off:      off,
		Addr:     addr,
		Len:      uint32(len(buf)),
		UserData: userData,
	}
}

// PrepWrite configures the [SQE] as an [OpWrite] operation from buf at file offset off.
func (s *SQE) PrepWrite(fd int32, buf []byte, off, userData uint64) {
	var addr uint64
	if len(buf) > 0 {
		addr = uint64(uintptr(unsafe.Pointer(&buf[0])))
	}
	*s = SQE{
		Opcode:   OpWrite,
		Fd:       fd,
		Off:      off,
		Addr:     addr,
		Len:      uint32(len(buf)),
		UserData: userData,
	}
}

// PrepRecv configures the [SQE] as an [OpRecv] socket operation into buf.
func (s *SQE) PrepRecv(fd int32, buf []byte, msgFlags uint32, userData uint64) {
	var addr uint64
	if len(buf) > 0 {
		addr = uint64(uintptr(unsafe.Pointer(&buf[0])))
	}
	*s = SQE{
		Opcode:   OpRecv,
		Fd:       fd,
		Addr:     addr,
		Len:      uint32(len(buf)),
		OpFlags:  msgFlags,
		UserData: userData,
	}
}

// PrepSend configures the [SQE] as an [OpSend] socket operation from buf.
func (s *SQE) PrepSend(fd int32, buf []byte, msgFlags uint32, userData uint64) {
	var addr uint64
	if len(buf) > 0 {
		addr = uint64(uintptr(unsafe.Pointer(&buf[0])))
	}
	*s = SQE{
		Opcode:   OpSend,
		Fd:       fd,
		Addr:     addr,
		Len:      uint32(len(buf)),
		OpFlags:  msgFlags,
		UserData: userData,
	}
}

// PrepFsync configures the [SQE] as an [OpFsync] file synchronization operation.
func (s *SQE) PrepFsync(fd int32, fsyncFlags uint32, userData uint64) {
	*s = SQE{
		Opcode:   OpFsync,
		Fd:       fd,
		OpFlags:  fsyncFlags,
		UserData: userData,
	}
}

// PrepClose configures the [SQE] as an [OpClose] file descriptor close operation.
func (s *SQE) PrepClose(fd int32, userData uint64) {
	*s = SQE{
		Opcode:   OpClose,
		Fd:       fd,
		UserData: userData,
	}
}

// CQE represents a 16-byte Linux io_uring completion queue entry (struct io_uring_cqe).
type CQE struct {
	UserData uint64
	Res      int32
	Flags    uint32
}

// Err converts a negative [CQE] result code into a [syscall.Errno] error, or returns nil on success.
func (c CQE) Err() error {
	if c.Res < 0 {
		return syscall.Errno(-c.Res)
	}
	return nil
}
