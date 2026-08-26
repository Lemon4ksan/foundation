// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package refkit_test

import (
	"reflect"
	"testing"

	"github.com/lemon4ksan/foundation/testkit/assert"
	"github.com/lemon4ksan/foundation/testkit/require"

	"github.com/lemon4ksan/foundation/refkit"
)

type SampleModel struct {
	Name  string   `json:"name,omitempty" url:"name,omitempty"`
	Tags  []string `json:"tags,comma"     url:"tags,comma"`
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
	assert.Equal(t, "int", refkit.TypeName(42))
}

func TestCheck_IsNil_IsZero_IsStruct_IsCollection(t *testing.T) {
	t.Parallel()

	// IsNil panic safety
	assert.True(t, refkit.IsNil(reflect.Value{}))
	assert.False(t, refkit.IsNil(reflect.ValueOf(42)))
	assert.False(t, refkit.IsNil(reflect.ValueOf("test")))
	assert.False(t, refkit.IsNil(reflect.ValueOf(SampleModel{})))

	var nilSlice []string
	assert.True(t, refkit.IsNil(reflect.ValueOf(nilSlice)))
	var nilMap map[string]int
	assert.True(t, refkit.IsNil(reflect.ValueOf(nilMap)))
	var nilPtr *SampleModel
	assert.True(t, refkit.IsNil(reflect.ValueOf(nilPtr)))

	// IsZero
	assert.True(t, refkit.IsZero(reflect.Value{}))
	assert.True(t, refkit.IsZero(reflect.ValueOf(0)))
	assert.True(t, refkit.IsZero(reflect.ValueOf("")))
	assert.False(t, refkit.IsZero(reflect.ValueOf(1)))

	// IsStruct & IsCollection
	assert.True(t, refkit.IsStruct(&SampleModel{}))
	assert.True(t, refkit.IsStruct(SampleModel{}))
	assert.False(t, refkit.IsStruct([]int{1, 2}))

	assert.True(t, refkit.IsCollection([]string{"a"}))
	assert.True(t, refkit.IsCollection(map[string]int{"a": 1}))
	assert.False(t, refkit.IsCollection(SampleModel{}))
}

func TestAlloc_EnsureAlloc_And_NewOf(t *testing.T) {
	t.Parallel()

	type Container struct {
		Model *SampleModel
	}

	c := &Container{}
	fieldVal := reflect.ValueOf(c).Elem().FieldByName("Model")

	require.True(t, fieldVal.IsNil())
	elem, allocated := refkit.EnsureAlloc(fieldVal)
	assert.True(t, allocated)
	assert.True(t, elem.IsValid())
	assert.False(t, fieldVal.IsNil())

	// Calling again on already allocated pointer
	elem2, allocated2 := refkit.EnsureAlloc(fieldVal)
	assert.False(t, allocated2)
	assert.True(t, elem2.IsValid())

	// Non-pointer
	var x int = 10
	vx := reflect.ValueOf(x)
	elemX, allocX := refkit.EnsureAlloc(vx)
	assert.False(t, allocX)
	assert.Equal(t, int64(10), elemX.Int())

	// NewOf
	newVal := refkit.NewOf(reflect.TypeOf(SampleModel{}))
	assert.Equal(t, reflect.Pointer, newVal.Kind())
	assert.Equal(t, reflect.TypeOf(&SampleModel{}), newVal.Type())
}

func TestTag_Parsing(t *testing.T) {
	t.Parallel()

	st := reflect.TypeOf(SampleModel{})

	// 1. json:"name,omitempty"
	f0, _ := st.FieldByName("Name")
	tag0 := refkit.GetTag(f0, "url", "json")
	assert.Equal(t, "name", tag0.Name)
	assert.True(t, tag0.Has("omitempty"))
	assert.False(t, tag0.IsIgnored())

	// 2. json:"tags,comma"
	f1, _ := st.FieldByName("Tags")
	tag1 := refkit.GetTag(f1, "json")
	assert.Equal(t, "tags", tag1.Name)
	assert.True(t, tag1.Has("comma"))

	// 3. json:"-"
	f2, _ := st.FieldByName("Skip")
	tag2 := refkit.GetTag(f2, "json")
	assert.True(t, tag2.IsIgnored())

	// 4. ParseTag direct
	pTag := refkit.ParseTag("user_id,omitempty,inline,pk")
	assert.Equal(t, "user_id", pTag.Name)
	assert.True(t, pTag.Has("omitempty"))
	assert.True(t, pTag.Has("inline"))
	assert.True(t, pTag.Has("pk"))
	assert.False(t, pTag.Has("non_existent"))

	emptyTag := refkit.ParseTag("")
	assert.Empty(t, emptyTag.Name)
	assert.Empty(t, emptyTag.Options)
}

func BenchmarkRefkit_Deref(b *testing.B) {
	m := &SampleModel{Name: "bench"}
	p1 := &m
	p2 := &p1
	v := reflect.ValueOf(p2)

	b.ReportAllocs()
	for b.Loop() {
		_ = refkit.DerefValue(v)
	}
}

func BenchmarkRefkit_IsNil(b *testing.B) {
	var nilPtr *SampleModel
	v := reflect.ValueOf(nilPtr)

	b.ReportAllocs()
	for b.Loop() {
		_ = refkit.IsNil(v)
	}
}
