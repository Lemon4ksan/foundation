// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package generic provides a lightweight, high-performance, and type-safe
// utility toolkit for Go. It leverages Go generics to eliminate repetitive
// boilerplate code, manual type assertions, and unsafe reflection on hot paths.
//
// The package is strictly divided into six cohesive areas of responsibility:
//   - Basic Primitives (types.go): Fluent pointer manipulation, null-coalescing, and ternary operations.
//   - Slice & Map Operations (slices.go): High-performance eager slicing, grouping, chunking, and in-place filtering.
//   - Lazy Pipeline Sequences (lazy.go): Stream-like deferred transformations (Map, Filter, Take, Drop) and folding
//     operations (Reduce) powered by Go iterators, eliminating intermediate memory allocations.
//   - Monadic Abstractions (monads.go): Swift-inspired type-safe error and nullability wrappers (Optional, Result,
//     and TypedResult) to enforce deterministic state checking and safe concurrent data transfer.
//   - Custom Collections (collections.go): Thread-safe, generic collections, including sets and TTL-based in-memory caches.
//   - Advanced Concurrency (concurrency.go): Parallel maps, asynchronous Futures, SingleFlight task-suppressors,
//     resilient backoff/retry algorithms, and batch DataLoaders.
//
// # Design Principles
//
//   - Generics-First: Every function is parameterized, guaranteeing compile-time type safety.
//   - Zero-Dependency: The package relies strictly on the Go standard library, keeping its footprint minimal.
//   - Performance & Memory Safety: Concurrency primitives are thread-safe, and slice helpers are optimized
//     to reduce allocations on the heap (e.g., using sync.Pool, capacity pre-allocation, or in-place memory reuse).
//
// # Basic Primitives & Pointers
//
// In Go, taking the address of literals directly (e.g. &100) is invalid. The [Ptr] helper resolves
// this cleanly, allowing inline pointer construction. Additionally, [Deref] and [DerefOr] provide
// safe dereferencing of pointers without nil-pointer panics:
//
//	type Config struct {
//		Limit *int
//	}
//
//	cfg := Config{
//		Limit: generic.Ptr(100), // Clean literal pointer
//	}
//
//	val := generic.DerefOr(cfg.Limit, 10) // Safely falls back to 10 if nil
//
// # High-Performance Slice Operations
//
// Manipulating slices in Go often requires writing repetitive loop structures.
// This package provides functional eager helpers like [IndexBy] and [FilterInPlace]:
//
//	items := []string{"foo", "bar", "baz"}
//
//	// Index slice into a map with O(1) lookups
//	indexed := generic.IndexBy(items, func(s string) string { return s[:1] }) // map[f:foo b:bar]
//
//	// Chunk slices into batches of size N
//	batches := generic.Chunked(items, 2) // [][]string{{"foo", "bar"}, {"baz"}}
//
// # Lazy Pipeline Sequences (Go 1.23+)
//
// When chaining multiple eager transformations (e.g., Map followed by Filter), Go traditionally
// allocates new intermediate slices for each stage. This package provides lazy alternatives
// using Go 1.23+ [iter.Seq] to process elements on-the-fly with O(1) space complexity:
//
//	numbers := []int{1, 2, 3, 4, 5}
//
//	// Build a lazy transformation pipeline. No memory is allocated during definition.
//	lazyPipeline := generic.FilterLazy(
//		generic.MapLazy(generic.ToSeq(numbers), func(n int) int { return n * 2 }),
//		func(n int) bool { return n > 5 },
//	)
//
//	// Collect results back into a slice, allocating memory only once.
//	results := generic.ToSlice(lazyPipeline) // []int{6, 8, 10}
//
// # Monadic Safety Abstractions & Application Guidelines
//
// The package introduces Swift-inspired monadic types - [Optional], [Result], and [TypedResult] - to
// provide deterministic state checking. While idiomatic Go prefers returning tuples like (T, error)
// or (T, bool) for linear execution, monadic abstractions solve specific architectural challenges where
// tuples cannot be easily expressed or where type ambiguity exists.
//
// ## Recommended Application Guidelines
//
//  1. Partial Updates (JSON / Config PATCH Requests):
//     Use [Optional] to distinguish between three distinct payload states without using pointers:
//     - Field absent in request -> Leave unchanged.
//     - Field provided as null/zero -> Reset value.
//     - Field provided with value -> Update value.
//
//     type UpdateUserDTO struct {
//     Name generic.Optional[string] `json:"name"`
//     }
//
//  2. Concurrent Worker Pools & Channel Transfer:
//     Go channels carry a single value per send (chan V). Returning (T, error) over a channel traditionally
//     requires defining temporary one-off response structs. Use [Result] as a standardized, single-value container:
//
//     results := make(chan generic.Result[UserData], 10)
//     results <- generic.Success(userData)
//     results <- generic.Failure[UserData](err)
//
//  3. Batch Operation Outcomes:
//     Store mixed success and failure states across multi-threaded operations cleanly in a slice ([]Result[T]).
//
//  4. Interface Footgun Protection ([TypedResult]):
//     In Go, assigning a nil concrete pointer to an error interface creates a non-nil interface value
//     (where err != nil evaluates to true). [TypedResult] uses an internal state flag to dynamically
//     prevent false-positive error checks across domain layer boundaries.
//
// ## Architectural Boundaries (Where NOT to use)
//
//   - Public Package Boundaries: Keep public exported function signatures idiomatic by returning standard (T, error).
//   - Linear Sequential Logic: Prefer standard "if err != nil" control flow over monadic chaining for simple execution paths.
//
// # Advanced Concurrency & Resilience
//
// Writing stable, multi-threaded pipelines in Go requires careful synchronization. This package provides
// highly optimized concurrency primitives:
//   - [ParallelMap]: Concurrently transforms a slice with a strict worker pool limit.
//   - [ParallelForEach]: Concurrently runs side-effect tasks with semaphore bounds and error aggregation.
//   - [Future]: Simple, thread-safe async-execution wrapper (Promise pattern).
//   - [SingleFlight]: Thread-safe task suppressor that merges concurrent calls to prevent backend spam.
//   - [Backoff]: Thread-safe exponential backoff with randomized jitter (AWS algorithm).
//
// # Concurrency Example
//
//	package main
//
//	import (
//		"context"
//		"fmt"
//		"time"
//		"github.com/lemon4ksan/foundation/generic"
//	)
//
//	func main() {
//		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//		defer cancel()
//
//		inputs := []string{"item1", "item2", "item3"}
//
//		// Process items concurrently, limiting to 2 parallel goroutines
//		results := generic.ParallelMap(ctx, inputs, 2, func(c context.Context, item string) string {
//			return item + "_processed"
//		})
//
//		fmt.Println("Processed results:", results)
//	}
package generic
