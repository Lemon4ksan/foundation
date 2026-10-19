// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sysnet

import (
	"io"
	"net"
	"syscall"
)

// TuneSocketConn applies low-latency OS syscall flags (TCP_NODELAY and socket buffer tuning) to a [net.Conn].
func TuneSocketConn(conn net.Conn) {
	TuneSocketConnWithFlags(conn, 0)
}

// TuneSocketConnWithFlags applies low-latency OS syscall flags (TCP_NODELAY, TCP_FASTOPEN, SO_BUSY_POLL) to a [net.Conn].
func TuneSocketConnWithFlags(conn net.Conn, flags uint64) {
	if conn == nil {
		return
	}

	if tcpConn, ok := conn.(*net.TCPConn); ok {
		_ = tcpConn.SetNoDelay(true)
		_ = tcpConn.SetKeepAlive(true)

		rawConn, err := tcpConn.SyscallConn()
		if err != nil {
			return
		}

		_ = rawConn.Control(func(fd uintptr) {
			if flags == 0 {
				tuneSocketFD(fd)
			} else {
				tuneSocketFlagsFD(fd, flags)
			}
		})
	}
}

// WriteVectorBuffers writes multiple byte slice buffers to conn in a single OS vectorized syscall (WSASend on Windows, writev on Linux).
//
//go:inline
func WriteVectorBuffers(conn net.Conn, buffers [][]byte) (int64, error) {
	if conn == nil || len(buffers) == 0 {
		return 0, nil
	}

	return writeVectorBuffers(conn, buffers)
}

func writeVectorBuffers(conn net.Conn, buffers [][]byte) (int64, error) {
	netBufs := make(net.Buffers, len(buffers))
	copy(netBufs, buffers)

	return netBufs.WriteTo(conn)
}

// BatchUDPConn wraps a [net.PacketConn] to execute vectorized UDP datagram batching.
type BatchUDPConn struct {
	net.PacketConn
}

// NewBatchUDPConn wraps pconn with vectorized socket datagram batching capabilities.
func NewBatchUDPConn(pconn net.PacketConn) *BatchUDPConn {
	return &BatchUDPConn{PacketConn: pconn}
}

// WriteVector writes multiple UDP datagram payload buffers in a single vectorized syscall.
//
//go:inline
func (c *BatchUDPConn) WriteVector(buffers [][]byte) (int64, error) {
	if c == nil || c.PacketConn == nil || len(buffers) == 0 {
		return 0, nil
	}

	return c.writeVector(buffers)
}

func (c *BatchUDPConn) writeVector(buffers [][]byte) (int64, error) {
	return c.writeVectorTo(buffers, nil)
}

// SyscallConnector wraps the standard SyscallConn method allowing access to underlying OS file descriptors.
type SyscallConnector interface {
	SyscallConn() (syscall.RawConn, error)
}

func advanceBuffers(buffers [][]byte, written int64) [][]byte {
	for len(buffers) > 0 && written > 0 {
		headLen := int64(len(buffers[0]))
		if headLen <= written {
			written -= headLen
			buffers = buffers[1:]
		} else {
			buffers[0] = buffers[0][written:]
			break
		}
	}
	return buffers
}

func writeVectorBuffersLoop(raw syscall.RawConn, buffers [][]byte) (int64, error) {
	var (
		totalWritten int64
		sysErr       error
	)

	remaining := make([][]byte, len(buffers))
	copy(remaining, buffers)
	errCtrl := raw.Write(func(fd uintptr) bool {
		for len(remaining) > 0 {
			for len(remaining) > 0 && len(remaining[0]) == 0 {
				remaining = remaining[1:]
			}
			if len(remaining) == 0 {
				break
			}

			var written int64
			written, sysErr = writeVectorBuffersFD(fd, remaining)
			if written > 0 {
				totalWritten += written
				remaining = advanceBuffers(remaining, written)
			}
			if sysErr != nil {
				return !isWouldBlock(sysErr)
			}
			if written == 0 {
				sysErr = io.ErrShortWrite
				return true
			}
		}
		return true
	})
	if errCtrl != nil {
		return totalWritten, errCtrl
	}
	return totalWritten, sysErr
}

// WriteVectorTo writes multiple UDP datagram payload buffers to a target destination address.
//
//go:inline
func (c *BatchUDPConn) WriteVectorTo(buffers [][]byte, addr net.Addr) (int64, error) {
	if c == nil || c.PacketConn == nil || len(buffers) == 0 {
		return 0, nil
	}

	return c.writeVectorTo(buffers, addr)
}

func (c *BatchUDPConn) writeVectorTo(buffers [][]byte, addr net.Addr) (int64, error) {
	if addr == nil {
		if sysConn, ok := c.PacketConn.(SyscallConnector); ok {
			if raw, err := sysConn.SyscallConn(); err == nil && raw != nil {
				totalWritten, err := writeVectorBuffersLoop(raw, buffers)
				if err == nil {
					return totalWritten, nil
				}
			}
		}
	}

	var total int64
	for _, buf := range buffers {
		if len(buf) == 0 {
			continue
		}
		var (
			n   int
			err error
		)
		if addr == nil {
			if w, ok := c.PacketConn.(io.Writer); ok {
				n, err = w.Write(buf)
			} else {
				n, err = c.WriteTo(buf, nil)
			}
		} else {
			n, err = c.WriteTo(buf, addr)
		}
		total += int64(n)
		if err != nil {
			return total, err
		}
	}

	return total, nil
}

// SetReadBuffer sets the size of the operating system's receive buffer associated with the UDP connection.
func (c *BatchUDPConn) SetReadBuffer(bytes int) error {
	if u, ok := c.PacketConn.(*net.UDPConn); ok {
		return u.SetReadBuffer(bytes)
	}

	return nil
}

// SetWriteBuffer sets the size of the operating system's transmit buffer associated with the UDP connection.
func (c *BatchUDPConn) SetWriteBuffer(bytes int) error {
	if u, ok := c.PacketConn.(*net.UDPConn); ok {
		return u.SetWriteBuffer(bytes)
	}

	return nil
}

// SetGSO configures UDP Generic Segmentation Offload (GSO) segment size on the underlying socket.
func (c *BatchUDPConn) SetGSO(segmentSize int) error {
	sysConn, ok := c.PacketConn.(SyscallConnector)
	if !ok {
		return nil
	}
	raw, err := sysConn.SyscallConn()
	if err != nil {
		return err
	}

	var sysErr error
	_ = raw.Control(func(fd uintptr) {
		sysErr = setUDPSegmentFD(fd, segmentSize)
	})
	return sysErr
}

// SetGRO enables or disables UDP Generic Receive Offload (GRO) on the underlying socket.
func (c *BatchUDPConn) SetGRO(enable bool) error {
	sysConn, ok := c.PacketConn.(SyscallConnector)
	if !ok {
		return nil
	}
	raw, err := sysConn.SyscallConn()
	if err != nil {
		return err
	}

	var sysErr error
	_ = raw.Control(func(fd uintptr) {
		sysErr = setUDPGROFD(fd, enable)
	})
	return sysErr
}

// SyscallConn returns a raw network connection for OS syscall access.
func (c *BatchUDPConn) SyscallConn() (syscall.RawConn, error) {
	if sys, ok := c.PacketConn.(SyscallConnector); ok {
		return sys.SyscallConn()
	}

	return nil, nil
}
