// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package refkit

import (
	"reflect"
)

// DerefType resolves the underlying non-pointer type, unwrapping all consecutive pointer layers (e.g. ***T -> T).
func DerefType(t reflect.Type) reflect.Type {
	if t == nil {
		return nil
	}

	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	return t
}

// DerefValue unwraps consecutive pointer and interface levels until reaching the concrete value.
// If any pointer along the chain is nil or the value is invalid, returns an invalid [reflect.Value].
func DerefValue(v reflect.Value) reflect.Value {
	for v.IsValid() && (v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface) {
		if v.Kind() == reflect.Pointer && v.IsNil() {
			return reflect.Value{}
		}

		v = v.Elem()
	}

	return v
}

// IndirectType dereferences a single pointer level if t is a pointer type.
func IndirectType(t reflect.Type) reflect.Type {
	if t != nil && t.Kind() == reflect.Pointer {
		return t.Elem()
	}

	return t
}

// IndirectValue dereferences a single pointer level if v is a non-nil pointer.
func IndirectValue(v reflect.Value) reflect.Value {
	if v.IsValid() && v.Kind() == reflect.Pointer && !v.IsNil() {
		return v.Elem()
	}

	return v
}

// IndirectKind returns the [reflect.Kind] of the dereferenced type or value.
func IndirectKind(v any) reflect.Kind {
	if v == nil {
		return reflect.Invalid
	}

	switch x := v.(type) {
	case reflect.Value:
		return DerefValue(x).Kind()
	case reflect.Type:
		dt := DerefType(x)
		if dt == nil {
			return reflect.Invalid
		}

		return dt.Kind()
	default:
		dt := DerefType(reflect.TypeOf(v))
		if dt == nil {
			return reflect.Invalid
		}

		return dt.Kind()
	}
}

// TypeName returns a clean, human-readable name of any value or type without pointer prefixes.
func TypeName(v any) string {
	if v == nil {
		return "<nil>"
	}

	var t reflect.Type

	switch x := v.(type) {
	case reflect.Type:
		t = x
	case reflect.Value:
		if !x.IsValid() {
			return "<nil>"
		}

		t = x.Type()
	default:
		t = reflect.TypeOf(v)
	}

	t = DerefType(t)
	if t == nil {
		return "<nil>"
	}

	if name := t.Name(); name != "" {
		return name
	}

	return t.String()
}

// FullTypeName returns the full reflect.Type string representation of any value or type.
func FullTypeName(v any) string {
	if v == nil {
		return "<nil>"
	}

	switch x := v.(type) {
	case reflect.Type:
		return x.String()
	case reflect.Value:
		if !x.IsValid() {
			return "<nil>"
		}

		return x.Type().String()
	default:
		return reflect.TypeOf(v).String()
	}
}
