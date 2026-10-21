// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package iouring implements a zero-allocation Linux io_uring submission and completion
// ring-buffer engine for asynchronous network and storage I/O.
//
// By mapping submission ([SQE]) and completion ([CQE]) rings directly into shared
// user-kernel virtual memory via sys_io_uring_setup and mmap, callers can batch and
// execute socket and file descriptors operations without per-operation syscall
// transitions or heap allocations.
//
// On non-Linux platforms, all types and methods remain available for cross-platform
// compilation and deterministically return [ErrIOUringUnsupported].
//
// # Compared to the standard library
//
// Stdlib counterpart: net / os (epoll-backed runtime netpoller)
//
// Rejected compromise: The standard Go netpoller relies on readiness-based epoll
// notifications and issues individual read/write syscalls per I/O operation.
//
// Accepted cost: Direct shared-memory ring management requires Linux 5.1+ kernel
// support and explicit SQE/CQE lifecycle handling.
//
// Allocations: zero
package iouring
