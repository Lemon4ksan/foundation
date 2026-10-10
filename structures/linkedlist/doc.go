// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package list implements an array-backed generic doubly linked list.
//
// List provides standard doubly linked list manipulations without the type assertion overhead of [container/list], utilizing Go generics for strong typing and Push iterators like [List.Values] and [List.Backward] for traversal.
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
