// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package borrow provides Rust-grade linear ownership semantics,
// zero-allocation borrowed handles, generational lifetime verification,
// and scoped execution arenas.
//
// # Core Concepts
//
//   - Single Ownership: Represented by [Box], an owned container that must
//     be explicitly consumed, moved, or released back to its allocator.
//   - Aliasing XOR Mutability: Enforced dynamically at runtime. [Box.Borrow]
//     acquires a shared immutable borrow ([Ref]), allowing multiple concurrent
//     readers. [Box.BorrowMut] acquires an exclusive mutable borrow ([Mut]).
//     Acquiring an exclusive borrow while readers exist, or acquiring any borrow
//     while an exclusive borrow is active, returns an error or panics.
//   - Explicit Borrow Lifecycle: Active borrows must be released via [Ref.Release]
//     or [Mut.Release] to free the slot for subsequent borrows. Double-release is
//     safe and idempotent. Exclusive borrows can also be downgraded into shared
//     borrows using [Mut.Freeze].
//   - Linear Move Semantics: [Box.Move] transfers ownership to a new container
//     and advances the slot generation, deterministically invalidating any
//     references borrowed before the move.
//   - Scoped Lifetimes & Bound Slots: Resources allocated within a [Scope]
//     (such as [Scope.AllocBytes] or [Alloc]) are bound to their lexical scope arena.
//     Calling [Bytes.Release] on a scope-allocated buffer releases that handle
//     without destroying the enclosing scope arena. Exiting the scope atomically
//     recycles all allocated resources.
//   - Generation Counters: To prevent silent Use-After-Free (UAF) and data
//     corruption, all borrowed references and zero-copy slices carry a
//     generation ID.
//
// Stdlib counterpart: sync.Pool
//
// Rejected compromise: Allowing concurrent mutation and missing Use-After-Free verification.
//
// Accepted cost: Developer ergonomics of explicit lifecycle management and minor CPU overhead from generation counters.
//
// Allocations: amortized
package borrow
