// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generic

// Number represents any integer or floating-point type.
type Number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64
}

// Range represents a numeric interval from Start (inclusive) to End (exclusive).
type Range[T Number] struct {
	Start T
	End   T
}

// NewRange creates a new [Range] from start to end.
func NewRange[T Number](start, end T) Range[T] {
	return Range[T]{Start: start, End: end}
}

// Contains returns true if the value is within [Start, End).
func (r Range[T]) Contains(val T) bool {
	return val >= r.Start && val < r.End
}

// Stride iterates from Start to End with a given positive step size, executing fn for each value.
// If fn returns false, iteration terminates early.
func (r Range[T]) Stride(step T, fn func(val T) bool) {
	if step <= 0 {
		return
	}

	for i := r.Start; i < r.End; i += step {
		if !fn(i) {
			break
		}
	}
}

// ToSlice converts the Range into a slice of values stepping by 1.
// Note: Only makes sense if Step is effectively 1 (typically for integers).
func (r Range[T]) ToSlice() []T {
	var res []T
	for i := r.Start; i < r.End; i++ {
		res = append(res, i)
	}

	return res
}
