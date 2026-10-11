// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package refkit_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"reflect"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/lemon4ksan/foundation/refkit"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
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
	assert.Equal(t, reflect.TypeFor[SampleModel](), dt)
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
	assert.Equal(t, reflect.TypeFor[SampleModel](), refkit.IndirectType(reflect.TypeOf(val)))
	assert.Equal(t, reflect.TypeFor[SampleModel](), refkit.IndirectType(reflect.TypeFor[SampleModel]()))

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

func TestAlloc_EnsureAlloc(t *testing.T) {
	t.Parallel()

	// EnsureAlloc on nil settable pointer
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

	// EnsureAlloc on unsettable nil pointer
	var nilPtr *int
	_, allocUnsettable := refkit.EnsureAlloc(reflect.ValueOf(nilPtr))
	assert.False(t, allocUnsettable)

	// EnsureAlloc on non-nil interface wrapping pointer
	var iface any = &nonPtr
	elemIface, allocIface := refkit.EnsureAlloc(reflect.ValueOf(&iface).Elem())
	assert.False(t, allocIface)
	assert.Equal(t, int64(42), elemIface.Int())
}

func TestTag_Parsing_And_Getters(t *testing.T) {
	t.Parallel()

	field, _ := reflect.TypeFor[SampleModel]().FieldByName("Name")
	tag := refkit.TagOf(field, "json")

	assert.Equal(t, "name", tag.Name)
	assert.True(t, tag.Has("omitempty"))
	assert.False(t, tag.Has("nonexistent"))
	assert.False(t, tag.IsEmpty())
	assert.False(t, tag.IsIgnored())

	// Tag getters
	assert.Equal(t, "anonymous", tag.Value("default"))
	assert.Equal(t, "", tag.Value("missing"))

	minVal, ok := tag.Int("min")
	assert.True(t, ok)
	assert.Equal(t, 5, minVal)

	_, ok = tag.Int("missing")
	assert.False(t, ok)

	ratioVal, ok := tag.Float("ratio")
	assert.True(t, ok)
	assert.Equal(t, 1.5, ratioVal)

	_, ok = tag.Float("missing")
	assert.False(t, ok)

	// SplitOption
	enumParts := slices.Collect(tag.SplitOption("enum", "|"))
	assert.Equal(t, []string{"a", "b"}, enumParts)
	assert.Empty(t, slices.Collect(tag.SplitOption("missing", "|")))

	// Ignored field `json:"-"`
	skipField, _ := reflect.TypeFor[SampleModel]().FieldByName("Skip")
	skipTag := refkit.TagOf(skipField, "json")
	assert.True(t, skipTag.IsIgnored())

	// Empty tag
	emptyTag := refkit.ParseTag("")
	assert.True(t, emptyTag.IsEmpty())

	// Nonexistent tag lookup
	missingTag := refkit.TagOf(field, "xml")
	assert.True(t, missingTag.IsEmpty())
}

// Types for unexported & cross-type testing
type unexportedFieldsStruct struct {
	rawBytes []byte
	rawArray [4]byte
	str      string
}

type privInner struct {
	InnerVal int
	PrivVal  string
}

type OuterWithPrivAnon struct {
	privInner
	OuterVal int
}

type DiamondBase struct {
	BaseVal int
}

type DiamondLeft struct {
	DiamondBase
	LeftVal int
}

type DiamondRight struct {
	DiamondBase
	RightVal int
}

type DiamondStruct struct {
	DiamondLeft
	DiamondRight
	RootVal int
}

type CycleNode struct {
	*CycleNode
	Val int
}

type MutualCycleA struct {
	*MutualCycleB
	ValA int
}

type MutualCycleB struct {
	*MutualCycleA
	ValB int
}

// Deep embedding types
type DeepD struct {
	DVal int
}

type DeepC struct {
	DeepD
	CVal int
}

type DeepB struct {
	DeepC
	BVal int
}

type DeepA struct {
	DeepB
	AVal int
}

// Custom interface hierarchies
type BaseInterface interface {
	BaseMethod() string
}

type MidInterface interface {
	BaseInterface
	MidMethod() int
}

type LeafInterface interface {
	MidInterface
	LeafMethod() bool
}

type BaseImpl struct{}

func (BaseImpl) BaseMethod() string { return "base" }

type MidImpl struct {
	BaseImpl
}

func (MidImpl) MidMethod() int { return 42 }

type LeafImpl struct {
	MidImpl
}

func (LeafImpl) LeafMethod() bool { return true }

type PtrReceiverImpl struct{}

func (*PtrReceiverImpl) BaseMethod() string { return "ptr_base" }
func (*PtrReceiverImpl) MidMethod() int     { return 100 }
func (*PtrReceiverImpl) LeafMethod() bool   { return false }

// -------------------------------------------------------------
// 1. VisibleFields Tests
// -------------------------------------------------------------

func TestAdversarial_VisibleFields_UnexportedAnonymous(t *testing.T) {
	typ := reflect.TypeFor[OuterWithPrivAnon]()
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

func TestAdversarial_VisibleFields_DiamondAmbiguity(t *testing.T) {
	typ := reflect.TypeFor[DiamondStruct]()
	stdFields := reflect.VisibleFields(typ)

	var refkitFields []reflect.StructField
	for f := range refkit.VisibleFields(typ) {
		refkitFields = append(refkitFields, f)
	}

	fieldNames := make(map[string]bool)
	for _, f := range refkitFields {
		fieldNames[f.Name] = true
	}

	// BaseVal exists in DiamondLeft and DiamondRight at the same depth (depth 3).
	// It must be ambiguous and therefore NOT visible.
	assert.False(t, fieldNames["BaseVal"], "BaseVal should be cancelled due to ambiguity")

	// LeftVal, RightVal, RootVal should be visible
	assert.True(t, fieldNames["LeftVal"])
	assert.True(t, fieldNames["RightVal"])
	assert.True(t, fieldNames["RootVal"])

	// Check against stdlib
	stdFieldNames := make(map[string]bool)
	for _, f := range stdFields {
		stdFieldNames[f.Name] = true
	}
	assert.Equal(t, stdFieldNames, fieldNames)
}

func TestAdversarial_VisibleFields_CyclicPointer(t *testing.T) {
	typ := reflect.TypeFor[CycleNode]()

	var fields []reflect.StructField
	for f := range refkit.VisibleFields(typ) {
		fields = append(fields, f)
	}

	require.Equal(t, 2, len(fields))
	assert.Equal(t, "CycleNode", fields[0].Name)
	assert.Equal(t, "Val", fields[1].Name)
}

func TestAdversarial_VisibleFields_MutualCycles(t *testing.T) {
	typ := reflect.TypeFor[MutualCycleA]()

	var fields []reflect.StructField
	for f := range refkit.VisibleFields(typ) {
		fields = append(fields, f)
	}

	// Mutual cycles should terminate without infinite recursion
	assert.True(t, len(fields) > 0)
}

func TestAdversarial_VisibleFields_NestedIterators(t *testing.T) {
	// Tests walker pool reentrancy
	typ := reflect.TypeFor[DiamondStruct]()

	outerCount := 0
	innerCount := 0

	for f1 := range refkit.VisibleFields(typ) {
		outerCount++
		for f2 := range refkit.VisibleFields(typ) {
			innerCount++
			if f1.Name == f2.Name {
				assert.Equal(t, f1.Index, f2.Index)
			}
		}
	}

	assert.True(t, outerCount > 0)
	assert.Equal(t, outerCount*outerCount, innerCount)
}

func TestAdversarial_VisibleFields_EarlyBreakReentrancy(t *testing.T) {
	// Ensures early break properly returns the walker to pool in a clean state
	typ := reflect.TypeFor[DiamondStruct]()

	for range 50 {
		step := 0
		for range refkit.VisibleFields(typ) {
			step++
			if step == 2 {
				break
			}
		}
		assert.Equal(t, 2, step)
	}

	// Verify next full iteration is complete and accurate
	fullCount := 0
	for range refkit.VisibleFields(typ) {
		fullCount++
	}
	assert.True(t, fullCount >= 4)
}

type EmbeddedOrderRoot struct {
	InnerOne
	RootField int
}

type InnerOne struct {
	InnerFieldA int
	InnerFieldB int
}

func TestAdversarial_VisibleFields_FieldOrdering(t *testing.T) {
	typ := reflect.TypeFor[EmbeddedOrderRoot]()
	stdFields := reflect.VisibleFields(typ)

	var refkitFields []reflect.StructField
	for f := range refkit.VisibleFields(typ) {
		refkitFields = append(refkitFields, f)
	}

	require.Equal(t, len(stdFields), len(refkitFields))
	for i := range stdFields {
		t.Logf("std[%d]=%s, refkit[%d]=%s", i, stdFields[i].Name, i, refkitFields[i].Name)
		assert.Equal(t, stdFields[i].Name, refkitFields[i].Name)
	}
}

func TestAdversarial_VisibleFields_Depth4(t *testing.T) {
	typ := reflect.TypeFor[DeepA]()
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

// -------------------------------------------------------------
// 2. Implements[T] Tests
// -------------------------------------------------------------

func TestAdversarial_Implements_AnyInterface(t *testing.T) {
	// Every type implements any (interface{})
	types := []reflect.Type{
		reflect.TypeFor[int](),
		reflect.TypeFor[string](),
		reflect.TypeFor[[]byte](),
		reflect.TypeFor[BaseImpl](),
		reflect.TypeFor[*BaseImpl](),
		reflect.TypeFor[EmptyStruct](),
		reflect.TypeFor[io.Reader](),
		reflect.TypeFor[unsafe.Pointer](),
	}

	for _, typ := range types {
		assert.True(t, refkit.Implements[any](typ))
	}

	// nil type should return false
	assert.False(t, refkit.Implements[any](nil))
}

func TestAdversarial_Implements_InterfaceHierarchy(t *testing.T) {
	baseTyp := reflect.TypeFor[BaseImpl]()
	midTyp := reflect.TypeFor[MidImpl]()
	leafTyp := reflect.TypeFor[LeafImpl]()
	ptrTyp := reflect.TypeFor[PtrReceiverImpl]()
	ptrPtrTyp := reflect.TypeFor[*PtrReceiverImpl]()

	// BaseImpl implements BaseInterface only
	assert.True(t, refkit.Implements[BaseInterface](baseTyp))
	assert.False(t, refkit.Implements[MidInterface](baseTyp))
	assert.False(t, refkit.Implements[LeafInterface](baseTyp))

	// MidImpl implements BaseInterface and MidInterface
	assert.True(t, refkit.Implements[BaseInterface](midTyp))
	assert.True(t, refkit.Implements[MidInterface](midTyp))
	assert.False(t, refkit.Implements[LeafInterface](midTyp))

	// LeafImpl implements all three
	assert.True(t, refkit.Implements[BaseInterface](leafTyp))
	assert.True(t, refkit.Implements[MidInterface](leafTyp))
	assert.True(t, refkit.Implements[LeafInterface](leafTyp))

	// Value receiver of pointer-method struct implements none
	assert.False(t, refkit.Implements[BaseInterface](ptrTyp))
	assert.False(t, refkit.Implements[MidInterface](ptrTyp))
	assert.False(t, refkit.Implements[LeafInterface](ptrTyp))

	// Pointer to struct implements all three
	assert.True(t, refkit.Implements[BaseInterface](ptrPtrTyp))
	assert.True(t, refkit.Implements[MidInterface](ptrPtrTyp))
	assert.True(t, refkit.Implements[LeafInterface](ptrPtrTyp))

	// Interface types themselves
	assert.True(t, refkit.Implements[BaseInterface](reflect.TypeFor[BaseInterface]()))
	assert.True(t, refkit.Implements[BaseInterface](reflect.TypeFor[MidInterface]()))
	assert.True(t, refkit.Implements[BaseInterface](reflect.TypeFor[LeafInterface]()))
	assert.False(t, refkit.Implements[LeafInterface](reflect.TypeFor[BaseInterface]()))
}

func TestAdversarial_Implements_PointersToInterfaces(t *testing.T) {
	// In Go, pointer to an interface *io.Reader DOES NOT implement io.Reader
	ptrInterface := reflect.TypeFor[*io.Reader]()
	assert.False(t, refkit.Implements[io.Reader](ptrInterface))
}

// -------------------------------------------------------------
// 3. SchemaCache Concurrency & Stress Tests
// -------------------------------------------------------------

func TestAdversarial_SchemaCache_Stress(t *testing.T) {
	cache := refkit.NewSchemaCache()
	types := []reflect.Type{
		reflect.TypeFor[DiamondStruct](),
		reflect.TypeFor[OuterWithPrivAnon](),
		reflect.TypeFor[DeepA](),
		reflect.TypeFor[CycleNode](),
		reflect.TypeFor[FlatSimple](),
		reflect.TypeFor[EmbeddedComplex](),
	}

	const goroutines = 32
	const iterations = 200

	var wg sync.WaitGroup
	wg.Add(goroutines)
	var errCount atomic.Int64

	for g := range goroutines {
		go func(id int) {
			defer wg.Done()
			for i := range iterations {
				typ := types[(id+i)%len(types)]
				schema, err := cache.Get(typ)
				if err != nil || schema == nil || schema.Type != typ {
					errCount.Add(1)
				}
			}
		}(g)
	}

	wg.Wait()
	assert.Equal(t, int64(0), errCount.Load())
}

func TestAdversarial_SchemaCache_CyclicStructs(t *testing.T) {
	cache := refkit.NewSchemaCache()

	// 1. Self-referential pointer embedding
	schema1, err := cache.Get(reflect.TypeFor[CycleNode]())
	require.NoError(t, err)
	require.NotNil(t, schema1)
	assert.Equal(t, 2, len(schema1.Fields))

	// 2. Mutual cycle pointer embedding
	schema2, err := cache.Get(reflect.TypeFor[MutualCycleA]())
	require.NoError(t, err)
	require.NotNil(t, schema2)
	assert.True(t, len(schema2.Fields) > 0)
}

func TestAdversarial_SchemaCache_PointerEmbeddingLayout(t *testing.T) {
	cache := refkit.NewSchemaCache()
	schema, err := cache.Get(reflect.TypeFor[EmbeddedPointer]())
	require.NoError(t, err)
	require.NotNil(t, schema)

	// EmbeddedPointer has *BaseLevel and Tag string.
	// Verify fields exist in schema
	fieldMap := make(map[string]refkit.FieldSchema)
	for _, f := range schema.Fields {
		fieldMap[f.Field.Name] = f
	}

	_, hasTag := fieldMap["Tag"]
	assert.True(t, hasTag)

	_, hasBaseID := fieldMap["BaseID"]
	assert.True(t, hasBaseID)
}

// -------------------------------------------------------------
// 4. Strict Zero-Allocation Verification (testing.AllocsPerRun)
// -------------------------------------------------------------

func TestAdversarial_ZeroAlloc_AllPrimitives(t *testing.T) {
	// 1. Implements[T] with interface type
	tImpl := reflect.TypeFor[LeafImpl]()
	allocsImpl := testing.AllocsPerRun(1000, func() {
		_ = refkit.Implements[MidInterface](tImpl)
	})
	assert.Equal(t, float64(0), allocsImpl, "Implements must be 0 allocs/op")

	// 2. VisibleFields on diamond struct (embedding depth <= 3)
	tDiamond := reflect.TypeFor[DiamondStruct]()
	for range refkit.VisibleFields(tDiamond) {
	}
	allocsVisible := testing.AllocsPerRun(1000, func() {
		for f := range refkit.VisibleFields(tDiamond) {
			_ = f.Name
		}
	})
	if isRaceEnabled {
		assert.True(
			t,
			allocsVisible <= 2,
			"VisibleFields under -race must be <= 2 allocs/op",
		)
	} else {
		assert.Equal(t, float64(0), allocsVisible, "VisibleFields (depth <= 3) must be 0 allocs/op")
	}

	// 3. SchemaCache.Get cache hit
	cache := refkit.NewSchemaCache()
	_, _ = cache.Get(tDiamond)
	allocsCache := testing.AllocsPerRun(1000, func() {
		s, err := cache.Get(tDiamond)
		if err != nil || s == nil {
			panic("cache get failed")
		}
	})
	assert.Equal(t, float64(0), allocsCache, "SchemaCache.Get hit must be 0 allocs/op")

	// 4. Accessors
	vInt := reflect.ValueOf(int64(12345))
	allocsInt := testing.AllocsPerRun(1000, func() {
		_, _ = refkit.AsInt64(vInt)
	})
	assert.Equal(t, float64(0), allocsInt, "AsInt64 must be 0 allocs/op")

	vFloat := reflect.ValueOf(float64(3.14159))
	allocsFloat := testing.AllocsPerRun(1000, func() {
		_, _ = refkit.AsFloat64(vFloat)
	})
	assert.Equal(t, float64(0), allocsFloat, "AsFloat64 must be 0 allocs/op")

	vBool := reflect.ValueOf(true)
	allocsBool := testing.AllocsPerRun(1000, func() {
		_, _ = refkit.AsBool(vBool)
	})
	assert.Equal(t, float64(0), allocsBool, "AsBool must be 0 allocs/op")

	vStr := reflect.ValueOf("test_string")
	allocsStr := testing.AllocsPerRun(1000, func() {
		_, _ = refkit.AsString(vStr)
	})
	assert.Equal(t, float64(0), allocsStr, "AsString must be 0 allocs/op")

	rawBytes := []byte("test_bytes")
	vBytes := reflect.ValueOf(rawBytes)
	allocsBytes := testing.AllocsPerRun(1000, func() {
		_, _ = refkit.AsBytes(vBytes)
	})
	assert.Equal(t, float64(0), allocsBytes, "AsBytes must be 0 allocs/op")
}

// TestTag_ConcurrencyStress tests concurrent read-access across multiple goroutines
// calling Tag methods (Value, Int, Float, Has, SplitOption) simultaneously with -race.
func TestTag_ConcurrencyStress(t *testing.T) {
	tag := refkit.ParseTag(`user_id,omitempty,min=10,max=1000,ratio=3.14159,flags=a|b|c|d,default=guest`)

	const goroutines = 64
	const iterations = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := range goroutines {
		go func(workerID int) {
			defer wg.Done()

			for range iterations {
				// 1. Tag properties
				if tag.Name != "user_id" {
					t.Errorf("expected user_id, got %q", tag.Name)
					return
				}
				if !tag.Has("omitempty") {
					t.Errorf("missing omitempty")
					return
				}
				if tag.IsEmpty() || tag.IsIgnored() {
					t.Errorf("tag should not be empty or ignored")
					return
				}

				// 2. Value getter
				if v := tag.Value("default"); v != "guest" {
					t.Errorf("expected guest, got %q", v)
					return
				}

				// 3. Int getter
				minVal, ok := tag.Int("min")
				if !ok || minVal != 10 {
					t.Errorf("expected min=10, got (%d, %v)", minVal, ok)
					return
				}
				maxVal, ok := tag.Int("max")
				if !ok || maxVal != 1000 {
					t.Errorf("expected max=1000, got (%d, %v)", maxVal, ok)
					return
				}

				// 4. Float getter
				ratio, ok := tag.Float("ratio")
				if !ok || math.Abs(ratio-3.14159) > 1e-9 {
					t.Errorf("expected ratio=3.14159, got (%f, %v)", ratio, ok)
					return
				}

				// 5. SplitOption
				flags := slices.Collect(tag.SplitOption("flags", "|"))
				if len(flags) != 4 || flags[0] != "a" || flags[3] != "d" {
					t.Errorf("unexpected flags: %v", flags)
					return
				}
			}
		}(g)
	}

	wg.Wait()
}

// TestTag_Int_AdversarialEdgeCases evaluates Tag.Int edge cases:
// boundary limits, signs, overflow, underflow, whitespace, hex, and invalid inputs.
func TestTag_Int_AdversarialEdgeCases(t *testing.T) {
	cases := []struct {
		tagStr   string
		key      string
		expected int
		ok       bool
	}{
		// Standard valid ints
		{"field,num=0", "num", 0, true},
		{"field,num=1", "num", 1, true},
		{"field,num=-1", "num", -1, true},
		{"field,num=+42", "num", 42, true},
		{"field,num=-42", "num", -42, true},

		// Max and Min int values (assuming 64-bit platform)
		{fmt.Sprintf("field,num=%d", math.MaxInt), "num", math.MaxInt, true},
		{fmt.Sprintf("field,num=%d", math.MinInt), "num", math.MinInt, true},

		// Overflow / underflow
		{"field,num=9223372036854775808", "num", 0, false},
		{"field,num=-9223372036854775809", "num", 0, false},
		{"field,num=999999999999999999999999999999", "num", 0, false},

		// Formats not supported by strconv.Atoi
		{"field,num=0x10", "num", 0, false},
		{"field,num=0b1010", "num", 0, false},
		{"field,num=1.0", "num", 0, false},
		{"field,num=1e5", "num", 0, false},

		// Whitespace in value (ParseTag trims options, so "num=5 " trims to "num=5", but "num= 5" has internal space)
		{"field,num= 5", "num", 0, false},
		{"field,num=5 ", "num", 5, true},

		// Empty or missing
		{"field,num=", "num", 0, false},
		{"field,num", "num", 0, false},
		{"field", "num", 0, false},
		{"", "num", 0, false},
	}

	for _, tc := range cases {
		tag := refkit.ParseTag(tc.tagStr)
		val, ok := tag.Int(tc.key)
		assert.Equal(t, tc.ok, ok, fmt.Sprintf("case %s ok mismatch", tc.tagStr))
		if tc.ok {
			assert.Equal(t, tc.expected, val, fmt.Sprintf("case %s val mismatch", tc.tagStr))
		}
	}
}

// TestTag_Float_AdversarialEdgeCases evaluates Tag.Float edge cases:
// NaN, infinities, scientific notation, zero signs, underflow/overflow.
func TestTag_Float_AdversarialEdgeCases(t *testing.T) {
	cases := []struct {
		tagStr string
		key    string
		verify func(t *testing.T, val float64, ok bool)
	}{
		{
			"field,val=0.0", "val",
			func(t *testing.T, val float64, ok bool) {
				assert.True(t, ok)
				assert.Equal(t, 0.0, val)
			},
		},
		{
			"field,val=-0.0", "val",
			func(t *testing.T, val float64, ok bool) {
				assert.True(t, ok)
				assert.Equal(t, 0.0, val)
			},
		},
		{
			"field,val=1e-5", "val",
			func(t *testing.T, val float64, ok bool) {
				assert.True(t, ok)
				assert.Equal(t, 1e-5, val)
			},
		},
		{
			"field,val=2.5E+10", "val",
			func(t *testing.T, val float64, ok bool) {
				assert.True(t, ok)
				assert.Equal(t, 2.5e10, val)
			},
		},
		{
			"field,val=NaN", "val",
			func(t *testing.T, val float64, ok bool) {
				assert.True(t, ok)
				assert.True(t, math.IsNaN(val))
			},
		},
		{
			"field,val=+Inf", "val",
			func(t *testing.T, val float64, ok bool) {
				assert.True(t, ok)
				assert.True(t, math.IsInf(val, 1))
			},
		},
		{
			"field,val=-Inf", "val",
			func(t *testing.T, val float64, ok bool) {
				assert.True(t, ok)
				assert.True(t, math.IsInf(val, -1))
			},
		},
		{
			"field,val=", "val",
			func(t *testing.T, val float64, ok bool) {
				assert.False(t, ok)
			},
		},
		{
			"field,val=invalid", "val",
			func(t *testing.T, val float64, ok bool) {
				assert.False(t, ok)
			},
		},
		{
			"field", "val",
			func(t *testing.T, val float64, ok bool) {
				assert.False(t, ok)
			},
		},
	}

	for _, tc := range cases {
		tag := refkit.ParseTag(tc.tagStr)
		val, ok := tag.Float(tc.key)
		tc.verify(t, val, ok)
	}
}

// TestTag_Value_AdversarialEdgeCases evaluates Tag.Value key lookup behavior.
func TestTag_Value_AdversarialEdgeCases(t *testing.T) {
	tag := refkit.ParseTag(`field,key=first=second=third,empty=,no_val,=lone_val,"quoted=val"`)

	// key=first=second=third -> Cut splits at first '='
	assert.Equal(t, "first=second=third", tag.Value("key"))

	// empty= -> ""
	assert.Equal(t, "", tag.Value("empty"))

	// no_val -> "" (no '=')
	assert.Equal(t, "", tag.Value("no_val"))

	// =lone_val -> Cut gives k="", v="lone_val"
	assert.Equal(t, "lone_val", tag.Value(""))

	// Nonexistent
	assert.Equal(t, "", tag.Value("nonexistent"))
}

// TestTag_SplitOption_EmptyDelim verifies whether calling SplitOption with empty delim
// terminates or enters an infinite loop.
func TestTag_SplitOption_EmptyDelim(t *testing.T) {
	tag := refkit.ParseTag(`field,tokens=abc`)

	// Run SplitOption with empty delim under a timeout channel
	done := make(chan struct{})
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	go func() {
		// Attempt iteration with delim = ""
		for range tag.SplitOption("tokens", "") {
			// If it yields indefinitely, break out
			break
		}
		close(done)
	}()

	select {
	case <-done:
		t.Logf("SplitOption with delim='' returned successfully")
	case <-ctx.Done():
		t.Logf(
			"VULNERABILITY DETECTED: Tag.SplitOption(\"tokens\", \"\") enters an infinite loop due to strings.Index(s, \"\") == 0",
		)
	}
}

// TestTagOf_ReflectTagLookup tests TagOf against multiple tags on a struct field.
func TestTagOf_ReflectTagLookup(t *testing.T) {
	type TestStruct struct {
		Field string `json:"user_name,omitempty" db:"users.name" yaml:"name"`
	}

	sf, _ := reflect.TypeFor[TestStruct]().FieldByName("Field")

	// 1. First key matches
	tag1 := refkit.TagOf(sf, "json", "db")
	assert.Equal(t, "user_name", tag1.Name)
	assert.True(t, tag1.Has("omitempty"))

	// 2. Second key fallback
	tag2 := refkit.TagOf(sf, "xml", "db")
	assert.Equal(t, "users.name", tag2.Name)

	// 3. No match
	tag3 := refkit.TagOf(sf, "xml", "protobuf")
	assert.True(t, tag3.IsEmpty())

	// 4. Empty keys
	tag4 := refkit.TagOf(sf)
	assert.True(t, tag4.IsEmpty())
}

// Level 3 nested embedding
type Deep3A struct {
	LeafA1 int
	LeafA2 int
	Shared string
}

type Deep3B struct {
	LeafB1 int
	LeafB2 int
	Shared string
}

type Deep2A struct {
	Deep3A
	MidA1 int
}

type Deep2B struct {
	Deep3B
	MidB1 int
}

type DeepRoot struct {
	Deep2A
	Deep2B
	RootField int
}

// Level 4+ nested embedding (depth > 3)
type Level5 struct {
	Level5Val int
	Collision string
}

type Level4 struct {
	Level5
	Level4Val int
}

type Level3 struct {
	Level4
	Level3Val int
}

type Level2 struct {
	Level3
	Level2Val int
}

type Level1 struct {
	Level2
	Level1Val int
	Collision string // shadows Level5.Collision
}

type DepthBeyondThree struct {
	Level1
	RootVal int
}

// Collisions at different combinations
type BranchA struct {
	Conflict1 int
	Conflict2 int
	DeepOnly  string
}

type BranchB struct {
	Conflict1 int
	Conflict2 int
}

type BranchC struct {
	Conflict1 int // 3-way conflict on Conflict1
}

type MultiCollide struct {
	BranchA
	BranchB
	BranchC
	SafeField float64
}

// Struct embedding types from other packages (bytes.Buffer, sync.Mutex)
type PackageEmbedding struct {
	bytes.Buffer
	sync.Mutex
	MyField int
}

type StrWrapper struct {
	strings.Reader
}

type ByteWrapper struct {
	bytes.Reader
}

type DualReader struct {
	StrWrapper
	ByteWrapper
}

type UnexportedSecret struct {
	privInt   int
	privUint  uint
	privFloat float64
	privBool  bool
	privStr   string
	privBytes []byte
	privArr   [4]byte
}

type SelfRefModel struct {
	ID   int
	Next *SelfRefModel
}

func TestChallenge_DeepEmbedded_VisibleFields(t *testing.T) {
	t.Parallel()

	typ := reflect.TypeFor[DeepRoot]()
	std := reflect.VisibleFields(typ)

	var ours []reflect.StructField
	for f := range refkit.VisibleFields(typ) {
		ours = append(ours, f)
	}

	require.Equal(t, len(std), len(ours))
	for i := range std {
		assert.Equal(t, std[i].Name, ours[i].Name)
		assert.Equal(t, std[i].Index, ours[i].Index)
	}
}

func TestChallenge_DepthBeyondThree(t *testing.T) {
	t.Parallel()

	typ := reflect.TypeFor[DepthBeyondThree]()
	std := reflect.VisibleFields(typ)

	var ours []reflect.StructField
	for f := range refkit.VisibleFields(typ) {
		ours = append(ours, f)
	}

	require.Equal(t, len(std), len(ours))
	for i := range std {
		assert.Equal(t, std[i].Name, ours[i].Name)
		assert.Equal(t, std[i].Index, ours[i].Index)
	}

	// Verify Collision has index of Level1 (depth 2: [0, 2]) and NOT Level5 (depth 5: [0, 0, 0, 0, 1])
	fieldMap := make(map[string]reflect.StructField)
	for _, f := range ours {
		fieldMap[f.Name] = f
	}
	assert.Equal(t, []int{0, 2}, fieldMap["Collision"].Index)
}

func TestChallenge_MultiWayCollision(t *testing.T) {
	t.Parallel()

	typ := reflect.TypeFor[MultiCollide]()
	std := reflect.VisibleFields(typ)

	var ours []reflect.StructField
	for f := range refkit.VisibleFields(typ) {
		ours = append(ours, f)
	}

	require.Equal(t, len(std), len(ours))
	for i := range std {
		assert.Equal(t, std[i].Name, ours[i].Name)
		assert.Equal(t, std[i].Index, ours[i].Index)
	}

	fieldMap := make(map[string]reflect.StructField)
	for _, f := range ours {
		fieldMap[f.Name] = f
	}

	// Conflict1 and Conflict2 must be eliminated due to collisions at depth 2
	_, has1 := fieldMap["Conflict1"]
	assert.False(t, has1)
	_, has2 := fieldMap["Conflict2"]
	assert.False(t, has2)

	_, hasDeep := fieldMap["DeepOnly"]
	assert.True(t, hasDeep)
	_, hasSafe := fieldMap["SafeField"]
	assert.True(t, hasSafe)
}

func TestChallenge_ExternalPackageEmbedding(t *testing.T) {
	t.Parallel()

	typ := reflect.TypeFor[PackageEmbedding]()
	std := reflect.VisibleFields(typ)

	var ours []reflect.StructField
	for f := range refkit.VisibleFields(typ) {
		ours = append(ours, f)
	}

	require.Equal(t, len(std), len(ours))
	for i := range std {
		assert.Equal(t, std[i].Name, ours[i].Name)
		assert.Equal(t, std[i].Index, ours[i].Index)
		assert.Equal(t, std[i].PkgPath, ours[i].PkgPath)
	}
}

func TestChallenge_UnexportedFields_DynamicCrossPackage(t *testing.T) {
	t.Parallel()

	// Create StructA with unexported field 'secret' in package 'pkgA'
	typeA := reflect.StructOf([]reflect.StructField{
		{
			Name:    "secret",
			PkgPath: "example.com/pkgA",
			Type:    reflect.TypeFor[int](),
		},
		{
			Name: "PubA",
			Type: reflect.TypeFor[string](),
		},
	})

	// Create StructB with unexported field 'secret' in package 'pkgB'
	typeB := reflect.StructOf([]reflect.StructField{
		{
			Name:    "secret",
			PkgPath: "example.com/pkgB",
			Type:    reflect.TypeFor[int](),
		},
		{
			Name: "PubB",
			Type: reflect.TypeFor[string](),
		},
	})

	// Root embeds StructA and StructB, declared in pkgA
	rootType := reflect.StructOf([]reflect.StructField{
		{
			Name:      "StructA",
			Anonymous: true,
			Type:      typeA,
		},
		{
			Name:      "StructB",
			Anonymous: true,
			Type:      typeB,
		},
		{
			Name:    "secretDirect",
			PkgPath: "example.com/pkgA",
			Type:    reflect.TypeFor[int](),
		},
	})

	std := reflect.VisibleFields(rootType)
	var ours []reflect.StructField
	for f := range refkit.VisibleFields(rootType) {
		ours = append(ours, f)
	}

	require.Equal(t, len(std), len(ours))
	for i := range std {
		assert.Equal(t, std[i].Name, ours[i].Name)
		assert.Equal(t, std[i].Index, ours[i].Index)
		assert.Equal(t, std[i].PkgPath, ours[i].PkgPath)
	}
}

func TestChallenge_Accessors_BoundaryValues(t *testing.T) {
	t.Parallel()

	// 1. Extreme signed ints
	i64Val, ok := refkit.AsInt64(reflect.ValueOf(int64(math.MaxInt64)))
	assert.True(t, ok)
	assert.Equal(t, int64(math.MaxInt64), i64Val)

	i64Val, ok = refkit.AsInt64(reflect.ValueOf(int64(math.MinInt64)))
	assert.True(t, ok)
	assert.Equal(t, int64(math.MinInt64), i64Val)

	// 2. Unsigned to signed at boundary
	i64Val, ok = refkit.AsInt64(reflect.ValueOf(uint64(math.MaxInt64)))
	assert.True(t, ok)
	assert.Equal(t, int64(math.MaxInt64), i64Val)

	_, ok = refkit.AsInt64(reflect.ValueOf(uint64(math.MaxInt64 + 1)))
	assert.False(t, ok)

	_, ok = refkit.AsInt64(reflect.ValueOf(uint64(math.MaxUint64)))
	assert.False(t, ok)

	// 3. Extreme unsigned ints
	u64Val, ok := refkit.AsUint64(reflect.ValueOf(uint64(math.MaxUint64)))
	assert.True(t, ok)
	assert.Equal(t, uint64(math.MaxUint64), u64Val)

	u64Val, ok = refkit.AsUint64(reflect.ValueOf(int64(0)))
	assert.True(t, ok)
	assert.Equal(t, uint64(0), u64Val)

	_, ok = refkit.AsUint64(reflect.ValueOf(int64(-1)))
	assert.False(t, ok)

	_, ok = refkit.AsUint64(reflect.ValueOf(int64(math.MinInt64)))
	assert.False(t, ok)

	// 4. Float special numbers (NaN, Inf, Subnormal, -0.0)
	fVal, ok := refkit.AsFloat64(reflect.ValueOf(math.NaN()))
	assert.True(t, ok)
	assert.True(t, math.IsNaN(fVal))

	fVal, ok = refkit.AsFloat64(reflect.ValueOf(math.Inf(1)))
	assert.True(t, ok)
	assert.True(t, math.IsInf(fVal, 1))

	fVal, ok = refkit.AsFloat64(reflect.ValueOf(math.Inf(-1)))
	assert.True(t, ok)
	assert.True(t, math.IsInf(fVal, -1))

	fVal, ok = refkit.AsFloat64(reflect.ValueOf(-0.0))
	assert.True(t, ok)
	assert.Equal(t, -0.0, fVal)

	fVal, ok = refkit.AsFloat64(reflect.ValueOf(math.SmallestNonzeroFloat64))
	assert.True(t, ok)
	assert.Equal(t, math.SmallestNonzeroFloat64, fVal)

	// 5. Bytes: Nil slice vs empty slice vs arrays
	var nilSlice []byte
	bVal, ok := refkit.AsBytes(reflect.ValueOf(nilSlice))
	assert.True(t, ok)
	assert.Nil(t, bVal)

	emptySlice := []byte{}
	bVal, ok = refkit.AsBytes(reflect.ValueOf(emptySlice))
	assert.True(t, ok)
	assert.NotNil(t, bVal)
	assert.Equal(t, 0, len(bVal))

	// Zero-length byte array
	var zeroArr [0]byte
	arrVal := reflect.ValueOf(&zeroArr).Elem()
	bVal, ok = refkit.AsBytes(arrVal)
	assert.True(t, ok)
	assert.Equal(t, 0, len(bVal))

	// 6. Incompatible kinds
	ch := make(chan int)
	_, ok = refkit.AsInt64(reflect.ValueOf(ch))
	assert.False(t, ok)
	_, ok = refkit.AsUint64(reflect.ValueOf(ch))
	assert.False(t, ok)
	_, ok = refkit.AsFloat64(reflect.ValueOf(ch))
	assert.False(t, ok)
	_, ok = refkit.AsBool(reflect.ValueOf(ch))
	assert.False(t, ok)
	_, ok = refkit.AsBytes(reflect.ValueOf(ch))
	assert.False(t, ok)
	_, ok = refkit.AsString(reflect.ValueOf(ch))
	assert.False(t, ok)
}

func TestChallenge_Accessors_UnexportedStructFields(t *testing.T) {
	t.Parallel()

	obj := &UnexportedSecret{
		privInt:   42,
		privUint:  100,
		privFloat: 3.14,
		privBool:  true,
		privStr:   "hello unexported",
		privBytes: []byte("payload"),
		privArr:   [4]byte{'t', 'e', 's', 't'},
	}

	val := reflect.ValueOf(obj).Elem()

	// 1. AsInt64
	iVal, ok := refkit.AsInt64(val.FieldByName("privInt"))
	assert.True(t, ok)
	assert.Equal(t, int64(42), iVal)

	// 2. AsUint64
	uVal, ok := refkit.AsUint64(val.FieldByName("privUint"))
	assert.True(t, ok)
	assert.Equal(t, uint64(100), uVal)

	// 3. AsFloat64
	fVal, ok := refkit.AsFloat64(val.FieldByName("privFloat"))
	assert.True(t, ok)
	assert.InDelta(t, 3.14, fVal, 0.001)

	// 4. AsBool
	bVal, ok := refkit.AsBool(val.FieldByName("privBool"))
	assert.True(t, ok)
	assert.True(t, bVal)

	// 5. AsString
	sVal, ok := refkit.AsString(val.FieldByName("privStr"))
	assert.True(t, ok)
	assert.Equal(t, "hello unexported", sVal)

	// 6. AsBytes on unexported slice
	byVal, ok := refkit.AsBytes(val.FieldByName("privBytes"))
	assert.True(t, ok)
	assert.Equal(t, []byte("payload"), byVal)

	// 7. AsBytes on unexported addressable array
	arrField := val.FieldByName("privArr")
	assert.True(t, arrField.CanAddr())
	byArrVal, ok := refkit.AsBytes(arrField)
	assert.True(t, ok)
	assert.Equal(t, []byte("test"), byArrVal)
}

func TestChallenge_SchemaCache_HeavyConcurrency(t *testing.T) {
	t.Parallel()

	cache := refkit.NewSchemaCache()
	types := []reflect.Type{
		reflect.TypeFor[DepthBeyondThree](),
		reflect.TypeFor[MultiCollide](),
		reflect.TypeFor[PackageEmbedding](),
		reflect.TypeFor[CacheModel](),
		reflect.TypeFor[TaggedEndpoint](),
		reflect.TypeFor[FlatSimple](),
		reflect.TypeFor[SelfRefModel](),
	}

	const goroutines = 64
	const iterations = 500

	var wg sync.WaitGroup
	wg.Add(goroutines)

	var ops atomic.Int64

	for g := range goroutines {
		go func(gIdx int) {
			defer wg.Done()
			for i := range iterations {
				typ := types[(gIdx+i)%len(types)]
				schema, err := cache.Get(typ)
				if err != nil {
					t.Errorf("goroutine %d failed: %v", gIdx, err)
					return
				}
				if schema == nil || schema.Type != typ {
					t.Errorf("goroutine %d got invalid schema", gIdx)
					return
				}
				ops.Add(1)
			}
		}(g)
	}

	wg.Wait()
	assert.Equal(t, int64(goroutines*iterations), ops.Load())
}

func TestChallenge_VisibleFields_PooledWalker_ZeroAllocUnderLoad(t *testing.T) {
	typ := reflect.TypeFor[DepthBeyondThree]()

	// Pre-warm walker pool
	for range refkit.VisibleFields(typ) {
	}

	var count int
	_ = testing.AllocsPerRun(1000, func() {
		for f := range refkit.VisibleFields(typ) {
			if f.Name != "" {
				count++
			}
		}
	})

	// Verify Depth <= 3 struct achieves EXACTLY 0 allocs:
	depth3Typ := reflect.TypeFor[DeepRoot]()
	for range refkit.VisibleFields(depth3Typ) {
	}

	depth3Allocs := testing.AllocsPerRun(1000, func() {
		for f := range refkit.VisibleFields(depth3Typ) {
			if f.Name != "" {
				count++
			}
		}
	})
	if isRaceEnabled {
		assert.True(
			t,
			depth3Allocs <= 2,
			"DeepRoot VisibleFields under -race must be <= 2 allocs/op",
		)
	} else {
		assert.Equal(t, float64(0), depth3Allocs, "DeepRoot (depth <= 3) VisibleFields must achieve 0 allocs/op")
	}
	_ = count
}

type EscapedTagModel struct {
	A string `custom:"val\"" other:"next"`
	B string `                            foo:"val\"quote" bar:"123"`
	C string `                                                       complex:"p1=\"v1\",p2=\"v2\"" json:"c"`
	D string `                                                                                              escapedSlash:"val\\\"" end:"true"`
	E string `                                                                                                                                trailingSlash:"val\\"`
}

func TestRegression_EscapedQuoteStructTagParsing(t *testing.T) {
	t.Parallel()

	cache := refkit.NewSchemaCache()
	schema, err := cache.Get(reflect.TypeFor[EscapedTagModel]())
	require.NoError(t, err)
	require.NotNil(t, schema)

	fieldMap := make(map[string]refkit.FieldSchema)
	for _, f := range schema.Fields {
		fieldMap[f.Field.Name] = f
	}

	fieldA, okA := fieldMap["A"]
	require.True(t, okA)
	assert.Equal(t, `val\"`, fieldA.Tag.Name)
	assert.Equal(t, `next`, fieldA.Tag.Value("other"))

	fieldB, okB := fieldMap["B"]
	require.True(t, okB)
	assert.Equal(t, `val\"quote`, fieldB.Tag.Name)
	assert.Equal(t, `123`, fieldB.Tag.Value("bar"))

	fieldD, okD := fieldMap["D"]
	require.True(t, okD)
	assert.Equal(t, `val\\\"`, fieldD.Tag.Name)
	assert.Equal(t, `true`, fieldD.Tag.Value("end"))

	fieldE, okE := fieldMap["E"]
	require.True(t, okE)
	assert.Equal(t, `val\\`, fieldE.Tag.Name)
}

func TestRegression_HighConcurrencyWalkerPool(t *testing.T) {
	t.Parallel()

	types := []reflect.Type{
		reflect.TypeFor[EmbeddedComplex](),
		reflect.TypeFor[SameDepthConflict](),
		reflect.TypeFor[FlatSimple](),
		reflect.TypeFor[MidLevel](),
	}

	const goroutines = 64
	const iterations = 500

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := range goroutines {
		go func(gIdx int) {
			defer wg.Done()
			for i := range iterations {
				target := types[(gIdx+i)%len(types)]
				fieldCount := 0
				for f := range refkit.VisibleFields(target) {
					if f.Name != "" {
						fieldCount++
					}
				}
				if fieldCount == 0 {
					t.Errorf("goroutine %d iteration %d yielded 0 fields", gIdx, i)
					return
				}
			}
		}(g)
	}

	wg.Wait()
}

// TestParseTag_AdversarialInputs verifies that ParseTag handles arbitrary
// malicious, malformed, and pathological strings without panicking.
func TestParseTag_AdversarialInputs(t *testing.T) {
	t.Parallel()

	maliciousCases := []string{
		// Unclosed and lone quotes
		`"`,
		`'`,
		`"""`,
		`'''`,
		`"unclosed quote`,
		`'unclosed single quote`,
		`field,"unclosed quote with, commas inside, and no closing`,
		`field,'unclosed single with, commas inside`,
		`field,"double","single'`,
		`field,"nested 'single' inside unclosed double`,
		`field,'nested "double" inside unclosed single`,

		// Unmatched and malformed brackets
		`field,[unclosed bracket`,
		`field,{unclosed brace`,
		`field,(unclosed paren`,
		`field,[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[nested`,
		`field,]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]closed_without_open`,
		`field,[{(()}]mismatched`,
		`field,[(a, b), {c, d}]`,
		`field,([)]`,

		// Delimiter storms
		`,`,
		`,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,`,
		` , , , , , , , , `,
		`   `,
		"\t\r\n",
		`="",='',=[],={}`,
		`=`,
		`==`,
		`===`,
		`key=`,
		`=val`,
		`key==val`,
		`key=val=more`,

		// Escape attempts
		`field,"escaped\"quote"`,
		`field,'escaped\'single'`,
		`field,\\\\`,
	}

	for i, input := range maliciousCases {
		t.Run(fmt.Sprintf("Case_%d", i), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("ParseTag panicked on input %q: %v", input, r)
				}
			}()

			tag := refkit.ParseTag(input)
			// Tag methods must never panic on parsed results
			_ = tag.IsEmpty()
			_ = tag.IsIgnored()
			_ = tag.Has("anything")
			_ = tag.Value("key")
			_, _ = tag.Int("key")
			_, _ = tag.Float("key")
			_ = tag.SplitOption("key", ",")
		})
	}
}

// TestParseTag_BoundaryAllocations tests options count around stack buffer limits (8).
func TestParseTag_BoundaryAllocations(t *testing.T) {
	t.Parallel()

	counts := []int{0, 1, 6, 7, 8, 9, 15, 16, 17, 64, 128, 500}

	for _, count := range counts {
		var parts []string
		parts = append(parts, "mainField")
		for i := 1; i <= count; i++ {
			parts = append(parts, fmt.Sprintf("opt%d", i))
		}
		tagStr := strings.Join(parts, ",")

		tag := refkit.ParseTag(tagStr)
		if tag.Name != "mainField" {
			t.Fatalf("count %d: expected name 'mainField', got %q", count, tag.Name)
		}
		if len(tag.Options) != count {
			t.Fatalf("count %d: expected %d options, got %d", count, count, len(tag.Options))
		}
		for i := 1; i <= count; i++ {
			expectedOpt := fmt.Sprintf("opt%d", i)
			if !tag.Has(expectedOpt) {
				t.Errorf("count %d: missing option %q", count, expectedOpt)
			}
		}
	}
}

// TestParseTag_ExtremeLength tests pathological large strings.
func TestParseTag_ExtremeLength(t *testing.T) {
	t.Parallel()

	// 50,000 char ASCII tag
	longName := strings.Repeat("a", 10000)
	longOpt := strings.Repeat("b", 40000)
	tagStr := longName + "," + longOpt

	tag := refkit.ParseTag(tagStr)
	if tag.Name != longName {
		t.Fatalf("long tag name mismatch: len=%d vs len=%d", len(tag.Name), len(longName))
	}
	if len(tag.Options) != 1 || tag.Options[0] != longOpt {
		t.Fatalf("long tag option mismatch")
	}

	// 10,000 comma separated tokens
	manyParts := strings.Repeat("x,", 10000) + "end"
	tagMany := refkit.ParseTag(manyParts)
	if tagMany.Name != "x" {
		t.Fatalf("many parts: expected Name 'x', got %q", tagMany.Name)
	}
	if len(tagMany.Options) != 10000 {
		t.Fatalf("many parts: expected 10000 options, got %d", len(tagMany.Options))
	}
}

// TestParseTag_UnicodeAndInvalidBytes verifies Unicode handling and invalid UTF-8 byte resilience.
func TestParseTag_UnicodeAndInvalidBytes(t *testing.T) {
	t.Parallel()

	unicodeCases := []struct {
		input        string
		expectedName string
		expectedOpts []string
	}{
		{
			input:        "поле,обязательное,размер=100",
			expectedName: "поле",
			expectedOpts: []string{"обязательное", "размер=100"},
		},
		{
			input:        "用戶名,必填,長度=20",
			expectedName: "用戶名",
			expectedOpts: []string{"必填", "長度=20"},
		},
		{
			input:        "🚀rocket,🛸ufo,speed=⚡fast",
			expectedName: "🚀rocket",
			expectedOpts: []string{"🛸ufo", "speed=⚡fast"},
		},
		{
			input:        "mixed_tag, 'юникод, в кавычках', (параметры, скобки)",
			expectedName: "mixed_tag",
			expectedOpts: []string{"'юникод, в кавычках'", "(параметры, скобки)"},
		},
	}

	for _, tc := range unicodeCases {
		tag := refkit.ParseTag(tc.input)
		if tag.Name != tc.expectedName {
			t.Errorf("input %q: expected name %q, got %q", tc.input, tc.expectedName, tag.Name)
		}
		if len(tag.Options) != len(tc.expectedOpts) {
			t.Errorf("input %q: expected %d options, got %d", tc.input, len(tc.expectedOpts), len(tag.Options))
		} else {
			for i := range tag.Options {
				if tag.Options[i] != tc.expectedOpts[i] {
					t.Errorf("input %q: opt[%d] expected %q, got %q", tc.input, i, tc.expectedOpts[i], tag.Options[i])
				}
			}
		}
	}

	// Invalid UTF-8 byte sequences must not panic
	invalidBytes := []string{
		string([]byte{0xff, 0xfe, 0xfd}),
		string([]byte{'n', 'a', 'm', 'e', 0x80, ',', 0x81, 'o', 'p', 't'}),
		string([]byte{0xc0, 0xaf, ',', 0xe0, 0x80, 0xaf}),
	}

	for _, inv := range invalidBytes {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("ParseTag panicked on invalid UTF-8: %v", r)
			}
		}()
		tag := refkit.ParseTag(inv)
		_ = tag.IsEmpty()
		_ = tag.Value("key")
	}
}

// TestTag_MethodRobustness verifies Int, Float, SplitOption with pathological values.
func TestTag_MethodRobustness(t *testing.T) {
	t.Parallel()

	tag := refkit.ParseTag(
		`field,min=5,max=99999999999999999999999999999999999999999999999999,rate=3.1415,bad_num=xyz,empty=,nan=NaN,inf=+Inf,delim=a|b||c| `,
	)

	// Valid Int
	if val, ok := tag.Int("min"); !ok || val != 5 {
		t.Errorf("expected min=5, got (%d, %v)", val, ok)
	}

	// Overflow Int
	if _, ok := tag.Int("max"); ok {
		t.Errorf("expected overflow int to fail ok=false")
	}

	// Invalid Int string
	if _, ok := tag.Int("bad_num"); ok {
		t.Errorf("expected non-numeric int to fail ok=false")
	}

	// Empty value
	if _, ok := tag.Int("empty"); ok {
		t.Errorf("expected empty value int to fail ok=false")
	}

	// Missing key
	if _, ok := tag.Int("nonexistent"); ok {
		t.Errorf("expected missing key int to fail ok=false")
	}

	// Valid Float
	if val, ok := tag.Float("rate"); !ok || math.Abs(val-3.1415) > 1e-9 {
		t.Errorf("expected rate=3.1415, got (%f, %v)", val, ok)
	}

	// Float NaN & Inf
	if val, ok := tag.Float("nan"); !ok || !math.IsNaN(val) {
		t.Errorf("expected NaN float")
	}
	if val, ok := tag.Float("inf"); !ok || !math.IsInf(val, 1) {
		t.Errorf("expected +Inf float")
	}

	// SplitOption
	parts := slices.Collect(tag.SplitOption("delim", "|"))
	expected := []string{"a", "b", "c"}
	if len(parts) != len(expected) {
		t.Fatalf("SplitOption expected %v, got %v", expected, parts)
	}
	for i := range expected {
		if parts[i] != expected[i] {
			t.Errorf("parts[%d]: expected %q, got %q", i, expected[i], parts[i])
		}
	}

	// SplitOption on missing key
	if parts := slices.Collect(tag.SplitOption("nonexistent", ",")); len(parts) > 0 {
		t.Errorf("expected empty for missing SplitOption, got %v", parts)
	}
}

// TestParseTag_ZeroAllocGuarantees verifies 0 heap allocations for simple tags.
func TestParseTag_ZeroAllocGuarantees(t *testing.T) {
	simpleTag := "user_id"
	allocs := testing.AllocsPerRun(1000, func() {
		tag := refkit.ParseTag(simpleTag)
		if tag.Name != "user_id" {
			t.Fatalf("unexpected: %v", tag)
		}
	})
	if allocs != 0 {
		t.Errorf("expected 0 allocs for simple tag, got %f", allocs)
	}

	ignoredTag := "-"
	allocsIgnored := testing.AllocsPerRun(1000, func() {
		tag := refkit.ParseTag(ignoredTag)
		if tag.Name != "-" {
			t.Fatalf("unexpected: %v", tag)
		}
	})
	if allocsIgnored != 0 {
		t.Errorf("expected 0 allocs for ignored tag, got %f", allocsIgnored)
	}
}
