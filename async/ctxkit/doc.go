// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package ctxkit provides a high-performance, flat-array implementation of [context.Context].
//
// The package is designed for high-throughput request pipelines where context metadata
// is frequently added. It stores up to 8 key-value pairs in a contiguous inline array,
// eliminating heap allocations and pointer-chasing for shallow context chains.
// If the capacity is exceeded, it falls back to a slice. Use [Wrap] at pipeline
// ingress to convert a standard [context.Context] into a [*Context], and then pass
// it everywhere. For generic value extraction, use [Value] and [ValueOr].
// A [Pool] is available for recycling contexts across requests to keep allocations at zero.
//
// # Compared to the standard library
//
// Stdlib counterpart: context
//
// Rejected compromise: context.WithValue boxes every key and value and allocates a new heap node, forming a linked list.
//
// Accepted cost: The context struct is larger, and deep context chains over 8 values will allocate a slice.
//
// Allocations: amortized
package ctxkit
