// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package trie provides edge-compressed radix trees (Patricia tries) for zero-allocation
// exact key and longest-prefix lookups.
//
// Standard hash maps only support exact key lookups. Evaluating longest prefix matches
// or iterating over keys sharing a common prefix requires linear scans across all entries,
// taking O(N) time. Compressed radix search trees provide deterministic O(K) lookups
// where K is the key length, with zero heap allocations on read paths.
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: compressed radix tree and longest-prefix matching.
//
// Rejected compromise: Linear scanning of map keys or regex-based matching for prefix lookups.
//
// Accepted cost: Trees use more memory per entry than flat hash maps and require edge splitting on insertion.
//
// Allocations: amortized
package trie
