// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package ipc provides local Inter-Process Communication over Unix domain sockets and Windows named pipes.
//
// The package exposes functions to create HTTP transports bound to local sockets, allowing standard
// HTTP clients to communicate with local daemons like Docker. Use [NewUnixTransport] for Unix sockets
// and [NewNamedPipeTransport] for Windows named pipes.
//
// # Compared to the standard library
//
// Stdlib counterpart: net
//
// Rejected compromise: none (this package primarily wraps standard HTTP client transports to bridge with OS-specific local sockets).
//
// Accepted cost: caller relies on standard library HTTP transports which allocate per request.
//
// Allocations: amortized
package ipc
