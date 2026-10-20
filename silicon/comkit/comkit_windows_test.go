// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows

package comkit_test

import (
	"sync/atomic"
	"syscall"
	"testing"
	"unsafe"

	"github.com/lemon4ksan/foundation/silicon/comkit"
)

type mockCOMObject struct {
	vtbl     *mockVtbl
	refCount int32
	lastSum  uintptr
}

type mockVtbl struct {
	queryInterface uintptr
	addRef         uintptr
	release        uintptr
	call0          uintptr
	call1          uintptr
	call2          uintptr
	call3          uintptr
	call4          uintptr
	call5          uintptr
	call6          uintptr
	call7          uintptr
	call8          uintptr
	call9          uintptr
}

func newMockCOMObject() *mockCOMObject {
	vtbl := &mockVtbl{
		queryInterface: syscall.NewCallback(func(this *mockCOMObject, iid *comkit.GUID, out *uintptr) uintptr {
			if iid == nil || out == nil {
				return comkit.E_POINTER.Uintptr()
			}
			if iid.Equal(comkit.IIDIUnknown) {
				atomic.AddInt32(&this.refCount, 1)
				*out = uintptr(unsafe.Pointer(this))
				return comkit.S_OK.Uintptr()
			}
			*out = 0
			return comkit.E_NOINTERFACE.Uintptr()
		}),
		addRef: syscall.NewCallback(func(this *mockCOMObject) uintptr {
			return uintptr(atomic.AddInt32(&this.refCount, 1))
		}),
		release: syscall.NewCallback(func(this *mockCOMObject) uintptr {
			return uintptr(atomic.AddInt32(&this.refCount, -1))
		}),
		call0: syscall.NewCallback(func(this *mockCOMObject) uintptr {
			this.lastSum = 42
			return uintptr(comkit.S_OK)
		}),
		call1: syscall.NewCallback(func(this *mockCOMObject, a1 uintptr) uintptr {
			this.lastSum = a1
			return uintptr(comkit.S_OK)
		}),
		call2: syscall.NewCallback(func(this *mockCOMObject, a1, a2 uintptr) uintptr {
			this.lastSum = a1 + a2
			return uintptr(comkit.S_OK)
		}),
		call3: syscall.NewCallback(func(this *mockCOMObject, a1, a2, a3 uintptr) uintptr {
			this.lastSum = a1 + a2 + a3
			return uintptr(comkit.S_OK)
		}),
		call4: syscall.NewCallback(func(this *mockCOMObject, a1, a2, a3, a4 uintptr) uintptr {
			this.lastSum = a1 + a2 + a3 + a4
			return uintptr(comkit.S_OK)
		}),
		call5: syscall.NewCallback(func(this *mockCOMObject, a1, a2, a3, a4, a5 uintptr) uintptr {
			this.lastSum = a1 + a2 + a3 + a4 + a5
			return uintptr(comkit.S_OK)
		}),
		call6: syscall.NewCallback(func(this *mockCOMObject, a1, a2, a3, a4, a5, a6 uintptr) uintptr {
			this.lastSum = a1 + a2 + a3 + a4 + a5 + a6
			return uintptr(comkit.S_OK)
		}),
		call7: syscall.NewCallback(func(this *mockCOMObject, a1, a2, a3, a4, a5, a6, a7 uintptr) uintptr {
			this.lastSum = a1 + a2 + a3 + a4 + a5 + a6 + a7
			return uintptr(comkit.S_OK)
		}),
		call8: syscall.NewCallback(func(this *mockCOMObject, a1, a2, a3, a4, a5, a6, a7, a8 uintptr) uintptr {
			this.lastSum = a1 + a2 + a3 + a4 + a5 + a6 + a7 + a8
			return uintptr(comkit.S_OK)
		}),
		call9: syscall.NewCallback(func(this *mockCOMObject, a1, a2, a3, a4, a5, a6, a7, a8, a9 uintptr) uintptr {
			this.lastSum = a1 + a2 + a3 + a4 + a5 + a6 + a7 + a8 + a9
			return uintptr(comkit.S_OK)
		}),
	}
	return &mockCOMObject{vtbl: vtbl, refCount: 1}
}

func TestVtblCalls_And_IUnknown(t *testing.T) {
	mock := newMockCOMObject()
	inst := uintptr(unsafe.Pointer(mock))
	ptr := comkit.NewPtr(inst)

	if got := ptr.AddRef(); got != 2 {
		t.Fatalf("AddRef() = %d, want 2", got)
	}
	if got := ptr.Release(); got != 1 {
		t.Fatalf("Release() = %d, want 1", got)
	}

	queried, hr := ptr.QueryInterface(&comkit.IIDIUnknown)
	if hr.Failed() || queried.Raw() != inst {
		t.Fatalf("QueryInterface(IIDIUnknown) failed: hr=0x%X, ptr=0x%X", uint32(hr), queried.Raw())
	}
	if got := comkit.ComRelease(queried.Raw()); got != 1 {
		t.Fatalf("ComRelease() = %d, want 1", got)
	}

	_, hrNo := ptr.QueryInterface(&comkit.IIDIDispatch)
	if hrNo != comkit.E_NOINTERFACE {
		t.Fatalf("QueryInterface(IIDIDispatch) = 0x%X, want E_NOINTERFACE", uint32(hrNo))
	}

	// Null pointer guard tests
	if comkit.VtblCall0(0, 3) != comkit.E_POINTER {
		t.Fatal("expected E_POINTER on nil instance")
	}
	if (comkit.Ptr{}).AddRef() != 0 || (comkit.Ptr{}).Release() != 0 || comkit.ComRelease(0) != 0 {
		t.Fatal("expected 0 on nil Release/AddRef")
	}

	// Arity dispatch tests (slots 3..10)
	if hr := comkit.VtblCall0(inst, 3); hr.Failed() || mock.lastSum != 42 {
		t.Fatalf("VtblCall0 failed: hr=%v sum=%d", hr, mock.lastSum)
	}
	if hr := comkit.VtblCall1(inst, 4, 10); hr.Failed() || mock.lastSum != 10 {
		t.Fatalf("VtblCall1 failed: hr=%v sum=%d", hr, mock.lastSum)
	}
	if hr := comkit.VtblCall2(inst, 5, 10, 20); hr.Failed() || mock.lastSum != 30 {
		t.Fatalf("VtblCall2 failed: hr=%v sum=%d", hr, mock.lastSum)
	}
	if hr := comkit.VtblCall3(inst, 6, 1, 2, 3); hr.Failed() || mock.lastSum != 6 {
		t.Fatalf("VtblCall3 failed: hr=%v sum=%d", hr, mock.lastSum)
	}
	if hr := comkit.VtblCall4(inst, 7, 1, 2, 3, 4); hr.Failed() || mock.lastSum != 10 {
		t.Fatalf("VtblCall4 failed: hr=%v sum=%d", hr, mock.lastSum)
	}
	if hr := comkit.VtblCall5(inst, 8, 1, 2, 3, 4, 5); hr.Failed() || mock.lastSum != 15 {
		t.Fatalf("VtblCall5 failed: hr=%v sum=%d", hr, mock.lastSum)
	}
	if hr := comkit.VtblCall6(inst, 9, 1, 2, 3, 4, 5, 6); hr.Failed() || mock.lastSum != 21 {
		t.Fatalf("VtblCall6 failed: hr=%v sum=%d", hr, mock.lastSum)
	}
	if hr := comkit.VtblCall7(inst, 10, 1, 2, 3, 4, 5, 6, 7); hr.Failed() || mock.lastSum != 28 {
		t.Fatalf("VtblCall7 failed: hr=%v sum=%d", hr, mock.lastSum)
	}
	if hr := comkit.VtblCall8(inst, 11, 1, 2, 3, 4, 5, 6, 7, 8); hr.Failed() || mock.lastSum != 36 {
		t.Fatalf("VtblCall8 failed: hr=%v sum=%d", hr, mock.lastSum)
	}
	if hr := comkit.VtblCall9(inst, 12, 1, 2, 3, 4, 5, 6, 7, 8, 9); hr.Failed() || mock.lastSum != 45 {
		t.Fatalf("VtblCall9 failed: hr=%v sum=%d", hr, mock.lastSum)
	}

	// Zero-allocation hot path verification
	allocs := testing.AllocsPerRun(1000, func() {
		_ = comkit.VtblCall2(inst, 5, 7, 8)
	})
	if allocs != 0 {
		t.Fatalf("VtblCall2 allocated %v times, want 0", allocs)
	}
}

func TestBSTR_HSTRING_And_CoTaskMem(t *testing.T) {
	// 1. BSTR tests (including emoji surrogate pair)
	const sample = "Hello, Stasis COM 🚀!"
	b := comkit.AllocBSTR(sample)
	if b == 0 {
		t.Fatal("AllocBSTR returned 0")
	}

	var buf [128]byte
	allocs := testing.AllocsPerRun(100, func() {
		res := comkit.AppendBSTRUTF8(buf[:0], b)
		if string(res) != sample {
			t.Fatalf("AppendBSTRUTF8 = %q, want %q", string(res), sample)
		}
	})
	if allocs != 0 {
		t.Fatalf("AppendBSTRUTF8 allocated %v times, want 0", allocs)
	}

	if got := comkit.BSTRToString(b); got != sample {
		t.Fatalf("BSTRToString = %q, want %q", got, sample)
	}

	emptyB := comkit.AllocBSTR("")
	if got := comkit.BSTRToString(emptyB); got != "" {
		t.Fatalf("empty BSTRToString = %q, want empty", got)
	}

	// 2. WinRT HSTRING tests
	hs, err := comkit.NewHString(sample)
	if err != nil || hs == 0 {
		t.Fatalf("NewHString failed: hs=%v, err=%v", hs, err)
	}
	defer comkit.DeleteHString(hs)

	allocsH := testing.AllocsPerRun(100, func() {
		res := comkit.AppendHStringUTF8(buf[:0], hs)
		if string(res) != sample {
			t.Fatalf("AppendHStringUTF8 = %q, want %q", string(res), sample)
		}
	})
	if allocsH != 0 {
		t.Fatalf("AppendHStringUTF8 allocated %v times, want 0", allocsH)
	}

	emptyHS, err := comkit.NewHString("")
	if err != nil || emptyHS != 0 {
		t.Fatalf("NewHString(\"\") = %v, %v", emptyHS, err)
	}

	// 3. CoTaskMemAlloc / AppendCoTaskUTF8 / DrainCoTaskUTF16
	raw := comkit.CoTaskMemAlloc(16)
	if raw == nil {
		t.Fatal("CoTaskMemAlloc(16) returned nil")
	}
	u16 := unsafe.Slice((*uint16)(raw), 6)
	u16[0] = 'O'
	u16[1] = 'K'
	u16[2] = 0
	res := comkit.AppendCoTaskUTF8(buf[:0], (*uint16)(raw))
	if string(res) != "OK" {
		t.Fatalf("AppendCoTaskUTF8 = %q, want \"OK\"", string(res))
	}

	raw2 := comkit.CoTaskMemAlloc(16)
	u16b := unsafe.Slice((*uint16)(raw2), 4)
	u16b[0] = 'G'
	u16b[1] = 'O'
	u16b[2] = 0
	if got := comkit.DrainCoTaskUTF16((*uint16)(raw2)); got != "GO" {
		t.Fatalf("DrainCoTaskUTF16 = %q, want \"GO\"", got)
	}
}

func TestLockThreadMTA_And_WinRTActivation(t *testing.T) {
	unlock, err := comkit.LockThreadMTA()
	if err != nil {
		t.Fatalf("LockThreadMTA failed: %v", err)
	}
	defer unlock()

	if comkit.E_FAIL.Uint32() != 0x80004005 {
		t.Fatalf("E_FAIL.Uint32() = 0x%X, want 0x80004005", comkit.E_FAIL.Uint32())
	}

	// Null guard checks
	if _, err := comkit.CoCreateInstance(nil, 0, comkit.ClsctxInprocServer, &comkit.IIDIUnknown); err == nil {
		t.Fatal("expected error for nil clsid in CoCreateInstance")
	}
	if _, err := comkit.RoGetActivationFactory("Windows.Foundation.Uri", nil); err == nil {
		t.Fatal("expected error for nil iid in RoGetActivationFactory")
	}

	// Activate standard CUIAutomation COM object via CoCreateInstance
	clsidUIA, _ := comkit.ParseGUID("{FF48DBA4-60EF-4201-AA87-54103EEF594E}")
	uia, err := comkit.CoCreateInstance(&clsidUIA, 0, comkit.ClsctxInprocServer, &comkit.IIDIUnknown)
	if err != nil {
		t.Fatalf("CoCreateInstance(CUIAutomation) failed: %v", err)
	}
	if uia.IsNil() {
		t.Fatal("expected non-nil CUIAutomation instance")
	}
	uia.Release()

	// Activate standard Windows.Foundation.Uri factory via RoGetActivationFactory
	// IUriRuntimeClassFactory IID: {44A9796F-723E-4FDF-A218-033E75B0C084}
	iidUriFactory, err := comkit.ParseGUID("{44A9796F-723E-4FDF-A218-033E75B0C084}")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := comkit.RoGetActivationFactory("Windows.Foundation.Uri", &iidUriFactory)
	if err != nil {
		t.Fatalf("RoGetActivationFactory(Windows.Foundation.Uri) failed: %v", err)
	}
	if factory.IsNil() {
		t.Fatal("expected non-nil factory pointer")
	}
	defer factory.Release()

	// Activate Windows.Foundation.Collections.PropertySet via RoActivateInstance
	propSet, err := comkit.RoActivateInstance("Windows.Foundation.Collections.PropertySet")
	if err != nil {
		t.Fatalf("RoActivateInstance(PropertySet) failed: %v", err)
	}
	if propSet.IsNil() {
		t.Fatal("expected non-nil PropertySet instance")
	}
	propSet.Release()

	// Verify edge cases on nil/zero memory helpers
	comkit.FreeBSTR(0)
	if len(comkit.AppendBSTRUTF8(nil, 0)) != 0 {
		t.Fatal("expected empty slice for zero BSTR")
	}
	if comkit.BSTRToString(0) != "" {
		t.Fatal("expected empty string for zero BSTR")
	}
	comkit.DeleteHString(0)
	if len(comkit.AppendHStringUTF8(nil, 0)) != 0 {
		t.Fatal("expected empty slice for zero HSTRING")
	}
	if comkit.CoTaskMemAlloc(0) != nil {
		t.Fatal("expected nil for CoTaskMemAlloc(0)")
	}
	comkit.CoTaskMemFree(nil)
	if len(comkit.AppendCoTaskUTF8(nil, nil)) != 0 {
		t.Fatal("expected empty slice for nil CoTask pointer")
	}
	if comkit.DrainCoTaskUTF16(nil) != "" {
		t.Fatal("expected empty string for nil CoTask pointer")
	}

	// Verify VariantClear, SAFEARRAY, and CoSetProxyBlanket
	if err := comkit.VariantClear(nil); err != nil {
		t.Fatalf("VariantClear(nil) failed: %v", err)
	}
	vb := comkit.NewVariantBSTR(comkit.AllocBSTR("variant-clear-test"))
	if err := comkit.VariantClear(&vb); err != nil || vb.VT != comkit.VTEmpty {
		t.Fatalf("VariantClear(&vb) failed: err=%v, vt=%d", err, vb.VT)
	}

	psa := comkit.SafeArrayCreateVector(comkit.VTI4, 0, 3)
	if psa == 0 {
		t.Fatal("SafeArrayCreateVector returned 0")
	}
	lb, ub, err := comkit.SafeArrayBounds(psa)
	if err != nil || lb != 0 || ub != 2 {
		t.Fatalf("SafeArrayBounds = (%d, %d, %v), want (0, 2, nil)", lb, ub, err)
	}
	var intsBuf [4]int32
	ints, err := comkit.AppendSafeArrayInt32(intsBuf[:0], psa)
	if err != nil || len(ints) != 3 {
		t.Fatalf("AppendSafeArrayInt32 = (%v, %v), want len 3", ints, err)
	}
	if err := comkit.SafeArrayDestroy(psa); err != nil {
		t.Fatalf("SafeArrayDestroy failed: %v", err)
	}
	if err := comkit.SafeArrayDestroy(0); err != nil {
		t.Fatalf("SafeArrayDestroy(0) failed: %v", err)
	}
	if _, _, err := comkit.SafeArrayBounds(0); err == nil {
		t.Fatal("expected error for SafeArrayBounds(0)")
	}
	if res, err := comkit.AppendSafeArrayInt32(nil, 0); err != nil || len(res) != 0 {
		t.Fatalf("AppendSafeArrayInt32(0) = (%v, %v)", res, err)
	}
	err = comkit.CoSetProxyBlanket(
		0,
		comkit.RPCAuthnWinNT,
		comkit.RPCAuthzNone,
		0,
		comkit.RPCAuthnLevelDefault,
		comkit.RPCImpLevelImpersonate,
		0,
		comkit.EOACNone,
	)
	if err == nil {
		t.Fatal("expected error for CoSetProxyBlanket(0, ...)")
	}
}
