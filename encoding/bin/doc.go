// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package bin provides high-performance, zero-allocation binary encoding, decoding, and JIT-cached automatic struct serialization.
//
// bin uses a fluent Reader and Writer interface to sequentially process primitive types without allocations. It includes sticky error handling to avoid repetitive error checks on every read. The struct marshaler uses JIT analysis to write directly via standard ABI-safe pointers, bypassing reflection in the hot path.
//
// Key entry points include [NewReader], [NewWriter], and [MarshalBE].
//
// # Compared to the standard library
//
// Stdlib counterpart: encoding/binary
//
// Rejected compromise: encoding/binary uses reflection extensively for structs, causing significant overhead and allocations on every call.
//
// Accepted cost: JIT caching requires an initial setup cost, and ABI-safe pointer arithmetic bypasses some of Go's safe memory boundaries.
//
// Allocations: zero
package bin
