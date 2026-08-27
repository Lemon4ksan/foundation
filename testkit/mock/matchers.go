// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mock

import (
	"fmt"
	"reflect"
)

// Matcher represents a call argument matcher.
type Matcher interface {
	Matches(x any) bool
	String() string
}

type anyMatcher struct{}

// Any returns a matcher matching anything.
func Any() Matcher                  { return anyMatcher{} }
func (anyMatcher) Matches(any) bool { return true }
func (anyMatcher) String() string   { return "is anything" }

type eqMatcher struct{ x any }

// Eq returns a matcher matching by deep equality.
func Eq(x any) Matcher                 { return eqMatcher{x: x} }
func (m eqMatcher) Matches(x any) bool { return reflect.DeepEqual(m.x, x) }
func (m eqMatcher) String() string     { return fmt.Sprintf("is equal to %v", m.x) }

type nilMatcher struct{}

// Nil returns a matcher matching nil pointers, slices, maps, or interfaces.
func Nil() Matcher { return nilMatcher{} }

func (nilMatcher) Matches(x any) bool {
	if x == nil {
		return true
	}

	v := reflect.ValueOf(x)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
func (nilMatcher) String() string { return "is nil" }

type notMatcher struct{ m Matcher }

// Not inverts the inner matcher.
func Not(m any) Matcher {
	if matcher, ok := m.(Matcher); ok {
		return notMatcher{m: matcher}
	}

	return notMatcher{m: Eq(m)}
}
func (n notMatcher) Matches(x any) bool { return !n.m.Matches(x) }
func (n notMatcher) String() string     { return fmt.Sprintf("not(%v)", n.m) }

type assignableToTypeOfMatcher struct{ target reflect.Type }

// AssignableToTypeOf matches if arg is assignable to the target type.
func AssignableToTypeOf(x any) Matcher {
	if t, ok := x.(reflect.Type); ok {
		return assignableToTypeOfMatcher{target: t}
	}

	return assignableToTypeOfMatcher{target: reflect.TypeOf(x)}
}

func (a assignableToTypeOfMatcher) Matches(x any) bool {
	if x == nil {
		return false
	}

	return reflect.TypeOf(x).AssignableTo(a.target)
}

func (a assignableToTypeOfMatcher) String() string {
	return fmt.Sprintf("is assignable to %v", a.target)
}

type lenMatcher struct{ n int }

// Len matches if arg has length n.
func Len(n int) Matcher { return lenMatcher{n: n} }

func (l lenMatcher) Matches(x any) bool {
	if x == nil {
		return false
	}

	v := reflect.ValueOf(x)
	switch v.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == l.n
	default:
		return false
	}
}
func (l lenMatcher) String() string { return fmt.Sprintf("has length %d", l.n) }
