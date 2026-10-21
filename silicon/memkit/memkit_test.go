// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package memkit_test

import (
	"fmt"
	"testing"
	"unsafe"

	"github.com/lemon4ksan/foundation/silicon/memkit"
	"github.com/lemon4ksan/foundation/testing/assert"
)

type leadingZeroMarkers struct {
	_   memkit.NoCopy
	_   memkit.NoCompare
	_   memkit.Align64
	_   memkit.HostLayout
	val uint64
}

type trailingZeroMarker struct {
	val uint64
	_   memkit.NoCopy
}

func TestZeroSizedMarkers_LayoutAndOrdering(t *testing.T) {
	t.Parallel()

	var nc memkit.NoCopy
	nc.Lock()
	nc.Unlock()

	assert.Equal(t, uintptr(0), unsafe.Sizeof(memkit.NoCopy{}))
	assert.Equal(t, uintptr(1), unsafe.Alignof(memkit.NoCopy{}))

	assert.Equal(t, uintptr(0), unsafe.Sizeof(memkit.NoCompare{}))
	assert.Equal(t, unsafe.Alignof(uintptr(0)), unsafe.Alignof(memkit.NoCompare{}))

	assert.Equal(t, uintptr(0), unsafe.Sizeof(memkit.Align64{}))
	assert.Equal(t, uintptr(8), unsafe.Alignof(memkit.Align64{}))

	// When zero-sized markers are placed at the start of a struct, they add 0 bytes.
	assert.Equal(t, uintptr(8), unsafe.Sizeof(leadingZeroMarkers{}))
	assert.Equal(t, uintptr(8), unsafe.Alignof(leadingZeroMarkers{}))

	// When a zero-sized marker is placed at the end of a non-empty struct, the Go
	// compiler pads the struct by 1 byte rounded up to alignment (8 -> 16 bytes).
	assert.Equal(t, uintptr(16), unsafe.Sizeof(trailingZeroMarker{}))
}

func TestCacheLinePad(t *testing.T) {
	t.Parallel()

	assert.True(t, memkit.CacheLineSize >= 64)
	assert.Equal(t, uintptr(memkit.CacheLineSize), unsafe.Sizeof(memkit.CacheLinePad{}))
}

func TestEfaceAndIface(t *testing.T) {
	t.Parallel()

	assert.True(t, memkit.IsNilInterface(nil))

	var typedNil *int
	var boxedNil any = typedNil
	// Standard Go comparison reports false because Eface.Type != nil.
	assert.False(t, boxedNil == nil)
	// IsNilInterface inspects Eface.Data and reports true in O(1).
	assert.True(t, memkit.IsNilInterface(boxedNil))

	x := 42
	var boxedVal any = &x
	assert.False(t, memkit.IsNilInterface(boxedVal))

	ef := memkit.EfaceOf(&boxedVal)
	assert.NotNil(t, ef.Type)
	assert.Equal(t, unsafe.Pointer(&x), ef.Data)

	var strIface fmt.Stringer = dummyStringer{}
	ifa := memkit.IfaceOf(&strIface)
	assert.NotNil(t, ifa.Itab)
}

type dummyStringer struct{}

func (dummyStringer) String() string { return "ok" }

func TestNoEscape(t *testing.T) {
	t.Parallel()

	val := 99
	p := memkit.NoEscape(&val)
	assert.Equal(t, 99, *p)

	raw := memkit.NoEscapePointer(unsafe.Pointer(&val))
	assert.Equal(t, unsafe.Pointer(&val), raw)

	assert.Nil(t, memkit.NoEscapeBytes(nil))
	buf := []byte("silicon-memkit")
	noEscBuf := memkit.NoEscapeBytes(buf)
	assert.Equal(t, "silicon-memkit", string(noEscBuf))

	assert.Equal(t, "", memkit.NoEscapeString(""))
	s := "zero-escape"
	assert.Equal(t, s, memkit.NoEscapeString(s))
}

func TestUintptrToPointer(t *testing.T) {
	t.Parallel()

	var word uint64 = 0xCAFEBABE
	addr := uintptr(unsafe.Pointer(&word))
	ptr := memkit.UintptrToPointer(addr)
	assert.Equal(t, unsafe.Pointer(&word), ptr)
	assert.Equal(t, uint64(0xCAFEBABE), *(*uint64)(ptr))
}
