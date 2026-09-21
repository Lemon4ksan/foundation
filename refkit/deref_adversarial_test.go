// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package refkit_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/refkit"
)

type SampleStruct struct {
	ID   int
	Name string
}

func TestEmpirical_Refkit_Deref(t *testing.T) {
	// Nil pointers
	var nilInt *int
	if got := refkit.Deref(nilInt); got != 0 {
		t.Fatalf("expected 0 for nil *int, got %d", got)
	}

	var nilStr *string
	if got := refkit.Deref(nilStr); got != "" {
		t.Fatalf("expected empty string for nil *string, got %q", got)
	}

	var nilStruct *SampleStruct
	if got := refkit.Deref(nilStruct); got != (SampleStruct{}) {
		t.Fatalf("expected zero struct for nil *SampleStruct, got %+v", got)
	}

	var nilSlice *[]int
	if got := refkit.Deref(nilSlice); got != nil {
		t.Fatalf("expected nil slice for nil *[]int, got %v", got)
	}

	// Non-nil pointers
	val := 42
	if got := refkit.Deref(&val); got != 42 {
		t.Fatalf("expected 42, got %d", got)
	}

	strVal := "hello"
	if got := refkit.Deref(&strVal); got != "hello" {
		t.Fatalf("expected 'hello', got %q", got)
	}

	structVal := SampleStruct{ID: 1, Name: "test"}
	if got := refkit.Deref(&structVal); got != structVal {
		t.Fatalf("expected %+v, got %+v", structVal, got)
	}
}

func TestEmpirical_Refkit_DerefOr(t *testing.T) {
	// Nil pointer with fallback
	var nilInt *int
	if got := refkit.DerefOr(nilInt, 999); got != 999 {
		t.Fatalf("expected fallback 999, got %d", got)
	}

	var nilStr *string
	if got := refkit.DerefOr(nilStr, "default"); got != "default" {
		t.Fatalf("expected fallback 'default', got %q", got)
	}

	// Non-nil pointer with fallback
	val := 123
	if got := refkit.DerefOr(&val, 999); got != 123 {
		t.Fatalf("expected 123, got %d", got)
	}
}

func BenchmarkDeref_Primitive(b *testing.B) {
	val := 42
	ptr := &val

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_ = refkit.Deref(ptr)
	}
}

func BenchmarkDerefOr_Primitive(b *testing.B) {
	val := 42
	ptr := &val

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_ = refkit.DerefOr(ptr, 0)
	}
}
