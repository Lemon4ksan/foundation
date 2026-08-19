// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generic

import "slices"

// Map applies the transformer function fn to each element of the slice
// and returns a new slice containing the results.
//
// If the input slice is nil, Map returns nil.
//
// # Complexity
//
//   - Time: O(N)
//   - Space: O(N) allocations for the returned slice.
func Map[F, T any](slice []F, fn func(F) T) []T {
	if slice == nil || fn == nil {
		return nil
	}

	res := make([]T, len(slice))
	for i, v := range slice {
		res[i] = fn(v)
	}

	return res
}

// FlatMap applies the transformer function fn to each element of the slice
// and flattens the resulting slices into a single, merged slice.
//
// If the input slice is nil, FlatMap returns nil.
//
// # Complexity
//
//   - Time: O(N * M), where M is the average length of slices returned by fn.
//   - Space: O(N * M) allocations for the returned slice.
func FlatMap[F, T any](slice []F, fn func(F) []T) []T {
	if slice == nil || fn == nil {
		return nil
	}

	var res []T
	for _, v := range slice {
		res = append(res, fn(v)...)
	}

	return res
}

// Chunked splits a slice into multiple slices of the specified size.
//
// It returns nil if the slice is empty, or if size is less than or equal to 0.
// The last chunk may contain fewer elements if len(slice) is not a multiple of size.
//
// # Complexity
//
//   - Time: O(N)
//   - Space: O(N) since the returned chunks reference the original backing array.
func Chunked[T any](slice []T, size int) [][]T {
	if len(slice) == 0 || size <= 0 {
		return nil
	}

	chunks := make([][]T, 0, (len(slice)+size-1)/size)
	for i := 0; i < len(slice); i += size {
		end := i + size
		if end > len(slice) {
			end = len(slice)
		}

		chunks = append(chunks, slice[i:end])
	}

	return chunks
}

// Keys returns a slice containing all keys from the provided map.
// The order of the keys in the returned slice is undefined.
//
// If the map is nil, Keys returns nil.
//
// # Complexity
//
//   - Time: O(N)
//   - Space: O(N) allocations for the returned slice.
func Keys[K comparable, V any](m map[K]V) []K {
	if m == nil {
		return nil
	}

	res := make([]K, 0, len(m))
	for k := range m {
		res = append(res, k)
	}

	return res
}

// Values returns a slice containing all values from the provided map.
// The order of the values in the returned slice is undefined.
//
// If the map is nil, Values returns nil.
//
// # Complexity
//
//   - Time: O(N)
//   - Space: O(N) allocations for the returned slice.
func Values[K comparable, V any](m map[K]V) []V {
	if m == nil {
		return nil
	}

	res := make([]V, 0, len(m))
	for _, v := range m {
		res = append(res, v)
	}

	return res
}

// IndexBy returns a map where the keys are the results of applying fn to each
// element of the slice, and the values are the elements themselves.
//
// If the input slice is nil, IndexBy returns nil.
//
// # Complexity
//
//   - Time: O(N)
//   - Space: O(N) allocations for the returned map.
func IndexBy[K comparable, V any](slice []V, fn func(V) K) map[K]V {
	if slice == nil || fn == nil {
		return nil
	}

	res := make(map[K]V, len(slice))
	for _, v := range slice {
		res[fn(v)] = v
	}

	return res
}

// Unique returns a new slice containing only the unique elements from the original slice.
// It preserves the relative order of the elements.
//
// # Complexity
//
//   - Time: O(N)
//   - Space: O(N) allocations for the deduplication map and the returned slice.
func Unique[T comparable](slice []T) []T {
	if len(slice) == 0 {
		return slice
	}

	seen := make(map[T]struct{}, len(slice))

	res := make([]T, 0, len(slice))
	for _, v := range slice {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			res = append(res, v)
		}
	}

	return res
}

// GroupBy groups the elements of a slice into a map of slices based on a key
// extracted by the key-extractor function fn.
//
// If the input slice is nil, GroupBy returns nil.
//
// # Complexity
//
//   - Time: O(N)
//   - Space: O(N) allocations for the maps and grouped slices.
func GroupBy[K comparable, V any](slice []V, fn func(V) K) map[K][]V {
	if slice == nil || fn == nil {
		return nil
	}

	res := make(map[K][]V)
	for _, v := range slice {
		key := fn(v)
		res[key] = append(res[key], v)
	}

	return res
}

// Any reports whether at least one element of the slice satisfies the predicate fn.
func Any[T any](slice []T, fn func(T) bool) bool {
	if fn == nil {
		return false
	}

	return slices.ContainsFunc(slice, fn)
}

// All reports whether all elements of the slice satisfy the predicate fn.
func All[T any](slice []T, fn func(T) bool) bool {
	if fn == nil {
		return len(slice) == 0
	}

	for _, v := range slice {
		if !fn(v) {
			return false
		}
	}

	return true
}

// Find searches for the first element in the slice that satisfies the predicate fn.
// It returns the found element and a boolean indicating success. If no element matches,
// it returns the zero value of T and false.
func Find[T any](slice []T, fn func(T) bool) (T, bool) {
	if fn == nil {
		var zero T
		return zero, false
	}

	for _, v := range slice {
		if fn(v) {
			return v, true
		}
	}

	var zero T

	return zero, false
}

// FilterInPlace filters the provided slice in place without allocating new memory.
//
// It clears the unused trailing elements of the modified backing array using the
// built-in clear() function to allow the Garbage Collector to reclaim referenced memory.
//
// # Complexity
//
//   - Time: O(N)
//   - Space: O(1) auxiliary space (Zero allocations).
func FilterInPlace[T any](slice []T, fn func(T) bool) []T {
	if fn == nil {
		return slice
	}

	n := 0
	for _, v := range slice {
		if fn(v) {
			slice[n] = v
			n++
		}
	}

	clear(slice[n:])

	return slice[:n]
}

// Filter returns a new slice containing only elements that satisfy the predicate fn.
//
// If the input slice is nil or empty, Filter returns nil.
func Filter[T any](slice []T, fn func(T) bool) []T {
	if len(slice) == 0 || fn == nil {
		return nil
	}

	var res []T
	for _, v := range slice {
		if fn(v) {
			res = append(res, v)
		}
	}

	return res
}
