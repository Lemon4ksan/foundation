// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package refkit_test

import (
	"reflect"
	"sync"
	"testing"
	"unsafe"

	"github.com/lemon4ksan/foundation/refkit"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

type CacheModel struct {
	ID   int64    `json:"id"             db:"primary_key"`
	Name string   `json:"name,omitempty"                  validate:"min=3,max=50"`
	Tags []string `json:"tags"`
	FlatSimple
}

type TaggedEndpoint struct {
	Path    string `route:"/api/v1/users"`
	Method  string `                       method:"GET"`
	Timeout int    `route:"timeout,sec=30"`
	Custom  string `                                    custom:"bare_value"`
	Empty   string
}

func TestSchemaCache_Get(t *testing.T) {
	t.Parallel()

	cache := refkit.NewSchemaCache()
	typ := reflect.TypeFor[CacheModel]()

	// 1. Cache Miss (Populate)
	schema, err := cache.Get(typ)
	require.NoError(t, err)
	require.NotNil(t, schema)
	assert.Equal(t, typ, schema.Type)
	assert.Equal(t, typ.Size(), schema.Size)

	// Check fields
	assert.True(t, len(schema.Fields) >= 4)
	idField := schema.Fields[0]
	assert.Equal(t, "ID", idField.Field.Name)
	assert.Equal(t, uintptr(0), idField.Offset)
	assert.Equal(t, "id", idField.Tag.Name)
	assert.Equal(t, "primary_key", idField.Tag.Get("db"))

	nameField := schema.Fields[1]
	assert.Equal(t, "Name", nameField.Field.Name)
	assert.True(t, nameField.Tag.Has("omitempty"))
	minLen, ok := nameField.Tag.GetInt("min")
	assert.True(t, ok)
	assert.Equal(t, 3, minLen)

	// 2. Cache Hit
	schema2, err := cache.Get(typ)
	require.NoError(t, err)
	assert.Same(t, schema, schema2)

	// 3. Pointer to struct
	schemaPtr, err := cache.Get(reflect.TypeFor[*CacheModel]())
	require.NoError(t, err)
	assert.Same(t, schema, schemaPtr)
}

func TestSchemaCache_CumulativeOffsets(t *testing.T) {
	t.Parallel()

	cache := refkit.NewSchemaCache()
	typ := reflect.TypeFor[EmbeddedComplex]()

	schema, err := cache.Get(typ)
	require.NoError(t, err)

	instance := EmbeddedComplex{
		MidLevel: MidLevel{
			BaseLevel: BaseLevel{
				BaseID: 101,
			},
			MidID: 202,
		},
		TopID: 303,
	}

	ptr := unsafe.Pointer(&instance)
	for _, f := range schema.Fields {
		switch f.Field.Name {
		case "TopID":
			val := *(*int)(unsafe.Pointer(uintptr(ptr) + f.Offset))
			assert.Equal(t, 303, val)
		case "MidID":
			val := *(*int)(unsafe.Pointer(uintptr(ptr) + f.Offset))
			assert.Equal(t, 202, val)
		case "BaseID":
			val := *(*int)(unsafe.Pointer(uintptr(ptr) + f.Offset))
			assert.Equal(t, 101, val)
		}
	}
}

func TestSchemaCache_TagParsing(t *testing.T) {
	t.Parallel()

	cache := refkit.NewSchemaCache()
	schema, err := cache.Get(reflect.TypeFor[TaggedEndpoint]())
	require.NoError(t, err)

	fieldMap := make(map[string]refkit.FieldSchema)
	for _, f := range schema.Fields {
		fieldMap[f.Field.Name] = f
	}

	pathF := fieldMap["Path"]
	assert.Equal(t, "/api/v1/users", pathF.Tag.Get("route"))

	methodF := fieldMap["Method"]
	assert.Equal(t, "GET", methodF.Tag.Get("method"))

	timeoutF := fieldMap["Timeout"]
	assert.Equal(t, "timeout,sec=30", timeoutF.Tag.Get("route"))
	sec, ok := timeoutF.Tag.GetInt("sec")
	assert.True(t, ok)
	assert.Equal(t, 30, sec)

	customF := fieldMap["Custom"]
	assert.Equal(t, "bare_value", customF.Tag.Name)
	assert.Equal(t, "bare_value", customF.Tag.Get("custom"))

	emptyF := fieldMap["Empty"]
	assert.True(t, emptyF.Tag.IsEmpty())
}

func TestSchemaCache_Errors(t *testing.T) {
	t.Parallel()

	cache := refkit.NewSchemaCache()

	// Nil type
	_, err := cache.Get(nil)
	assert.Error(t, err)

	// Non-struct types
	nonStructs := []reflect.Type{
		reflect.TypeFor[int](),
		reflect.TypeFor[string](),
		reflect.TypeFor[[]byte](),
		reflect.TypeFor[chan int](),
		reflect.TypeFor[map[string]int](),
	}
	for _, nonStruct := range nonStructs {
		_, err := cache.Get(nonStruct)
		assert.Error(t, err)
	}
}

func TestSchemaCache_ZeroAlloc(t *testing.T) {
	cache := refkit.NewSchemaCache()
	typ := reflect.TypeFor[CacheModel]()

	// Pre-warm cache
	_, err := cache.Get(typ)
	require.NoError(t, err)

	var sink *refkit.StructSchema
	allocs := testing.AllocsPerRun(1000, func() {
		s, err := cache.Get(typ)
		if err != nil {
			t.Fatal(err)
		}
		sink = s
	})

	assert.Equal(t, float64(0), allocs, "SchemaCache.Get on warm cache hit must achieve 0 allocs/op")
	_ = sink
}

func TestSchemaCache_Concurrency(t *testing.T) {
	t.Parallel()

	cache := refkit.NewSchemaCache()
	types := []reflect.Type{
		reflect.TypeFor[CacheModel](),
		reflect.TypeFor[EmbeddedComplex](),
		reflect.TypeFor[TaggedEndpoint](),
		reflect.TypeFor[FlatSimple](),
	}

	const goroutines = 32
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func(gIdx int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				targetTyp := types[(gIdx+i)%len(types)]
				schema, err := cache.Get(targetTyp)
				if err != nil {
					t.Errorf("goroutine %d failed on iteration %d: %v", gIdx, i, err)
					return
				}
				if schema == nil || schema.Size == 0 {
					t.Errorf("goroutine %d got invalid schema", gIdx)
					return
				}
			}
		}(g)
	}

	wg.Wait()
}

func BenchmarkSchemaCache_Get_Hit(b *testing.B) {
	cache := refkit.NewSchemaCache()
	typ := reflect.TypeFor[CacheModel]()
	_, _ = cache.Get(typ)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		s, _ := cache.Get(typ)
		_ = s
	}
}

func TestScenario_HighThroughputSerializerFastPath(t *testing.T) {
	t.Parallel()

	cache := refkit.NewSchemaCache()
	model := CacheModel{
		ID:         42,
		Name:       "alice",
		Tags:       []string{"admin", "ops"},
		FlatSimple: FlatSimple{ID: 100, Name: "inner", IsActive: true},
	}

	schema, err := cache.Get(reflect.TypeOf(model))
	require.NoError(t, err)

	val := reflect.ValueOf(model)
	var fieldsWritten int
	for _, f := range schema.Fields {
		fieldVal := val.FieldByIndex(f.Field.Index)
		switch f.Field.Type.Kind() {
		case reflect.Int, reflect.Int64:
			num, ok := refkit.AsInt64(fieldVal)
			assert.True(t, ok)
			assert.True(t, num == 42 || num == 100 || num == 0)
			fieldsWritten++
		case reflect.String:
			s, ok := refkit.AsString(fieldVal)
			assert.True(t, ok)
			assert.True(t, s == "alice" || s == "inner")
			fieldsWritten++
		case reflect.Bool:
			b, ok := refkit.AsBool(fieldVal)
			assert.True(t, ok)
			assert.True(t, b)
			fieldsWritten++
		}
	}
	assert.True(t, fieldsWritten >= 4)
}

func TestScenario_CodecTagBasedFieldFiltering(t *testing.T) {
	t.Parallel()

	cache := refkit.NewSchemaCache()
	schema, err := cache.Get(reflect.TypeFor[TaggedEndpoint]())
	require.NoError(t, err)

	routeFields := make(map[string]string)
	for _, f := range schema.Fields {
		if route := f.Tag.Get("route"); route != "" {
			routeFields[f.Field.Name] = route
		}
	}

	assert.Equal(t, "/api/v1/users", routeFields["Path"])
	assert.Equal(t, "timeout,sec=30", routeFields["Timeout"])
	assert.Equal(t, 2, len(routeFields))
}
