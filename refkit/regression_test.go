// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package refkit_test

import (
	"reflect"
	"sync"
	"testing"

	"github.com/lemon4ksan/foundation/refkit"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

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

	for g := 0; g < goroutines; g++ {
		go func(gIdx int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
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
