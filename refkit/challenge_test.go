// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package refkit_test

import (
	"bytes"
	"math"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/lemon4ksan/foundation/refkit"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

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

	for g := 0; g < goroutines; g++ {
		go func(gIdx int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
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
	assert.Equal(t, float64(0), depth3Allocs, "DeepRoot (depth <= 3) VisibleFields must achieve 0 allocs/op")
	_ = count
}
