// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package disjointset implements an array-backed Disjoint Set Union (DSU / Union-Find) data structure.
// Elements are indexed from 0 to N-1. It provides near-O(1) amortized set operations
// using two-pass iterative path compression and union-by-size heuristics.
// All steady-state queries, unions, and resets operate with zero heap allocations.
package disjointset

// DisjointSet implements an array-backed Disjoint Set Union data structure.
//
// Read-only queries such as Len and Count (when not concurrently mutated) are safe
// for concurrent access. Because Find performs internal path compression (mutating parent pointers),
// all operations involving Find, Union, Connected, Size, and Reset must be externally
// synchronized (e.g. using sync.Mutex) in concurrent environments.
type DisjointSet struct {
	parent []int // parent[i] is the parent of element i; if parent[i] == i, i is the set root
	size   []int // size[i] is the number of elements in the component rooted at i
	count  int   // count is the current number of disjoint sets / connected components
}

// New creates and initializes a new DisjointSet tracking n elements (0 to n-1).
// Each element initially belongs to its own singleton set with size 1.
//
// Panics with "disjointset: negative element count" if n < 0.
// If n == 0, returns an empty DisjointSet.
func New(n int) *DisjointSet {
	if n < 0 {
		panic("disjointset: negative element count")
	}
	if n == 0 {
		return &DisjointSet{}
	}

	parent := make([]int, n)
	size := make([]int, n)
	for i := 0; i < n; i++ {
		parent[i] = i
		size[i] = 1
	}

	return &DisjointSet{
		parent: parent,
		size:   size,
		count:  n,
	}
}

// Len returns the total number of elements tracked by the disjoint set.
func (d *DisjointSet) Len() int {
	if d == nil {
		return 0
	}
	return len(d.parent)
}

// Count returns the number of disjoint sets (connected components) currently tracked.
func (d *DisjointSet) Count() int {
	if d == nil {
		return 0
	}
	return d.count
}

// Find returns the representative root of the set containing element x.
// It executes full two-pass iterative path compression, pointing all traversed
// elements directly to the root for optimal amortized O(alpha(N)) performance.
//
// Panics with "disjointset: index out of range" if x is out of bounds [0, Len()).
func (d *DisjointSet) Find(x int) int {
	if d == nil || x < 0 || x >= len(d.parent) {
		panic("disjointset: index out of range")
	}

	// Fast path: x is already its own root.
	if d.parent[x] == x {
		return x
	}

	// Pass 1: Find the representative root.
	root := d.parent[x]
	for root != d.parent[root] {
		root = d.parent[root]
	}

	// Pass 2: Compress the path by updating all traversed nodes directly to root.
	curr := x
	for curr != root {
		next := d.parent[curr]
		d.parent[curr] = root
		curr = next
	}

	return root
}

// Union merges the disjoint sets containing elements x and y using the union-by-size heuristic.
// The root of the smaller set is attached as a child of the root of the larger set.
// If both sets have equal size, y's root is attached under x's root.
//
// Returns true if x and y belonged to different sets and were successfully merged,
// or false if they were already in the same set.
//
// Panics with "disjointset: index out of range" if either x or y is out of bounds [0, Len()).
func (d *DisjointSet) Union(x, y int) bool {
	rootX := d.Find(x)
	rootY := d.Find(y)
	if rootX == rootY {
		return false
	}

	// Attach smaller component to larger component.
	if d.size[rootX] < d.size[rootY] {
		d.parent[rootX] = rootY
		d.size[rootY] += d.size[rootX]
	} else {
		d.parent[rootY] = rootX
		d.size[rootX] += d.size[rootY]
	}
	d.count--
	return true
}

// Connected reports whether elements x and y belong to the same disjoint set.
//
// Panics with "disjointset: index out of range" if either x or y is out of bounds [0, Len()).
func (d *DisjointSet) Connected(x, y int) bool {
	return d.Find(x) == d.Find(y)
}

// Size returns the number of elements in the disjoint set containing element x.
//
// Panics with "disjointset: index out of range" if x is out of bounds [0, Len()).
func (d *DisjointSet) Size(x int) int {
	root := d.Find(x)
	return d.size[root]
}

// Reset restores all elements into individual singleton sets (parent[i] = i, size[i] = 1, count = Len()).
// Backing slices are reused in place without allocating new heap memory.
func (d *DisjointSet) Reset() {
	if d == nil {
		return
	}
	n := len(d.parent)
	for i := 0; i < n; i++ {
		d.parent[i] = i
		d.size[i] = 1
	}
	d.count = n
}
