// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package borrow provides Rust-grade linear ownership semantics, zero-allocation borrowed handles, generational lifetime verification, and scoped execution arenas.
//
// The borrow package implements the three fundamental axioms of affine logic and memory safety. Single Ownership is represented by Box, an owned container that must be explicitly consumed, moved, or released back to its allocator. Aliasing XOR Mutability ensures a resource can have either multiple shared readers (Ref) or exactly one exclusive writer (Mut), but never both simultaneously. Outlives and Scoped Lifetimes guarantee that resources created via Scoped are bound to their lexical scope and automatically recycled. To prevent silent Use-After-Free (UAF) and data corruption, all borrowed references and zero-copy slices carry a generation counter. Attempting to access memory from an invalidated or recycled resource results in an immediate, deterministic panic.
//
// # Compared to the standard library
//
// Stdlib counterpart: sync.Pool
//
// Rejected compromise: Allowing concurrent mutation and missing Use-After-Free verification.
//
// Accepted cost: Developer ergonomics of explicit lifecycle management and minor CPU overhead from generation counters.
//
// Allocations: zero
package borrow
