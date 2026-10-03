// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package offheap provides unmanaged direct memory allocation and slab allocators
// bypassing the Go Garbage Collector (GC).
//
// Maintaining hundreds of megabytes of cached network packets, proxy buffers, or
// long-lived protocol state on the Go runtime heap forces the garbage collector
// to traverse large pointer graphs on every GC cycle. This traversal introduces
// unpredictable Stop-The-World latency spikes and causes heap fragmentation.
// Allocating raw memory outside the Go heap eliminates GC scan overhead and provides
// deterministic memory lifecycle control. This package allocates contiguous memory pages
// directly via OS virtual memory APIs without registering pointers in the Go runtime heap.
//
// Types allocated with [AllocStruct] MUST be Plain Old Data (POD) structures.
// They MUST NOT contain Go heap pointers, strings, maps, channels, or slice headers.
// The Go GC does NOT scan off-heap physical pages. Any Go heap object referenced from
// off-heap memory will appear unreachable to the GC and be collected, causing use-after-free.
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: memory allocated directly from the OS bypassing the Go GC.
//
// Rejected compromise: The Go runtime garbage collector scanning large pointer graphs of cached data on every GC cycle, causing unpredictable latency spikes.
//
// Accepted cost: Memory is entirely unmanaged by the Go runtime and must be explicitly freed to prevent leaks. The caller assumes all safety responsibilities, including ensuring that only POD structures are stored off-heap.
//
// Allocations: amortized
package offheap
