// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generic

import "iter"

// MapLazy transforms a sequence lazily.
//
// Each element is mapped on-demand as the resulting iterator [iter.Seq] is consumed.
// It allocates zero intermediate slices regardless of the pipeline depth.
func MapLazy[F, T any](seq iter.Seq[F], fn func(F) T) iter.Seq[T] {
	return func(yield func(T) bool) {
		if seq == nil || fn == nil || yield == nil {
			return
		}

		seq(func(v F) bool {
			return yield(fn(v))
		})
	}
}

// FilterLazy filters a sequence lazily.
//
// Elements are evaluated against the predicate function fn on-the-fly,
// allocating zero intermediate slices.
func FilterLazy[T any](seq iter.Seq[T], fn func(T) bool) iter.Seq[T] {
	return func(yield func(T) bool) {
		if seq == nil || fn == nil || yield == nil {
			return
		}

		seq(func(v T) bool {
			if fn(v) {
				return yield(v)
			}

			return true // continue iteration
		})
	}
}

// Reduce folds a sequence into a single accumulated value of type T.
//
// It starts with the initial value and sequentially executes the accumulator
// function fn on each element.
//
// # Complexity
//
//   - Time: O(N)
//   - Space: O(1) auxiliary space.
func Reduce[F, T any](seq iter.Seq[F], initial T, fn func(T, F) T) T {
	if seq == nil || fn == nil {
		return initial
	}

	res := initial
	seq(func(v F) bool {
		res = fn(res, v)
		return true
	})

	return res
}

// Take returns a lazy sequence [iter.Seq] containing only the first n elements
// of the source sequence.
func Take[T any](seq iter.Seq[T], n int) iter.Seq[T] {
	return func(yield func(T) bool) {
		if seq == nil || yield == nil || n <= 0 {
			return
		}

		count := 0
		seq(func(v T) bool {
			if !yield(v) {
				return false
			}

			count++

			return count < n
		})
	}
}

// Drop returns a lazy sequence [iter.Seq] that skips the first n elements
// of the source sequence.
func Drop[T any](seq iter.Seq[T], n int) iter.Seq[T] {
	return func(yield func(T) bool) {
		if seq == nil || yield == nil {
			return
		}

		count := 0
		seq(func(v T) bool {
			if count < n {
				count++
				return true // skip and continue
			}

			return yield(v)
		})
	}
}

// ToSeq converts a standard flat slice into a lazy iterator sequence [iter.Seq].
func ToSeq[T any](slice []T) iter.Seq[T] {
	return func(yield func(T) bool) {
		if yield == nil {
			return
		}

		for _, v := range slice {
			if !yield(v) {
				return
			}
		}
	}
}

// ToSlice collects all elements from a lazy sequence [iter.Seq] and returns
// them as a flat slice, allocating memory exactly once.
func ToSlice[T any](seq iter.Seq[T]) []T {
	if seq == nil {
		return nil
	}

	var res []T
	seq(func(v T) bool {
		res = append(res, v)
		return true
	})

	return res
}
