// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package list implements a high-performance, array-backed generic doubly linked list.
//
// List provides standard list manipulations (push, pop, move, remove) while avoiding the type assertion overhead typical of container/list. Using Go generics, elements remain strongly typed. Iteration is available via Push iterators like [List.Values] and [List.Backward].
//
// Start with [New] and populate with [List.PushBack] or [List.PushFront].
//
// # Compared to the standard library
//
// Stdlib counterpart: container/list
//
// Rejected compromise: container/list boxes all elements inside an interface{}, which degrades performance and memory locality.
//
// Accepted cost: Pointer jumping is still present, meaning traversal may not be as cache-friendly as contiguous arrays.
//
// Allocations: amortized
package list
