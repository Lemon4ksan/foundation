// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package minheap provides a high-performance, generic binary minimum heap ordered by key.
//
// Heap represents a generic, contiguous array-backed minimum heap suitable for priority queues, timers, and nearest-neighbor searches. Elements are ordered based on a comparable key. Push and Pop operations maintain heap invariants in logarithmic time.
//
// Use [Heap.Push], [Heap.Pop], and [Heap.Peek] to interact with the priority queue.
//
// # Compared to the standard library
//
// Stdlib counterpart: container/heap
//
// Rejected compromise: container/heap requires implementing sort.Interface and Push/Pop methods with interface{} boxing, forcing runtime type checks and allocations.
//
// Accepted cost: The caller must manage the generic types, and heap mutations require slicing operations rather than simple interfaces.
//
// Allocations: amortized
package minheap
