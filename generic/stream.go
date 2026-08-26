// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generic

import (
	"iter"
)

// Stream represents a fluent, zero-allocation sequence wrapper over standard Go [iter.Seq].
// It provides method-chainable functional transformations (such as Filter, Map, Take, Drop, and Distinct)
// that evaluate elements lazily on demand.
type Stream[T any] struct {
	seq iter.Seq[T]
}

// OfStream creates a [Stream] containing the provided variadic elements.
func OfStream[T any](items ...T) Stream[T] {
	return FromSlice(items)
}

// FromSlice creates a lazy [Stream] backed by a slice.
func FromSlice[T any](slice []T) Stream[T] {
	return Stream[T]{
		seq: func(yield func(T) bool) {
			for _, item := range slice {
				if !yield(item) {
					return
				}
			}
		},
	}
}

// FromSeq creates a [Stream] wrapping an existing [iter.Seq].
func FromSeq[T any](seq iter.Seq[T]) Stream[T] {
	return Stream[T]{seq: seq}
}

// Seq returns the underlying standard Go [iter.Seq] iterator.
func (s Stream[T]) Seq() iter.Seq[T] {
	return s.seq
}

// Filter retains only elements matching the predicate function.
func (s Stream[T]) Filter(predicate func(T) bool) Stream[T] {
	return Stream[T]{
		seq: FilterLazy(s.seq, predicate),
	}
}

// Map applies an in-place type transformation function to each element.
func (s Stream[T]) Map(fn func(T) T) Stream[T] {
	return Stream[T]{
		seq: MapLazy(s.seq, fn),
	}
}

// Take returns a stream containing at most the first n elements.
func (s Stream[T]) Take(n int) Stream[T] {
	return Stream[T]{
		seq: Take(s.seq, n),
	}
}

// Drop skips the first n elements.
func (s Stream[T]) Drop(n int) Stream[T] {
	return Stream[T]{
		seq: Drop(s.seq, n),
	}
}

// Distinct filters out duplicate elements based on a key extraction function.
func (s Stream[T]) Distinct(keyFn func(T) any) Stream[T] {
	return Stream[T]{
		seq: func(yield func(T) bool) {
			if s.seq == nil {
				return
			}
			seen := make(map[any]struct{})
			s.seq(func(v T) bool {
				key := keyFn(v)
				if _, exists := seen[key]; !exists {
					seen[key] = struct{}{}
					return yield(v)
				}
				return true
			})
		},
	}
}

// ForEach executes the given side-effect callback on each element.
func (s Stream[T]) ForEach(fn func(T)) {
	if s.seq == nil || fn == nil {
		return
	}
	s.seq(func(v T) bool {
		fn(v)
		return true
	})
}

// Collect evaluates the entire stream pipeline and collects elements into a newly allocated slice.
func (s Stream[T]) Collect() []T {
	if s.seq == nil {
		return nil
	}
	var res []T
	s.seq(func(v T) bool {
		res = append(res, v)
		return true
	})
	return res
}

// Count consumes the stream and returns the total number of elements.
func (s Stream[T]) Count() int {
	if s.seq == nil {
		return 0
	}
	var count int
	s.seq(func(T) bool {
		count++
		return true
	})
	return count
}

// First returns the first element of the stream and true, or zero-value and false if empty.
func (s Stream[T]) First() (T, bool) {
	var (
		res   T
		found bool
	)
	if s.seq == nil {
		return res, false
	}
	s.seq(func(v T) bool {
		res = v
		found = true
		return false
	})
	return res, found
}

// MapStream transforms elements from type F to type T lazily.
func MapStream[F, T any](s Stream[F], fn func(F) T) Stream[T] {
	return Stream[T]{
		seq: MapLazy(s.seq, fn),
	}
}

// ChunkStream batches elements into sub-slices of the specified size.
func ChunkStream[T any](s Stream[T], size int) Stream[[]T] {
	if size <= 0 {
		size = 1
	}

	return Stream[[]T]{
		seq: func(yield func([]T) bool) {
			if s.seq == nil {
				return
			}

			chunk := make([]T, 0, size)
			s.seq(func(v T) bool {
				chunk = append(chunk, v)
				if len(chunk) == size {
					if !yield(chunk) {
						return false
					}
					chunk = make([]T, 0, size)
				}
				return true
			})

			if len(chunk) > 0 {
				yield(chunk)
			}
		},
	}
}
