// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package refkit_test

import (
	"reflect"
	"testing"

	"github.com/lemon4ksan/foundation/refkit"
	"github.com/lemon4ksan/foundation/testkit/assert"
	"github.com/lemon4ksan/foundation/testkit/require"
)

type SampleModel struct {
	Name  string   `json:"name,omitempty,default=anonymous,min=5,ratio=1.5,enum=a|b"`
	Tags  []string `json:"tags,comma"`
	Skip  int      `json:"-"`
	Inner *Inner   `json:"inner,inline"`
}

type Inner struct {
	ID int `json:"id"`
}

func TestDerefType_And_DerefValue(t *testing.T) {
	t.Parallel()

	// 1. DerefType
	var m ***SampleModel
	dt := refkit.DerefType(reflect.TypeOf(m))
	assert.Equal(t, reflect.TypeOf(SampleModel{}), dt)
	assert.Nil(t, refkit.DerefType(nil))

	// 2. DerefValue non-nil
	val := &SampleModel{Name: "alice"}
	p1 := &val
	p2 := &p1
	dv := refkit.DerefValue(reflect.ValueOf(p2))
	require.True(t, dv.IsValid())
	assert.Equal(t, "alice", dv.FieldByName("Name").String())

	// 3. DerefValue nil pointer in chain
	var nilPtr *SampleModel
	nilP1 := &nilPtr
	dvNil := refkit.DerefValue(reflect.ValueOf(nilP1))
	assert.False(t, dvNil.IsValid())

	// 4. IndirectType and IndirectValue
	assert.Equal(t, reflect.TypeOf(SampleModel{}), refkit.IndirectType(reflect.TypeOf(val)))
	assert.Equal(t, reflect.TypeOf(SampleModel{}), refkit.IndirectType(reflect.TypeOf(SampleModel{})))

	iv := refkit.IndirectValue(reflect.ValueOf(val))
	assert.Equal(t, "alice", iv.FieldByName("Name").String())

	ivPlain := refkit.IndirectValue(reflect.ValueOf(42))
	assert.Equal(t, 42, int(ivPlain.Int()))
}

func TestIndirectKind_And_TypeName(t *testing.T) {
	t.Parallel()

	val := &SampleModel{Name: "bob"}
	assert.Equal(t, reflect.Struct, refkit.IndirectKind(val))
	assert.Equal(t, reflect.Struct, refkit.IndirectKind(reflect.ValueOf(val)))
	assert.Equal(t, reflect.Struct, refkit.IndirectKind(reflect.TypeOf(val)))
	assert.Equal(t, reflect.Invalid, refkit.IndirectKind(nil))

	assert.Equal(t, "SampleModel", refkit.TypeName(val))
	assert.Equal(t, "SampleModel", refkit.TypeName(SampleModel{}))
	assert.Equal(t, "SampleModel", refkit.TypeName(reflect.TypeOf(val)))
	assert.Equal(t, "SampleModel", refkit.TypeName(reflect.ValueOf(val)))
	assert.Equal(t, "<nil>", refkit.TypeName(nil))
	assert.Equal(t, "<nil>", refkit.TypeName(reflect.Value{}))
	assert.Equal(t, "int", refkit.TypeName(42))

	// FullTypeName
	assert.Equal(t, "<nil>", refkit.FullTypeName(nil))
	assert.Equal(t, "<nil>", refkit.FullTypeName(reflect.Value{}))
	assert.Equal(t, "*refkit_test.SampleModel", refkit.FullTypeName(val))
	assert.Equal(t, "*refkit_test.SampleModel", refkit.FullTypeName(reflect.TypeOf(val)))
	assert.Equal(t, "*refkit_test.SampleModel", refkit.FullTypeName(reflect.ValueOf(val)))
	assert.Equal(t, "int", refkit.FullTypeName(100))
}

func TestCheck_Kinds_And_Numbers(t *testing.T) {
	t.Parallel()

	// 1. IsNumeric
	assert.True(t, refkit.IsNumeric(reflect.Int))
	assert.True(t, refkit.IsNumeric(reflect.Int64))
	assert.True(t, refkit.IsNumeric(reflect.Uint32))
	assert.True(t, refkit.IsNumeric(reflect.Float64))
	assert.False(t, refkit.IsNumeric(reflect.String))

	// 2. IsInteger
	assert.True(t, refkit.IsInteger(reflect.Int16))
	assert.True(t, refkit.IsInteger(reflect.Uint8))
	assert.False(t, refkit.IsInteger(reflect.Float32))

	// 3. IsFloat
	assert.True(t, refkit.IsFloat(reflect.Float32))
	assert.True(t, refkit.IsFloat(reflect.Float64))
	assert.False(t, refkit.IsFloat(reflect.Int))

	// 4. IsSigned & IsUnsigned
	assert.True(t, refkit.IsSigned(reflect.Int32))
	assert.False(t, refkit.IsSigned(reflect.Uint32))
	assert.True(t, refkit.IsUnsigned(reflect.Uint64))
	assert.False(t, refkit.IsUnsigned(reflect.Int64))

	// 5. IsNil & IsZero
	assert.True(t, refkit.IsNil(reflect.Value{}))
	assert.False(t, refkit.IsNil(reflect.ValueOf(42)))
	var nilPtr *int
	assert.True(t, refkit.IsNil(reflect.ValueOf(nilPtr)))

	assert.True(t, refkit.IsZero(reflect.Value{}))
	assert.True(t, refkit.IsZero(reflect.ValueOf(0)))
	assert.False(t, refkit.IsZero(reflect.ValueOf(5)))

	// 6. IsStruct & IsCollection
	assert.True(t, refkit.IsStruct(&SampleModel{}))
	assert.False(t, refkit.IsStruct(123))
	assert.True(t, refkit.IsCollection([]int{1, 2}))
	assert.True(t, refkit.IsCollection(map[string]int{"a": 1}))
	assert.False(t, refkit.IsCollection("str"))
}

func TestAlloc_New_And_EnsureAlloc(t *testing.T) {
	t.Parallel()

	// 1. New[T]
	ptr := refkit.New[SampleModel]()
	assert.NotNil(t, ptr)
	assert.Equal(t, "", ptr.Name)

	// 2. NewOf
	valOf := refkit.NewOf(reflect.TypeOf(SampleModel{}))
	assert.True(t, valOf.IsValid())
	assert.False(t, refkit.NewOf(nil).IsValid())

	// 3. EnsureAlloc on nil settable pointer
	var target *SampleModel
	targetVal := reflect.ValueOf(&target).Elem()
	elem, allocated := refkit.EnsureAlloc(targetVal)
	assert.True(t, allocated)
	assert.NotNil(t, target)
	assert.Equal(t, reflect.Struct, elem.Kind())

	// EnsureAlloc on already non-nil pointer
	elem2, allocated2 := refkit.EnsureAlloc(targetVal)
	assert.False(t, allocated2)
	assert.Equal(t, reflect.Struct, elem2.Kind())

	// EnsureAlloc on non-pointer
	nonPtr := 42
	elemNonPtr, allocNonPtr := refkit.EnsureAlloc(reflect.ValueOf(nonPtr))
	assert.False(t, allocNonPtr)
	assert.Equal(t, int64(42), elemNonPtr.Int())

	// EnsureAlloc on invalid Value
	_, allocInvalid := refkit.EnsureAlloc(reflect.Value{})
	assert.False(t, allocInvalid)
}

func TestTag_Parsing_And_Getters(t *testing.T) {
	t.Parallel()

	field, _ := reflect.TypeOf(SampleModel{}).FieldByName("Name")
	tag := refkit.GetTag(field, "json")

	assert.Equal(t, "name", tag.Name)
	assert.True(t, tag.Has("omitempty"))
	assert.False(t, tag.Has("nonexistent"))
	assert.False(t, tag.IsEmpty())
	assert.False(t, tag.IsIgnored())

	// Tag getters
	assert.Equal(t, "anonymous", tag.Get("default"))
	assert.Equal(t, "", tag.Get("missing"))

	minVal, ok := tag.GetInt("min")
	assert.True(t, ok)
	assert.Equal(t, 5, minVal)

	_, ok = tag.GetInt("missing")
	assert.False(t, ok)

	ratioVal, ok := tag.GetFloat("ratio")
	assert.True(t, ok)
	assert.Equal(t, 1.5, ratioVal)

	_, ok = tag.GetFloat("missing")
	assert.False(t, ok)

	// SplitOption
	enumParts := tag.SplitOption("enum", "|")
	assert.Equal(t, []string{"a", "b"}, enumParts)
	assert.Nil(t, tag.SplitOption("missing", "|"))

	// Ignored field `json:"-"`
	skipField, _ := reflect.TypeOf(SampleModel{}).FieldByName("Skip")
	skipTag := refkit.GetTag(skipField, "json")
	assert.True(t, skipTag.IsIgnored())

	// Empty tag
	emptyTag := refkit.ParseTag("")
	assert.True(t, emptyTag.IsEmpty())

	// Nonexistent tag lookup
	missingTag := refkit.GetTag(field, "xml")
	assert.True(t, missingTag.IsEmpty())
}
