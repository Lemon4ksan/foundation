// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package borrow provides Rust-grade linear ownership semantics, zero-allocation
// borrowed handles, generational lifetime verification, and scoped execution arenas.
//
// # Core Concepts
//
// The borrow package implements the three fundamental axioms of affine logic and memory safety:
//
//  1. Single Ownership: Represented by [Box], an owned container that must be explicitly
//     consumed, moved, or released back to its allocator.
//  2. Aliasing XOR Mutability: A resource can have either multiple shared readers ([Ref])
//     or exactly one exclusive writer ([Mut]), but never both simultaneously.
//  3. Outlives & Scoped Lifetimes: Resources created via [Scoped] are bound to their lexical
//     scope and automatically recycled with zero heap allocations upon function return.
//
// # Runtime Generation Checks
//
// To prevent silent Use-After-Free (UAF) and data corruption, all borrowed references and
// zero-copy slices ([Bytes]) carry a generation counter. Attempting to access memory from
// an invalidated or recycled resource results in an immediate, deterministic panic.
package borrow
