// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !windows

package comkit

import (
	"errors"
	"unsafe"
)

var errUnsupportedOS = errors.New("comkit: Windows COM/WinRT ABI is not supported on this operating system")

// VtblCall0 is a non-Windows stub returning E_NOTIMPL.
func VtblCall0(instance uintptr, slot int) HRESULT {
	if instance == 0 || slot < 0 {
		return E_POINTER
	}
	return E_NOTIMPL
}

// VtblCall1 is a non-Windows stub returning E_NOTIMPL.
func VtblCall1(instance uintptr, slot int, _ uintptr) HRESULT {
	return VtblCall0(instance, slot)
}

// VtblCall2 is a non-Windows stub returning E_NOTIMPL.
func VtblCall2(instance uintptr, slot int, _, _ uintptr) HRESULT {
	return VtblCall0(instance, slot)
}

// VtblCall3 is a non-Windows stub returning E_NOTIMPL.
func VtblCall3(instance uintptr, slot int, _, _, _ uintptr) HRESULT {
	return VtblCall0(instance, slot)
}

// VtblCall4 is a non-Windows stub returning E_NOTIMPL.
func VtblCall4(instance uintptr, slot int, _, _, _, _ uintptr) HRESULT {
	return VtblCall0(instance, slot)
}

// VtblCall5 is a non-Windows stub returning E_NOTIMPL.
func VtblCall5(instance uintptr, slot int, _, _, _, _, _ uintptr) HRESULT {
	return VtblCall0(instance, slot)
}

// VtblCall6 is a non-Windows stub returning E_NOTIMPL.
func VtblCall6(instance uintptr, slot int, _, _, _, _, _, _ uintptr) HRESULT {
	return VtblCall0(instance, slot)
}

// VtblCall7 is a non-Windows stub returning E_NOTIMPL.
func VtblCall7(instance uintptr, slot int, _, _, _, _, _, _, _ uintptr) HRESULT {
	return VtblCall0(instance, slot)
}

// VtblCall8 is a non-Windows stub returning E_NOTIMPL.
func VtblCall8(instance uintptr, slot int, _, _, _, _, _, _, _, _ uintptr) HRESULT {
	return VtblCall0(instance, slot)
}

// VtblCall9 is a non-Windows stub returning E_NOTIMPL.
func VtblCall9(instance uintptr, slot int, _, _, _, _, _, _, _, _, _ uintptr) HRESULT {
	return VtblCall0(instance, slot)
}

// QueryInterface is a non-Windows stub.
func (p Ptr) QueryInterface(iid *GUID) (Ptr, HRESULT) {
	if p.raw == 0 || iid == nil {
		return Ptr{}, E_POINTER
	}
	return Ptr{}, E_NOTIMPL
}

// AddRef is a non-Windows stub.
func (p Ptr) AddRef() uint32 { return 0 }

// Release is a non-Windows stub.
func (p Ptr) Release() uint32 { return 0 }

// ComRelease is a non-Windows stub.
func ComRelease(_ uintptr) uint32 { return 0 }

// CoInitializeEx is a non-Windows stub.
func CoInitializeEx(_ uint32) error { return errUnsupportedOS }

// CoUninitialize is a non-Windows stub.
func CoUninitialize() {}

// RoInitialize is a non-Windows stub.
func RoInitialize(_ uint32) error { return errUnsupportedOS }

// LockThreadMTA is a non-Windows stub.
func LockThreadMTA() (func(), error) { return func() {}, errUnsupportedOS }

// CoCreateInstance is a non-Windows stub.
func CoCreateInstance(_ *GUID, _ uintptr, _ uint32, _ *GUID) (Ptr, error) {
	return Ptr{}, errUnsupportedOS
}

// CoSetProxyBlanket is a non-Windows stub.
func CoSetProxyBlanket(_ uintptr, _, _ uint32, _ uintptr, _, _ uint32, _ uintptr, _ uint32) error {
	return errUnsupportedOS
}

// VariantClear is a non-Windows stub.
func VariantClear(v *VARIANT) error {
	if v != nil {
		*v = VARIANT{}
	}
	return nil
}

// SafeArrayCreateVector is a non-Windows stub.
func SafeArrayCreateVector(_ VARTYPE, _ int32, _ uint32) SAFEARRAY { return 0 }

// SafeArrayDestroy is a non-Windows stub.
func SafeArrayDestroy(_ SAFEARRAY) error { return nil }

// SafeArrayBounds is a non-Windows stub.
func SafeArrayBounds(_ SAFEARRAY) (int32, int32, error) { return 0, -1, errUnsupportedOS }

// AppendSafeArrayInt32 is a non-Windows stub.
func AppendSafeArrayInt32(dst []int32, _ SAFEARRAY) ([]int32, error) { return dst, nil }

// RoGetActivationFactory is a non-Windows stub.
func RoGetActivationFactory(_ string, _ *GUID) (Ptr, error) {
	return Ptr{}, errUnsupportedOS
}

// RoActivateInstance is a non-Windows stub.
func RoActivateInstance(_ string) (Ptr, error) {
	return Ptr{}, errUnsupportedOS
}

// AllocBSTR is a non-Windows stub.
func AllocBSTR(_ string) BSTR { return 0 }

// FreeBSTR is a non-Windows stub.
func FreeBSTR(_ BSTR) {}

// AppendBSTRUTF8 is a non-Windows stub.
func AppendBSTRUTF8(dst []byte, _ BSTR) []byte { return dst }

// BSTRToString is a non-Windows stub.
func BSTRToString(_ BSTR) string { return "" }

// NewHString is a non-Windows stub.
func NewHString(_ string) (HSTRING, error) { return 0, errUnsupportedOS }

// DeleteHString is a non-Windows stub.
func DeleteHString(_ HSTRING) {}

// AppendHStringUTF8 is a non-Windows stub.
func AppendHStringUTF8(dst []byte, _ HSTRING) []byte { return dst }

// CoTaskMemAlloc is a non-Windows stub.
func CoTaskMemAlloc(_ uintptr) unsafe.Pointer { return nil }

// CoTaskMemFree is a non-Windows stub.
func CoTaskMemFree(_ unsafe.Pointer) {}

// AppendCoTaskUTF8 is a non-Windows stub.
func AppendCoTaskUTF8(dst []byte, ptr *uint16) []byte {
	if ptr == nil {
		return dst
	}
	return appendUTF16AsUTF8(dst, nil)
}

// DrainCoTaskUTF16 is a non-Windows stub.
func DrainCoTaskUTF16(_ *uint16) string { return "" }
