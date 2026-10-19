// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package disjointset

// Set represents an array-backed Disjoint Set Union data structure.
//
// The zero value is empty and ready to use.
// Read-only operations are safe for concurrent use; mutations require external synchronization.
// Note that [Set.Find], [Set.Connected], and [Set.Size] mutate internal state for path compression
// and require synchronization; use [Set.FindReadOnly], [Set.ConnectedReadOnly], and [Set.SizeReadOnly]
// for concurrent read-only access.
type Set struct {
	parent []int // parent[i] is the parent of element i; if parent[i] == i, i is the set root
	size   []int // size[i] is the number of elements in the component rooted at i
	count  int   // count is the current number of disjoint sets / connected components
}

// New creates and initializes a [Set] tracking n elements.
//
// Each element initially belongs to its own singleton set with size 1.
// Panics if n < 0.
func New(n int) *Set {
	if n < 0 {
		panic("disjointset: negative element count")
	}
	if n == 0 {
		return &Set{}
	}

	parent := make([]int, n)
	size := make([]int, n)
	for i := range n {
		parent[i] = i
		size[i] = 1
	}

	return &Set{
		parent: parent,
		size:   size,
		count:  n,
	}
}

// Len returns the total number of elements tracked by the disjoint set.
func (d *Set) Len() int {
	if d == nil {
		return 0
	}
	return len(d.parent)
}

// Count returns the number of disjoint sets currently tracked.
func (d *Set) Count() int {
	if d == nil {
		return 0
	}
	return d.count
}

// Find returns the representative root of the set containing element x.
//
// It executes full two-pass iterative path compression.
// Panics if x is out of bounds.
func (d *Set) Find(x int) int {
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

// FindReadOnly returns the representative root of the set containing element x
// without mutating internal state (no path compression).
//
// Safe for concurrent use with other read-only operations.
// Panics if x is out of bounds.
func (d *Set) FindReadOnly(x int) int {
	if d == nil || x < 0 || x >= len(d.parent) {
		panic("disjointset: index out of range")
	}

	root := x
	for root != d.parent[root] {
		root = d.parent[root]
	}
	return root
}

// Union merges the disjoint sets containing elements x and y using the union-by-size heuristic.
//
// Returns true if x and y belonged to different sets and were successfully merged, or false if they were already in the same set.
// Panics if either x or y is out of bounds.
func (d *Set) Union(x, y int) bool {
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
// Panics if either x or y is out of bounds.
func (d *Set) Connected(x, y int) bool {
	return d.Find(x) == d.Find(y)
}

// ConnectedReadOnly reports whether elements x and y belong to the same disjoint set
// without mutating internal state.
//
// Safe for concurrent use with other read-only operations.
// Panics if either x or y is out of bounds.
func (d *Set) ConnectedReadOnly(x, y int) bool {
	return d.FindReadOnly(x) == d.FindReadOnly(y)
}

// Size returns the number of elements in the disjoint set containing element x.
//
// Panics if x is out of bounds.
func (d *Set) Size(x int) int {
	root := d.Find(x)
	return d.size[root]
}

// SizeReadOnly returns the number of elements in the disjoint set containing element x
// without mutating internal state.
//
// Safe for concurrent use with other read-only operations.
// Panics if x is out of bounds.
func (d *Set) SizeReadOnly(x int) int {
	root := d.FindReadOnly(x)
	return d.size[root]
}

// Reset restores all elements into individual singleton sets.
func (d *Set) Reset() {
	if d == nil {
		return
	}
	n := len(d.parent)
	for i := range n {
		d.parent[i] = i
		d.size[i] = 1
	}
	d.count = n
}
