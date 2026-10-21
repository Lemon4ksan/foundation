// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build linux

package iouring

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	sysIOUringSetup    = 425
	sysIOUringEnter    = 426
	sysIOUringRegister = 427
)

type ioSQOffsets struct {
	head        uint32
	tail        uint32
	ringMask    uint32
	ringEntries uint32
	flags       uint32
	dropped     uint32
	array       uint32
	resv1       uint32
	userAddr    uint64
}

type ioCQOffsets struct {
	head        uint32
	tail        uint32
	ringMask    uint32
	ringEntries uint32
	overflow    uint32
	cqes        uint32
	flags       uint32
	resv1       uint32
	userAddr    uint64
}

type ioUringParams struct {
	sqEntries    uint32
	cqEntries    uint32
	flags        uint32
	sqThreadCPU  uint32
	sqThreadIdle uint32
	features     uint32
	wqFD         uint32
	resv         [3]uint32
	sqOff        ioSQOffsets
	cqOff        ioCQOffsets
}

type sqRing struct {
	khead        *uint32
	ktail        *uint32
	kringMask    *uint32
	kringEntries *uint32
	kflags       *uint32
	kdropped     *uint32
	array        []uint32
	sqes         []SQE
	sqeTail      uint32
	ringMmap     []byte
	sqesMmap     []byte
}

type cqRing struct {
	khead        *uint32
	ktail        *uint32
	kringMask    *uint32
	kringEntries *uint32
	koverflow    *uint32
	cqes         []CQE
	ringMmap     []byte
}

// Ring encapsulates a Linux io_uring instance backed by shared user-kernel memory rings.
type Ring struct {
	mu      sync.Mutex
	fd      int
	flags   uint32
	sq      sqRing
	cq      cqRing
	closed  atomic.Bool
	entries uint32
}

// New creates a new [Ring] with the specified number of queue entries and optional setup flags.
func New(entries uint32, flags ...uint32) (*Ring, error) {
	if entries == 0 {
		entries = 64
	}

	var setupFlags uint32
	if len(flags) > 0 {
		setupFlags = flags[0]
	}

	var p ioUringParams
	p.flags = setupFlags

	r1, _, err := unix.Syscall(sysIOUringSetup, uintptr(entries), uintptr(unsafe.Pointer(&p)), 0)
	if err != 0 {
		return nil, fmt.Errorf("iouring: sys_io_uring_setup failed: %w", err)
	}

	ringFd := int(r1)
	r := &Ring{
		fd:      ringFd,
		flags:   p.flags,
		entries: p.sqEntries,
	}

	if err := r.mmapRings(&p); err != nil {
		_ = unix.Close(ringFd)
		return nil, err
	}

	return r, nil
}

// FD returns the underlying kernel file descriptor of the io_uring instance.
func (r *Ring) FD() int {
	return r.fd
}

// Entries returns the configured submission queue capacity of the ring.
func (r *Ring) Entries() uint32 {
	return r.entries
}

func (r *Ring) mmapRings(p *ioUringParams) error {
	sqSize := uintptr(p.sqOff.array + p.sqEntries*4)
	cqSize := uintptr(p.cqOff.cqes + p.cqEntries*uint32(unsafe.Sizeof(CQE{})))

	if p.features&1 != 0 { // IORING_FEAT_SINGLE_MMAP
		if cqSize > sqSize {
			sqSize = cqSize
		}
		cqSize = sqSize
	}

	sqPtr, err := unix.Mmap(
		r.fd,
		IORING_OFF_SQ_RING,
		int(sqSize),
		unix.PROT_READ|unix.PROT_WRITE,
		unix.MAP_SHARED|unix.MAP_POPULATE,
	)
	if err != nil {
		return fmt.Errorf("iouring: mmap SQ ring: %w", err)
	}

	r.sq.ringMmap = sqPtr
	basePtr := unsafe.Pointer(&sqPtr[0])

	r.sq.khead = (*uint32)(unsafe.Add(basePtr, p.sqOff.head))
	r.sq.ktail = (*uint32)(unsafe.Add(basePtr, p.sqOff.tail))
	r.sq.kringMask = (*uint32)(unsafe.Add(basePtr, p.sqOff.ringMask))
	r.sq.kringEntries = (*uint32)(unsafe.Add(basePtr, p.sqOff.ringEntries))
	r.sq.kflags = (*uint32)(unsafe.Add(basePtr, p.sqOff.flags))
	r.sq.kdropped = (*uint32)(unsafe.Add(basePtr, p.sqOff.dropped))

	arrayPtr := unsafe.Add(basePtr, p.sqOff.array)
	r.sq.array = unsafe.Slice((*uint32)(arrayPtr), p.sqEntries)

	sqesSize := uintptr(p.sqEntries) * unsafe.Sizeof(SQE{})
	sqesPtr, err := unix.Mmap(
		r.fd,
		IORING_OFF_SQES,
		int(sqesSize),
		unix.PROT_READ|unix.PROT_WRITE,
		unix.MAP_SHARED|unix.MAP_POPULATE,
	)
	if err != nil {
		_ = unix.Munmap(sqPtr)
		return fmt.Errorf("iouring: mmap SQEs: %w", err)
	}

	r.sq.sqesMmap = sqesPtr
	sqesBase := unsafe.Pointer(&sqesPtr[0])
	r.sq.sqes = unsafe.Slice((*SQE)(sqesBase), p.sqEntries)

	var (
		cqPtr     []byte
		cqBasePtr unsafe.Pointer
	)

	if p.features&1 != 0 {
		cqBasePtr = basePtr
	} else {
		cqPtr, err = unix.Mmap(
			r.fd,
			IORING_OFF_CQ_RING,
			int(cqSize),
			unix.PROT_READ|unix.PROT_WRITE,
			unix.MAP_SHARED|unix.MAP_POPULATE,
		)
		if err != nil {
			_ = unix.Munmap(sqesPtr)
			_ = unix.Munmap(sqPtr)
			return fmt.Errorf("iouring: mmap CQ ring: %w", err)
		}

		r.cq.ringMmap = cqPtr
		cqBasePtr = unsafe.Pointer(&cqPtr[0])
	}

	r.cq.khead = (*uint32)(unsafe.Add(cqBasePtr, p.cqOff.head))
	r.cq.ktail = (*uint32)(unsafe.Add(cqBasePtr, p.cqOff.tail))
	r.cq.kringMask = (*uint32)(unsafe.Add(cqBasePtr, p.cqOff.ringMask))
	r.cq.kringEntries = (*uint32)(unsafe.Add(cqBasePtr, p.cqOff.ringEntries))
	r.cq.koverflow = (*uint32)(unsafe.Add(cqBasePtr, p.cqOff.overflow))

	cqesPtr := unsafe.Add(cqBasePtr, p.cqOff.cqes)
	r.cq.cqes = unsafe.Slice((*CQE)(cqesPtr), p.cqEntries)

	return nil
}

// GetSQE retrieves the next available submission queue entry for population.
// The caller must serialize access or hold external synchronization.
func (r *Ring) GetSQE() (*SQE, error) {
	if r.closed.Load() {
		return nil, ErrRingClosed
	}

	head := atomic.LoadUint32(r.sq.khead)
	tail := r.sq.sqeTail

	if (tail - head) >= r.entries {
		return nil, ErrRingFull
	}

	index := tail & *r.sq.kringMask
	sqe := &r.sq.sqes[index]
	*sqe = SQE{}

	r.sq.array[index] = index
	r.sq.sqeTail = tail + 1

	return sqe, nil
}

// Submit flushes all populated SQEs to the kernel without blocking for completions.
func (r *Ring) Submit() (int, error) {
	return r.SubmitAndWait(0)
}

// SubmitAndWait flushes populated SQEs to the kernel and waits for at least waitNr completions.
func (r *Ring) SubmitAndWait(waitNr uint32) (int, error) {
	if r.closed.Load() {
		return 0, ErrRingClosed
	}

	toSubmit := r.sq.sqeTail - atomic.LoadUint32(r.sq.ktail)
	if toSubmit > 0 {
		atomic.StoreUint32(r.sq.ktail, r.sq.sqeTail)
	}

	var flags uint32
	if waitNr > 0 {
		flags |= IORING_ENTER_GETEVENTS
	}

	res, _, err := unix.Syscall6(
		sysIOUringEnter,
		uintptr(r.fd),
		uintptr(toSubmit),
		uintptr(waitNr),
		uintptr(flags),
		0,
		0,
	)
	if err != 0 {
		return 0, err
	}

	return int(res), nil
}

// PeekCQE consumes and returns a ready completion queue entry without making a syscall if available.
func (r *Ring) PeekCQE() (CQE, bool) {
	if r.closed.Load() {
		return CQE{}, false
	}

	head := atomic.LoadUint32(r.cq.khead)
	tail := atomic.LoadUint32(r.cq.ktail)
	if head == tail {
		return CQE{}, false
	}

	index := head & *r.cq.kringMask
	cqe := r.cq.cqes[index]
	atomic.StoreUint32(r.cq.khead, head+1)

	return cqe, true
}

// WaitCQE blocks until a completion queue event is ready, advances the CQ head, and returns it.
func (r *Ring) WaitCQE() (CQE, error) {
	for {
		if r.closed.Load() {
			return CQE{}, ErrRingClosed
		}

		if cqe, ok := r.PeekCQE(); ok {
			return cqe, nil
		}

		if _, err := r.SubmitAndWait(1); err != nil {
			if errors.Is(err, syscall.EINTR) {
				continue
			}

			return CQE{}, err
		}
	}
}

// Close unmaps kernel shared memory rings and closes the io_uring file descriptor.
func (r *Ring) Close() error {
	if !r.closed.CompareAndSwap(false, true) {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.sq.sqesMmap) > 0 {
		_ = unix.Munmap(r.sq.sqesMmap)
		r.sq.sqesMmap = nil
	}

	if len(r.sq.ringMmap) > 0 {
		_ = unix.Munmap(r.sq.ringMmap)
		r.sq.ringMmap = nil
	}

	if len(r.cq.ringMmap) > 0 {
		_ = unix.Munmap(r.cq.ringMmap)
		r.cq.ringMmap = nil
	}

	return unix.Close(r.fd)
}

// IOUringConn implements [net.Conn] by routing socket reads and writes through a dedicated Linux io_uring instance.
type IOUringConn struct {
	mu     sync.Mutex
	fd     int
	ring   *Ring
	laddr  net.Addr
	raddr  net.Addr
	closed atomic.Bool
}

// DialIOUring connects to a remote TCP target and binds the socket to an [IOUringConn].
func DialIOUring(ctx context.Context, network, address string) (*IOUringConn, error) {
	host, portStr, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil || len(ips) == 0 {
		return nil, fmt.Errorf("iouring: resolve host failed: %w", err)
	}

	ip := ips[0]
	port, err := net.DefaultResolver.LookupPort(ctx, network, portStr)
	if err != nil {
		return nil, err
	}

	family := unix.AF_INET
	if ip.To4() == nil {
		family = unix.AF_INET6
	}

	fd, err := unix.Socket(family, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("iouring: socket creation failed: %w", err)
	}

	ring, err := New(32)
	if err != nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("iouring: create ring failed: %w", err)
	}

	var sa unix.Sockaddr
	if family == unix.AF_INET {
		var sa4 unix.SockaddrInet4
		sa4.Port = port
		copy(sa4.Addr[:], ip.To4())
		sa = &sa4
	} else {
		var sa6 unix.SockaddrInet6
		sa6.Port = port
		copy(sa6.Addr[:], ip.To16())
		sa = &sa6
	}

	if err := unix.Connect(fd, sa); err != nil && err != unix.EINPROGRESS {
		_ = ring.Close()
		_ = unix.Close(fd)
		return nil, fmt.Errorf("iouring: connect failed: %w", err)
	}

	lsa, _ := unix.Getsockname(fd)
	var localAddr net.Addr
	if lsa != nil {
		localAddr = sockaddrToAddr(lsa)
	}

	return &IOUringConn{
		fd:    fd,
		ring:  ring,
		laddr: localAddr,
		raddr: &net.TCPAddr{IP: ip, Port: port},
	}, nil
}

// Read reads data from the socket via [OpRecv].
func (c *IOUringConn) Read(b []byte) (int, error) {
	if c.closed.Load() {
		return 0, io.EOF
	}
	if len(b) == 0 {
		return 0, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	sqe, err := c.ring.GetSQE()
	if err != nil {
		return 0, err
	}

	sqe.PrepRecv(int32(c.fd), b, 0, 1)

	if _, err := c.ring.Submit(); err != nil {
		return 0, err
	}

	cqe, err := c.ring.WaitCQE()
	if err != nil {
		return 0, err
	}

	if err := cqe.Err(); err != nil {
		return 0, err
	}
	if cqe.Res == 0 {
		return 0, io.EOF
	}

	return int(cqe.Res), nil
}

// Write writes data to the socket via [OpSend].
func (c *IOUringConn) Write(b []byte) (int, error) {
	if c.closed.Load() {
		return 0, io.ErrClosedPipe
	}
	if len(b) == 0 {
		return 0, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	total := 0
	for total < len(b) {
		sqe, err := c.ring.GetSQE()
		if err != nil {
			return total, err
		}

		sqe.PrepSend(int32(c.fd), b[total:], 0, 2)

		if _, err := c.ring.Submit(); err != nil {
			return total, err
		}

		cqe, err := c.ring.WaitCQE()
		if err != nil {
			return total, err
		}

		if err := cqe.Err(); err != nil {
			return total, err
		}

		total += int(cqe.Res)
	}

	return total, nil
}

// Close terminates the socket and releases the underlying io_uring instance.
func (c *IOUringConn) Close() error {
	if !c.closed.CompareAndSwap(false, true) {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	_ = c.ring.Close()

	return unix.Close(c.fd)
}

// LocalAddr returns the local network address.
func (c *IOUringConn) LocalAddr() net.Addr {
	return c.laddr
}

// RemoteAddr returns the remote network address.
func (c *IOUringConn) RemoteAddr() net.Addr {
	return c.raddr
}

// SetDeadline sets the read and write deadlines associated with the connection.
func (c *IOUringConn) SetDeadline(t time.Time) error {
	_ = c.SetReadDeadline(t)
	return c.SetWriteDeadline(t)
}

// SetReadDeadline sets the deadline for future Read calls.
func (c *IOUringConn) SetReadDeadline(t time.Time) error {
	return nil
}

// SetWriteDeadline sets the deadline for future Write calls.
func (c *IOUringConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func sockaddrToAddr(sa unix.Sockaddr) net.Addr {
	switch s := sa.(type) {
	case *unix.SockaddrInet4:
		return &net.TCPAddr{IP: s.Addr[:], Port: s.Port}
	case *unix.SockaddrInet6:
		return &net.TCPAddr{IP: s.Addr[:], Port: s.Port}
	}

	return &net.TCPAddr{IP: netip.IPv4Unspecified().AsSlice(), Port: 0}
}
