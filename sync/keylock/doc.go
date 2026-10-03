// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package keylock provides a generic, thread-safe, striped/key-based locking mechanism.
//
// The [KeyMutex] allows separate goroutines to concurrently lock and work with different
// keys, preventing global application lock bottlenecks. It tracks per-key lock state and
// reference counts of waiting and holding goroutines. When the reference count drops
// to zero, the entry is atomically removed, preventing memory leaks in long-running
// applications with highly dynamic or infinite key sets.
//
// [KeyMutex.Unlock] panics if the key is not currently locked or does not exist,
// catching double-unlock and unlock-without-lock bugs at the point of failure.
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: locking by string/key identifier
//
// Rejected compromise: sync.Map or map[*]*sync.Mutex leaks memory over time as keys accumulate without automated cleanup.
//
// Accepted cost: slightly higher lock acquisition time due to global map lock and reference counting overhead.
//
// Allocations: amortized
package keylock
