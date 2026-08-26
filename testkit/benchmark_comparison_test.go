// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package testkit_test

import (
	"bytes"
	"testing"

	"github.com/lemon4ksan/foundation/testkit/assert"
)

type nopTB struct {
	testing.TB
}

func (n nopTB) Helper()                           {}
func (n nopTB) Errorf(format string, args ...any) {}
func (n nopTB) FailNow()                          {}

type sampleStruct struct {
	ID    int
	Name  string
	Tags  []string
	Flags map[string]bool
}

var (
	tb nopTB

	sampleA = sampleStruct{
		ID:    101,
		Name:  "TestUser",
		Tags:  []string{"admin", "ops", "audit"},
		Flags: map[string]bool{"active": true, "verified": true},
	}
	sampleB = sampleStruct{
		ID:    101,
		Name:  "TestUser",
		Tags:  []string{"admin", "ops", "audit"},
		Flags: map[string]bool{"active": true, "verified": true},
	}

	byteSliceA = bytes.Repeat([]byte("HTTP/2 framing and zero-allocation networking engine"), 10)
	byteSliceB = bytes.Repeat([]byte("HTTP/2 framing and zero-allocation networking engine"), 10)

	intSlice = []int{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
)

func BenchmarkAssert_NoError_Testkit(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		assert.NoError(tb, nil)
	}
}

func BenchmarkAssert_Equal_Int_Testkit(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		assert.Equal(tb, 42, 42)
	}
}

func BenchmarkAssert_Equal_Bytes_Testkit(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		assert.Equal(tb, byteSliceA, byteSliceB)
	}
}

func BenchmarkAssert_Equal_Struct_Testkit(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		assert.Equal(tb, sampleA, sampleB)
	}
}

func BenchmarkAssert_Contains_Slice_Testkit(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		assert.Contains(tb, intSlice, 80)
	}
}
