// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package proto implements a zero-allocation wire-format reader, scanner, and encoder
// for Protocol Buffers (proto2/proto3).
//
// Unlike full protobuf compilers and reflection runtimes, this package operates strictly
// at the binary wire level. It decodes and encodes varints, fixed-size integers, length-delimited
// records, map entries (field 1 = key, field 2 = value), packed repeated primitives across all scalar
// types, and embedded submessages directly on caller-provided byte buffers without heap allocations.
//
// The package provides zero-copy map entry scanning (ScanMapEntry, ScanMapEntries), safe embedded
// submessage recursion with depth and field budget protections (ScanMessage, WalkMessage), fast
// single-field and nested extractors (FindField, FindVarint, FindString, FindBytes) with branch-free
// skip logic, and 64-bit SWAR-vectorized varint processing exceeding 10 GB/s wire throughput.
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: zero-allocation Protocol Buffers wire encoding, decoding, packed arrays, map scanning, and recursion
//
// Rejected compromise: google.golang.org/protobuf uses heavy reflection, descriptor tables, and heap-allocated message trees.
//
// Accepted cost: callers work directly with field numbers, wire types, and raw byte views rather than generated structs.
//
// Allocations: zero
package proto
