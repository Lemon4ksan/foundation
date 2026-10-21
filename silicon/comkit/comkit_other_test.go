// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !windows

package comkit_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/silicon/comkit"
)

func TestNonWindowsStubs(t *testing.T) {
	if hr := comkit.VtblCall0(0, 0); hr != comkit.E_POINTER {
		t.Fatalf("expected E_POINTER for nil instance, got %v", hr)
	}
	if hr := comkit.VtblCall0(0x1000, -1); hr != comkit.E_POINTER {
		t.Fatalf("expected E_POINTER for negative slot, got %v", hr)
	}
	if hr := comkit.VtblCall0(0x1000, 0); hr != comkit.E_NOTIMPL {
		t.Fatalf("expected E_NOTIMPL on non-Windows, got %v", hr)
	}
	_ = comkit.VtblCall1(0x1000, 0, 1)
	_ = comkit.VtblCall2(0x1000, 0, 1, 2)
	_ = comkit.VtblCall3(0x1000, 0, 1, 2, 3)
	_ = comkit.VtblCall4(0x1000, 0, 1, 2, 3, 4)
	_ = comkit.VtblCall5(0x1000, 0, 1, 2, 3, 4, 5)
	_ = comkit.VtblCall6(0x1000, 0, 1, 2, 3, 4, 5, 6)
	_ = comkit.VtblCall7(0x1000, 0, 1, 2, 3, 4, 5, 6, 7)
	_ = comkit.VtblCall8(0x1000, 0, 1, 2, 3, 4, 5, 6, 7, 8)
	_ = comkit.VtblCall9(0x1000, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9)

	pZero := comkit.NewPtr(0)
	if _, hr := pZero.QueryInterface(&comkit.IIDIUnknown); hr != comkit.E_POINTER {
		t.Fatalf("expected E_POINTER for zero Ptr.QueryInterface, got %v", hr)
	}
	p := comkit.NewPtr(0x1000)
	if _, hr := p.QueryInterface(&comkit.IIDIUnknown); hr != comkit.E_NOTIMPL {
		t.Fatalf("expected E_NOTIMPL for non-Windows Ptr.QueryInterface, got %v", hr)
	}
	_ = p.AddRef()
	_ = p.Release()
	_ = comkit.ComRelease(0x1000)

	if err := comkit.CoInitializeEx(comkit.CoInitMultithreaded); err == nil {
		t.Fatal("expected error from CoInitializeEx on non-Windows")
	}
	comkit.CoUninitialize()
	if err := comkit.RoInitialize(comkit.RoInitMultithreaded); err == nil {
		t.Fatal("expected error from RoInitialize on non-Windows")
	}
	if unlock, err := comkit.LockThreadMTA(); err == nil {
		t.Fatal("expected error from LockThreadMTA on non-Windows")
	} else if unlock != nil {
		unlock()
	}

	_, _ = comkit.CoCreateInstance(&comkit.IIDIUnknown, 0, comkit.ClsctxAll, &comkit.IIDIUnknown)
	_ = comkit.CoSetProxyBlanket(0, 0, 0, 0, 0, 0, 0, 0)

	v := comkit.NewVariantI4(10)
	_ = comkit.VariantClear(&v)

	sa := comkit.SafeArrayCreateVector(comkit.VTI4, 0, 4)
	_, _, _ = comkit.SafeArrayBounds(sa)
	_, _ = comkit.AppendSafeArrayInt32(nil, sa)
	_ = comkit.SafeArrayDestroy(sa)

	_, _ = comkit.RoGetActivationFactory("Windows.Foundation.PropertyValue", &comkit.IIDIInspectable)
	_, _ = comkit.RoActivateInstance("Windows.Foundation.PropertyValue")

	bstr := comkit.AllocBSTR("test")
	_ = comkit.AppendBSTRUTF8(nil, bstr)
	_ = comkit.BSTRToString(bstr)
	comkit.FreeBSTR(bstr)

	hs, _ := comkit.NewHString("test")
	_ = comkit.AppendHStringUTF8(nil, hs)
	comkit.DeleteHString(hs)

	mem := comkit.CoTaskMemAlloc(16)
	comkit.CoTaskMemFree(mem)
	_ = comkit.AppendCoTaskUTF8(nil, nil)
	var ch uint16
	_ = comkit.AppendCoTaskUTF8(nil, &ch)
	_ = comkit.DrainCoTaskUTF16(nil)
}
