// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package refkit

import (
	"reflect"
)

// EnsureAlloc ensures that a pointer value is allocated, initializing it with a new zero value if nil and settable.
// Returns the dereferenced element value and a boolean indicating whether a new instance was allocated.
func EnsureAlloc(v reflect.Value) (elem reflect.Value, allocated bool) {
	if !v.IsValid() {
		return reflect.Value{}, false
	}

	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			if v.CanSet() {
				target := reflect.New(v.Type().Elem())
				v.Set(target)

				return target.Elem(), true
			}

			return reflect.Value{}, false
		}

		return v.Elem(), false
	}

	if v.Kind() == reflect.Interface && !v.IsNil() {
		return EnsureAlloc(v.Elem())
	}

	return v, false
}

// NewOf creates a new pointer value allocating zero-initialized storage for type t.
func NewOf(t reflect.Type) reflect.Value {
	if t == nil {
		return reflect.Value{}
	}

	return reflect.New(t)
}

// New creates a new zero-initialized heap-allocated instance of T.
func New[T any]() *T {
	return new(T)
}
