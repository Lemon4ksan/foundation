// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package json implements a silicon-grade, zero-allocation, lockless pure-Go JSON encoder and decoder.
//
// The package compiles type metadata once into an opcode execution sequence and performs direct memory writes
// via standard ABI-safe unsafe offsets without reflection overhead, runtime linkname hacks, or GC churn.
// For steady-state streaming data, [Iterator] and [Stream] provide completely zero-allocation parsing APIs.
// Note that convenience functions like [Marshal] and [Unmarshal] do allocate memory for boxing and slice creation.
//
// # Compared to the standard library
//
// Stdlib counterpart: encoding/json
//
// Rejected compromise: encoding/json prioritizes simplicity over speed, relying heavily on runtime reflection and heap allocations for every operation.
//
// Accepted cost: the fastest paths require manual iterator usage, and relying on unsafe memory layouts means it may require updates for new Go versions.
//
// Allocations: zero
package json
