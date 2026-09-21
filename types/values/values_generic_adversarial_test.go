// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package values_test

import (
	"math"
	"strconv"
	"sync"
	"testing"

	"github.com/lemon4ksan/foundation/types/values"
)

type (
	CustomInt   int
	CustomFloat float64
)

func TestEmpirical_Values_NumberString(t *testing.T) {
	// Signed integers
	if got := values.NumberString(int(42)); got != "42" {
		t.Fatalf("expected 42, got %s", got)
	}
	if got := values.NumberString(int(-100)); got != "-100" {
		t.Fatalf("expected -100, got %s", got)
	}
	if got := values.NumberString(int8(math.MinInt8)); got != "-128" {
		t.Fatalf("expected -128, got %s", got)
	}
	if got := values.NumberString(int8(math.MaxInt8)); got != "127" {
		t.Fatalf("expected 127, got %s", got)
	}
	if got := values.NumberString(int16(math.MinInt16)); got != "-32768" {
		t.Fatalf("expected -32768, got %s", got)
	}
	if got := values.NumberString(int32(math.MaxInt32)); got != "2147483647" {
		t.Fatalf("expected 2147483647, got %s", got)
	}
	if got := values.NumberString(int64(math.MinInt64)); got != strconv.FormatInt(math.MinInt64, 10) {
		t.Fatalf("expected min int64, got %s", got)
	}

	// Unsigned integers
	if got := values.NumberString(uint(0)); got != "0" {
		t.Fatalf("expected 0, got %s", got)
	}
	if got := values.NumberString(uint8(255)); got != "255" {
		t.Fatalf("expected 255, got %s", got)
	}
	if got := values.NumberString(uint16(65535)); got != "65535" {
		t.Fatalf("expected 65535, got %s", got)
	}
	if got := values.NumberString(uint32(math.MaxUint32)); got != "4294967295" {
		t.Fatalf("expected max uint32, got %s", got)
	}
	if got := values.NumberString(uint64(math.MaxUint64)); got != "18446744073709551615" {
		t.Fatalf("expected max uint64, got %s", got)
	}
	if got := values.NumberString(uintptr(12345)); got != "12345" {
		t.Fatalf("expected 12345, got %s", got)
	}

	// Floating point
	if got := values.NumberString(float32(3.14)); got != "3.14" {
		t.Fatalf("expected 3.14, got %s", got)
	}
	if got := values.NumberString(float64(-123.456)); got != "-123.456" {
		t.Fatalf("expected -123.456, got %s", got)
	}

	// Custom constrained types
	if got := values.NumberString(CustomInt(99)); got != "99" {
		t.Fatalf("expected 99, got %s", got)
	}
}

func TestEmpirical_Values_NumberString_Concurrent(t *testing.T) {
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 100 {
				res := values.NumberString(i)
				if res != strconv.Itoa(i) {
					t.Errorf("expected %d, got %s", i, res)
				}
			}
		}()
	}
	wg.Wait()
}

func BenchmarkNumberString_Int(b *testing.B) {
	val := 42

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_ = values.NumberString(val)
	}
}

func BenchmarkNumberString_Float(b *testing.B) {
	val := 3.14159

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_ = values.NumberString(val)
	}
}
