// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package deque implements a high-performance, generic double-ended circular queue.
//
// Deque provides amortized O(1) insertions and deletions at both the front and back. It uses a contiguous generic slice, avoiding the overhead of node-based data structures. The zero value is empty and ready for use. Backing storage is lazily allocated on the first push operation.
//
// Key operations include [Deque.PushFront], [Deque.PushBack], [Deque.PopFront], and [Deque.PopBack].
//
// # Compared to the standard library
//
// Stdlib counterpart: container/list
//
// Rejected compromise: container/list boxes every element into a heap-allocated list.Element and requires type assertions for values.
//
// Accepted cost: Reallocations during growth will temporarily pause operations, and elements must be homogeneous in type.
//
// Allocations: amortized
package deque
