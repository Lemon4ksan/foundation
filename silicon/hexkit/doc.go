// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package hexkit provides zero-allocation hexadecimal encoding and decoding.
//
// # Compared to the standard library
//
// Stdlib counterpart: encoding/hex
//
// Rejected compromise: Standard library hex encoding allocates new strings and byte slices on every call, creating garbage collector pressure.
//
// Accepted cost: Encoding and decoding happens into caller-provided buffers, pushing memory lifecycle management to the caller.
//
// Allocations: zero
package hexkit
