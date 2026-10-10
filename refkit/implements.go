// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package refkit

import (
	"reflect"
)

// Implements reports whether type t implements interface T.
//
// Unlike the standard library [reflect.Type.Implements], this helper does not panic
// if T is a non-interface type or if t is nil; instead it safely returns false.
// It executes in O(1) time with 0 heap allocations and is safe for concurrent use.
func Implements[T any](t reflect.Type) bool {
	if t == nil {
		return false
	}

	target := reflect.TypeFor[T]()
	if target.Kind() != reflect.Interface {
		return false
	}

	return t.Implements(target)
}
