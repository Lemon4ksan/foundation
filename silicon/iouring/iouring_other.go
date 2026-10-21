// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !linux

package iouring

import (
	"context"
	"net"
	"time"
)

// Ring is a cross-platform stub on non-Linux operating systems.
type Ring struct{}

// New returns [ErrIOUringUnsupported] on non-Linux platforms.
func New(entries uint32, flags ...uint32) (*Ring, error) {
	return nil, ErrIOUringUnsupported
}

// FD returns -1 on non-Linux platforms.
func (r *Ring) FD() int {
	return -1
}

// Entries returns 0 on non-Linux platforms.
func (r *Ring) Entries() uint32 {
	return 0
}

// GetSQE returns [ErrIOUringUnsupported] on non-Linux platforms.
func (r *Ring) GetSQE() (*SQE, error) {
	return nil, ErrIOUringUnsupported
}

// Submit returns [ErrIOUringUnsupported] on non-Linux platforms.
func (r *Ring) Submit() (int, error) {
	return 0, ErrIOUringUnsupported
}

// SubmitAndWait returns [ErrIOUringUnsupported] on non-Linux platforms.
func (r *Ring) SubmitAndWait(waitNr uint32) (int, error) {
	return 0, ErrIOUringUnsupported
}

// PeekCQE returns false on non-Linux platforms.
func (r *Ring) PeekCQE() (CQE, bool) {
	return CQE{}, false
}

// WaitCQE returns [ErrIOUringUnsupported] on non-Linux platforms.
func (r *Ring) WaitCQE() (CQE, error) {
	return CQE{}, ErrIOUringUnsupported
}

// Close is a no-op on non-Linux platforms.
func (r *Ring) Close() error {
	return nil
}

// IOUringConn is a cross-platform stub implementing [net.Conn] on non-Linux platforms.
type IOUringConn struct{}

// DialIOUring returns [ErrIOUringUnsupported] on non-Linux platforms.
func DialIOUring(ctx context.Context, network, address string) (*IOUringConn, error) {
	return nil, ErrIOUringUnsupported
}

// Read returns [ErrIOUringUnsupported] on non-Linux platforms.
func (c *IOUringConn) Read(b []byte) (int, error) {
	return 0, ErrIOUringUnsupported
}

// Write returns [ErrIOUringUnsupported] on non-Linux platforms.
func (c *IOUringConn) Write(b []byte) (int, error) {
	return 0, ErrIOUringUnsupported
}

// Close is a no-op on non-Linux platforms.
func (c *IOUringConn) Close() error {
	return nil
}

// LocalAddr returns nil on non-Linux platforms.
func (c *IOUringConn) LocalAddr() net.Addr {
	return nil
}

// RemoteAddr returns nil on non-Linux platforms.
func (c *IOUringConn) RemoteAddr() net.Addr {
	return nil
}

// SetDeadline returns [ErrIOUringUnsupported] on non-Linux platforms.
func (c *IOUringConn) SetDeadline(t time.Time) error {
	return ErrIOUringUnsupported
}

// SetReadDeadline returns [ErrIOUringUnsupported] on non-Linux platforms.
func (c *IOUringConn) SetReadDeadline(t time.Time) error {
	return ErrIOUringUnsupported
}

// SetWriteDeadline returns [ErrIOUringUnsupported] on non-Linux platforms.
func (c *IOUringConn) SetWriteDeadline(t time.Time) error {
	return ErrIOUringUnsupported
}
