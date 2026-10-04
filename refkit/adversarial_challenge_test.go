// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package refkit_test

import (
	"io"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"unsafe"

	"github.com/lemon4ksan/foundation/refkit"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

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

	for i := 0; i < 50; i++ {
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

	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
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
	assert.Equal(t, float64(0), allocsVisible, "VisibleFields (depth <= 3) must be 0 allocs/op")

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
