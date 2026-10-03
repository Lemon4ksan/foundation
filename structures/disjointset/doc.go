// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package disjointset implements an array-backed Disjoint Set Union (DSU / Union-Find) data structure.
//
// DisjointSet offers near-O(1) amortized set operations utilizing two-pass iterative path compression and union-by-size heuristics. It is optimized for zero steady-state allocations during Find and Union operations.
//
// Use [New] to instantiate, and [DisjointSet.Find], [DisjointSet.Union] to manage sets.
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: standard disjoint-set / union-find structure
//
// Rejected compromise: Using generic maps for union-find algorithms involves heavy constant factors, frequent rehashing, and many allocations.
//
// Accepted cost: Elements must be contiguous integers from 0 to N-1, requiring callers to maintain ID mappings for complex objects.
//
// Allocations: zero
package disjointset
