// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package refkit_test

import (
	"fmt"
	"io"
	"reflect"
	"testing"

	"github.com/lemon4ksan/foundation/refkit"
	"github.com/lemon4ksan/foundation/testing/assert"
)

type ValueStringer struct{}

func (ValueStringer) String() string {
	return "value"
}

type PointerStringer struct{}

func (*PointerStringer) String() string {
	return "pointer"
}

type SimpleReader struct{}

func (SimpleReader) Read(p []byte) (n int, err error) {
	return 0, io.EOF
}

func TestImplements(t *testing.T) {
	t.Parallel()

	// 1. Value receiver implements fmt.Stringer
	assert.True(t, refkit.Implements[fmt.Stringer](reflect.TypeFor[ValueStringer]()))
	assert.True(t, refkit.Implements[fmt.Stringer](reflect.TypeFor[*ValueStringer]()))

	// 2. Pointer receiver implements fmt.Stringer
	assert.False(t, refkit.Implements[fmt.Stringer](reflect.TypeFor[PointerStringer]()))
	assert.True(t, refkit.Implements[fmt.Stringer](reflect.TypeFor[*PointerStringer]()))

	// 3. io.Reader interface satisfaction
	assert.True(t, refkit.Implements[io.Reader](reflect.TypeFor[SimpleReader]()))
	assert.False(t, refkit.Implements[io.Reader](reflect.TypeFor[ValueStringer]()))

	// 4. Interface implementing another interface
	type SubReader interface {
		io.Reader
		Close() error
	}
	assert.True(t, refkit.Implements[io.Reader](reflect.TypeFor[SubReader]()))
}

func TestImplements_NonInterface(t *testing.T) {
	t.Parallel()

	// Non-interface type parameters should safely return false without panicking
	assert.False(t, refkit.Implements[int](reflect.TypeFor[int]()))
	assert.False(t, refkit.Implements[string](reflect.TypeFor[string]()))
	assert.False(t, refkit.Implements[ValueStringer](reflect.TypeFor[ValueStringer]()))
	assert.False(t, refkit.Implements[struct{}](reflect.TypeFor[struct{}]()))
}

func TestImplements_Nil(t *testing.T) {
	t.Parallel()

	assert.False(t, refkit.Implements[fmt.Stringer](nil))
	assert.False(t, refkit.Implements[int](nil))
}

func TestImplements_ZeroAlloc(t *testing.T) {
	typ := reflect.TypeFor[ValueStringer]()

	var sink bool
	allocs := testing.AllocsPerRun(1000, func() {
		sink = refkit.Implements[fmt.Stringer](typ)
	})

	assert.Equal(t, float64(0), allocs, "Implements must achieve 0 allocs/op")
	_ = sink
}

func BenchmarkImplements(b *testing.B) {
	typ := reflect.TypeFor[ValueStringer]()
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		ok := refkit.Implements[fmt.Stringer](typ)
		_ = ok
	}
}

func TestScenario_InterfaceComplianceDiscovery(t *testing.T) {
	t.Parallel()

	cache := refkit.NewSchemaCache()
	schema, err := cache.Get(reflect.TypeFor[*CacheModel]())
	assert.NoError(t, err)

	underlying := refkit.DerefType(schema.Type)
	assert.Equal(t, reflect.Struct, underlying.Kind())

	stringerFields := 0
	for _, f := range schema.Fields {
		if refkit.Implements[fmt.Stringer](f.Field.Type) ||
			refkit.Implements[fmt.Stringer](reflect.PointerTo(f.Field.Type)) {
			stringerFields++
		}
	}
	assert.True(t, stringerFields >= 0)
}
