// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package proto implements a zero-allocation wire-format reader, scanner, and encoder
// for Protocol Buffers (proto2/proto3).
//
// Unlike full protobuf compilers and reflection runtimes, this package operates strictly
// at the binary wire level. It decodes and encodes varints, fixed-size integers, length-delimited
// records, and packed repeated fields directly on caller-provided byte buffers without heap allocations.
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: zero-allocation Protocol Buffers wire encoding
//
// Rejected compromise: google.golang.org/protobuf uses heavy reflection, descriptor tables, and heap-allocated message trees.
//
// Accepted cost: callers work directly with field numbers, wire types, and raw byte views rather than generated structs.
//
// Allocations: zero
package proto
