// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package refkit_test

import (
	"reflect"
	"testing"

	"github.com/lemon4ksan/foundation/refkit"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

type FlatSimple struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	IsActive  bool   `json:"is_active"`
	privScore int
}

type BaseLevel struct {
	BaseID int
	Common string
}

type MidLevel struct {
	BaseLevel
	MidID    int
	DeepName string
}

type SiblingLevel struct {
	SiblingID int
	Other     string
}

type EmbeddedComplex struct {
	MidLevel
	SiblingLevel
	TopID    int
	Common   string // shadows BaseLevel.Common
	DeepName string // shadows MidLevel.DeepName
}

type LeftCollide struct {
	Conflict string
}

type RightCollide struct {
	Conflict string
}

type SameDepthConflict struct {
	LeftCollide
	RightCollide
	Normal string
}

type EmbeddedPointer struct {
	*BaseLevel
	Tag string
}

type SelfReferential struct {
	ID   int
	Next *SelfReferential
}

type EmptyStruct struct{}

func TestVisibleFields_FlatStruct(t *testing.T) {
	t.Parallel()

	typ := reflect.TypeFor[FlatSimple]()
	stdFields := reflect.VisibleFields(typ)

	var refkitFields []reflect.StructField
	for f := range refkit.VisibleFields(typ) {
		refkitFields = append(refkitFields, f)
	}

	require.Equal(t, len(stdFields), len(refkitFields))
	for i := range stdFields {
		assert.Equal(t, stdFields[i].Name, refkitFields[i].Name)
		assert.Equal(t, stdFields[i].Type, refkitFields[i].Type)
		assert.Equal(t, stdFields[i].Index, refkitFields[i].Index)
		assert.Equal(t, stdFields[i].Offset, refkitFields[i].Offset)
		assert.Equal(t, stdFields[i].Anonymous, refkitFields[i].Anonymous)
	}
}

func TestVisibleFields_EmbeddedStruct(t *testing.T) {
	t.Parallel()

	typ := reflect.TypeFor[EmbeddedComplex]()
	stdFields := reflect.VisibleFields(typ)

	var refkitFields []reflect.StructField
	for f := range refkit.VisibleFields(typ) {
		refkitFields = append(refkitFields, f)
	}

	require.Equal(t, len(stdFields), len(refkitFields))
	for i := range stdFields {
		assert.Equal(t, stdFields[i].Name, refkitFields[i].Name)
		assert.Equal(t, stdFields[i].Type, refkitFields[i].Type)
		assert.Equal(t, stdFields[i].Index, refkitFields[i].Index)
	}

	// Verify shadowing: TopID, Common, DeepName are at top level
	fieldMap := make(map[string]reflect.StructField)
	for _, f := range refkitFields {
		fieldMap[f.Name] = f
	}

	assert.Equal(t, 1, len(fieldMap["Common"].Index))
	assert.Equal(t, 1, len(fieldMap["DeepName"].Index))
}

func TestVisibleFields_SameDepthCollision(t *testing.T) {
	t.Parallel()

	typ := reflect.TypeFor[SameDepthConflict]()
	stdFields := reflect.VisibleFields(typ)

	var refkitFields []reflect.StructField
	for f := range refkit.VisibleFields(typ) {
		refkitFields = append(refkitFields, f)
	}

	require.Equal(t, len(stdFields), len(refkitFields))
	for i := range stdFields {
		assert.Equal(t, stdFields[i].Name, refkitFields[i].Name)
		assert.Equal(t, stdFields[i].Index, refkitFields[i].Index)
	}

	fieldMap := make(map[string]reflect.StructField)
	for _, f := range refkitFields {
		fieldMap[f.Name] = f
	}

	// Conflict exists in both LeftCollide and RightCollide at same depth, so it must be canceled out
	_, hasConflict := fieldMap["Conflict"]
	assert.False(t, hasConflict)
	_, hasNormal := fieldMap["Normal"]
	assert.True(t, hasNormal)
}

func TestVisibleFields_EmbeddedPointer(t *testing.T) {
	t.Parallel()

	typ := reflect.TypeFor[EmbeddedPointer]()
	stdFields := reflect.VisibleFields(typ)

	var refkitFields []reflect.StructField
	for f := range refkit.VisibleFields(typ) {
		refkitFields = append(refkitFields, f)
	}

	require.Equal(t, len(stdFields), len(refkitFields))
	for i := range stdFields {
		assert.Equal(t, stdFields[i].Name, refkitFields[i].Name)
		assert.Equal(t, stdFields[i].Index, refkitFields[i].Index)
	}
}

func TestVisibleFields_EarlyTermination(t *testing.T) {
	t.Parallel()

	typ := reflect.TypeFor[EmbeddedComplex]()

	count := 0
	for range refkit.VisibleFields(typ) {
		count++
		if count == 3 {
			break
		}
	}
	assert.Equal(t, 3, count)

	// Early termination on flat struct
	flatCount := 0
	for range refkit.VisibleFields(reflect.TypeFor[FlatSimple]()) {
		flatCount++
		if flatCount == 1 {
			break
		}
	}
	assert.Equal(t, 1, flatCount)
}

func TestVisibleFields_EdgeCases(t *testing.T) {
	t.Parallel()

	// 1. Nil type
	nilCount := 0
	for range refkit.VisibleFields(nil) {
		nilCount++
	}
	assert.Equal(t, 0, nilCount)

	// 2. Non-struct types
	nonStructs := []reflect.Type{
		reflect.TypeFor[int](),
		reflect.TypeFor[string](),
		reflect.TypeFor[[]byte](),
		reflect.TypeFor[map[string]int](),
		reflect.TypeFor[chan int](),
		reflect.TypeFor[func()](),
	}
	for _, nonStruct := range nonStructs {
		c := 0
		for range refkit.VisibleFields(nonStruct) {
			c++
		}
		assert.Equal(t, 0, c)
	}

	// 3. Pointer to struct
	ptrCount := 0
	for range refkit.VisibleFields(reflect.TypeFor[*FlatSimple]()) {
		ptrCount++
	}
	assert.Equal(t, 4, ptrCount)

	// 4. Empty struct
	emptyCount := 0
	for range refkit.VisibleFields(reflect.TypeFor[EmptyStruct]()) {
		emptyCount++
	}
	assert.Equal(t, 0, emptyCount)

	// 5. Recursive struct
	recurCount := 0
	for range refkit.VisibleFields(reflect.TypeFor[SelfReferential]()) {
		recurCount++
	}
	assert.Equal(t, 2, recurCount)
}

func TestVisibleFields_ZeroAlloc(t *testing.T) {
	flatTyp := reflect.TypeFor[FlatSimple]()
	var count int

	// Warmup
	for f := range refkit.VisibleFields(flatTyp) {
		if f.Name != "" {
			count++
		}
	}

	allocsFlat := testing.AllocsPerRun(1000, func() {
		for f := range refkit.VisibleFields(flatTyp) {
			if f.Name == "" {
				t.Fatal("empty name")
			}
			count++
		}
	})
	assert.Equal(t, float64(0), allocsFlat, "flat struct VisibleFields must achieve 0 allocs/op")

	// Embedded struct zero-alloc check
	embTyp := reflect.TypeFor[EmbeddedComplex]()
	for f := range refkit.VisibleFields(embTyp) {
		if f.Name != "" {
			count++
		}
	}

	allocsEmb := testing.AllocsPerRun(1000, func() {
		for f := range refkit.VisibleFields(embTyp) {
			if f.Name == "" {
				t.Fatal("empty name")
			}
			count++
		}
	})
	if isRaceEnabled {
		assert.True(
			t,
			allocsEmb <= 2,
			"embedded struct VisibleFields under -race must be <= 2 allocs/op",
		)
	} else {
		assert.Equal(t, float64(0), allocsEmb, "embedded struct VisibleFields must achieve 0 allocs/op")
	}
	_ = count
}

func BenchmarkVisibleFields_Refkit(b *testing.B) {
	typ := reflect.TypeFor[FlatSimple]()
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for f := range refkit.VisibleFields(typ) {
			_ = f.Name
		}
	}
}

func BenchmarkVisibleFields_Refkit_Embedded(b *testing.B) {
	typ := reflect.TypeFor[EmbeddedComplex]()
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for f := range refkit.VisibleFields(typ) {
			_ = f.Name
		}
	}
}

func BenchmarkVisibleFields_Stdlib(b *testing.B) {
	typ := reflect.TypeFor[FlatSimple]()
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		fields := reflect.VisibleFields(typ)
		for i := range fields {
			_ = fields[i].Name
		}
	}
}
