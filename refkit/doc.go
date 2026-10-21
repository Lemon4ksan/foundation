// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package refkit provides zero-allocation reflection inspection, struct layout caching,
// and high-speed struct tag parsing for high-throughput serializers and systems code.
//
// Standard Go reflection frequently introduces performance bottlenecks in schema
// builders, codecs, and serializers due to interface boxing heap allocations, slice
// allocations during field traversal, and repetitive tag string parsing. This package
// eliminates these overheads through five core primitives:
//
// 1. [VisibleFields]: A Go 1.27 push iterator (iter.Seq[reflect.StructField]) that
// traverses struct fields and embedded fields without allocating slices on the heap.
//
// 2. Unboxed Accessors: Non-allocating primitive extractors ([AsInt64], [AsUint64],
// [AsFloat64], [AsBool], [AsBytes], [AsString]) that inspect [reflect.Value] without
// calling [reflect.Value.Interface], preventing interface boxing allocations.
//
// 3. [SchemaCache]: A thread-safe, read-optimized cache storing parsed struct layouts
// ([StructSchema] and [FieldSchema]) with pre-computed field byte offsets (or
// [OffsetInvalid] for indirect pointer-embedded fields) and pre-parsed tags, delivering
// zero heap allocations on cache hit hot paths.
//
// 4. [Implements]: A generic interface compliance checker utilizing [reflect.TypeFor]
// that validates interface satisfaction safely without panics or boxing.
//
// 5. [Tag] and [ParseTag]: High-speed struct tag parsing with ASCII fast-path and stack
// buffering for zero-allocation key-value and option extraction.
//
// Allocating string conversion wrappers (such as ToString) are intentionally excluded
// per ADR-0003 to maintain zero-allocation invariants across the package.
//
// # Compared to the standard library
//
// Stdlib counterpart: reflect
//
// Rejected compromise: Slice allocations in reflect.VisibleFields, interface boxing
// allocations in reflect.Value.Interface, runtime panics on kind mismatches, and
// redundant string-scanning in reflect.StructTag.
//
// Accepted cost: Memory overhead for SchemaCache pre-computed layouts, stack-bounded
// recursion depth for deeply nested anonymous structs, and exclusion of generic string
// formatting utilities.
//
// Allocations: amortized
package refkit
