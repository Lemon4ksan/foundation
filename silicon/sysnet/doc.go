// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package sysnet provides low-level OS socket syscall overrides via syscall.RawConn,
// configuring TCP_QUICKACK, TCP_NODELAY, TCP_FASTOPEN, and SO_BUSY_POLL to minimize network tail latency.
//
// # Compared to the standard library
//
// Stdlib counterpart: net
//
// Rejected compromise: The standard net package hides underlying socket tuning and applies generic defaults, leaving high-frequency trading and low-latency systems unable to optimize kernel queues.
//
// Accepted cost: Relying on raw syscalls introduces OS-specific dependencies (Linux/Darwin) and makes network initialization more verbose.
//
// Allocations: amortized
package sysnet
