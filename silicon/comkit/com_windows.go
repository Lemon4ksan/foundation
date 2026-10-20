// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows

package comkit

import (
	"runtime"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modOle32    = windows.NewLazySystemDLL("ole32.dll")
	modOleAut32 = windows.NewLazySystemDLL("oleaut32.dll")
	modCombase  = windows.NewLazySystemDLL("combase.dll")

	procCoInitializeEx            = modOle32.NewProc("CoInitializeEx")
	procCoUninitialize            = modOle32.NewProc("CoUninitialize")
	procCoCreateInstance          = modOle32.NewProc("CoCreateInstance")
	procCoSetProxyBlanket         = modOle32.NewProc("CoSetProxyBlanket")
	procCoTaskMemAlloc            = modOle32.NewProc("CoTaskMemAlloc")
	procCoTaskMemFree             = modOle32.NewProc("CoTaskMemFree")
	procSysAllocStringLen         = modOleAut32.NewProc("SysAllocStringLen")
	procSysFreeString             = modOleAut32.NewProc("SysFreeString")
	procVariantClear              = modOleAut32.NewProc("VariantClear")
	procSafeArrayCreateVector     = modOleAut32.NewProc("SafeArrayCreateVector")
	procSafeArrayDestroy          = modOleAut32.NewProc("SafeArrayDestroy")
	procSafeArrayGetLBound        = modOleAut32.NewProc("SafeArrayGetLBound")
	procSafeArrayGetUBound        = modOleAut32.NewProc("SafeArrayGetUBound")
	procSafeArrayAccessData       = modOleAut32.NewProc("SafeArrayAccessData")
	procSafeArrayUnaccessData     = modOleAut32.NewProc("SafeArrayUnaccessData")
	procRoInitialize              = modCombase.NewProc("RoInitialize")
	procRoActivateInstance        = modCombase.NewProc("RoActivateInstance")
	procRoGetActivationFactory    = modCombase.NewProc("RoGetActivationFactory")
	procWindowsCreateString       = modCombase.NewProc("WindowsCreateString")
	procWindowsDeleteString       = modCombase.NewProc("WindowsDeleteString")
	procWindowsGetStringRawBuffer = modCombase.NewProc("WindowsGetStringRawBuffer")
)

// VtblCall0 invokes the C++ virtual method at `slot` with 0 explicit arguments (only `this`).
// It performs zero heap allocations.
func VtblCall0(instance uintptr, slot int) HRESULT {
	fn := VtblSlot(instance, slot)
	if fn == 0 {
		return E_POINTER
	}
	r1, _, _ := syscall.SyscallN(fn, instance)
	return HRESULT(int32(r1))
}

// VtblCall1 invokes the C++ virtual method at `slot` with 1 argument after `this`.
func VtblCall1(instance uintptr, slot int, a1 uintptr) HRESULT {
	fn := VtblSlot(instance, slot)
	if fn == 0 {
		return E_POINTER
	}
	r1, _, _ := syscall.SyscallN(fn, instance, a1)
	return HRESULT(int32(r1))
}

// VtblCall2 invokes the C++ virtual method at `slot` with 2 arguments after `this`.
func VtblCall2(instance uintptr, slot int, a1, a2 uintptr) HRESULT {
	fn := VtblSlot(instance, slot)
	if fn == 0 {
		return E_POINTER
	}
	r1, _, _ := syscall.SyscallN(fn, instance, a1, a2)
	return HRESULT(int32(r1))
}

// VtblCall3 invokes the C++ virtual method at `slot` with 3 arguments after `this`.
func VtblCall3(instance uintptr, slot int, a1, a2, a3 uintptr) HRESULT {
	fn := VtblSlot(instance, slot)
	if fn == 0 {
		return E_POINTER
	}
	r1, _, _ := syscall.SyscallN(fn, instance, a1, a2, a3)
	return HRESULT(int32(r1))
}

// VtblCall4 invokes the C++ virtual method at `slot` with 4 arguments after `this`.
func VtblCall4(instance uintptr, slot int, a1, a2, a3, a4 uintptr) HRESULT {
	fn := VtblSlot(instance, slot)
	if fn == 0 {
		return E_POINTER
	}
	r1, _, _ := syscall.SyscallN(fn, instance, a1, a2, a3, a4)
	return HRESULT(int32(r1))
}

// VtblCall5 invokes the C++ virtual method at `slot` with 5 arguments after `this`.
func VtblCall5(instance uintptr, slot int, a1, a2, a3, a4, a5 uintptr) HRESULT {
	fn := VtblSlot(instance, slot)
	if fn == 0 {
		return E_POINTER
	}
	r1, _, _ := syscall.SyscallN(fn, instance, a1, a2, a3, a4, a5)
	return HRESULT(int32(r1))
}

// VtblCall6 invokes the C++ virtual method at `slot` with 6 arguments after `this`.
func VtblCall6(instance uintptr, slot int, a1, a2, a3, a4, a5, a6 uintptr) HRESULT {
	fn := VtblSlot(instance, slot)
	if fn == 0 {
		return E_POINTER
	}
	r1, _, _ := syscall.SyscallN(fn, instance, a1, a2, a3, a4, a5, a6)
	return HRESULT(int32(r1))
}

// VtblCall7 invokes the C++ virtual method at `slot` with 7 arguments after `this`.
func VtblCall7(instance uintptr, slot int, a1, a2, a3, a4, a5, a6, a7 uintptr) HRESULT {
	fn := VtblSlot(instance, slot)
	if fn == 0 {
		return E_POINTER
	}
	r1, _, _ := syscall.SyscallN(fn, instance, a1, a2, a3, a4, a5, a6, a7)
	return HRESULT(int32(r1))
}

// VtblCall8 invokes the C++ virtual method at `slot` with 8 arguments after `this`.
func VtblCall8(instance uintptr, slot int, a1, a2, a3, a4, a5, a6, a7, a8 uintptr) HRESULT {
	fn := VtblSlot(instance, slot)
	if fn == 0 {
		return E_POINTER
	}
	r1, _, _ := syscall.SyscallN(fn, instance, a1, a2, a3, a4, a5, a6, a7, a8)
	return HRESULT(int32(r1))
}

// VtblCall9 invokes the C++ virtual method at `slot` with 9 arguments after `this`
// (e.g. ID3D11DeviceContext::CopySubresourceRegion).
func VtblCall9(instance uintptr, slot int, a1, a2, a3, a4, a5, a6, a7, a8, a9 uintptr) HRESULT {
	fn := VtblSlot(instance, slot)
	if fn == 0 {
		return E_POINTER
	}
	r1, _, _ := syscall.SyscallN(fn, instance, a1, a2, a3, a4, a5, a6, a7, a8, a9)
	return HRESULT(int32(r1))
}

// QueryInterface queries the COM object for the interface identified by `iid` (IUnknown slot 0).
func (p Ptr) QueryInterface(iid *GUID) (Ptr, HRESULT) {
	if p.raw == 0 || iid == nil {
		return Ptr{}, E_POINTER
	}
	var out uintptr
	hr := VtblCall2(p.raw, 0, uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&out)))
	return Ptr{raw: out}, hr
}

// AddRef increments the COM object's reference count (IUnknown slot 1) and returns the new count.
func (p Ptr) AddRef() uint32 {
	fn := VtblSlot(p.raw, 1)
	if fn == 0 {
		return 0
	}
	r1, _, _ := syscall.SyscallN(fn, p.raw)
	return uint32(r1)
}

// Release decrements the COM object's reference count (IUnknown slot 2) and returns the remaining count.
func (p Ptr) Release() uint32 {
	fn := VtblSlot(p.raw, 2)
	if fn == 0 {
		return 0
	}
	r1, _, _ := syscall.SyscallN(fn, p.raw)
	return uint32(r1)
}

// ComRelease safely decrements the reference count of a raw COM interface pointer if non-zero.
func ComRelease(instance uintptr) uint32 {
	if instance == 0 {
		return 0
	}
	return Ptr{raw: instance}.Release()
}

// CoInitializeEx initializes the COM library on the current OS thread.
// RPC_E_CHANGED_MODE (0x80010106) and S_FALSE (1) are treated as benign success.
func CoInitializeEx(coInit uint32) error {
	r1, _, _ := syscall.SyscallN(procCoInitializeEx.Addr(), 0, uintptr(coInit))
	hr := HRESULT(int32(r1))
	if hr.Failed() && uint32(hr) != 0x80010106 {
		return hr.Err("CoInitializeEx")
	}
	return nil
}

// CoUninitialize closes the COM library on the current OS thread.
func CoUninitialize() {
	_, _, _ = syscall.SyscallN(procCoUninitialize.Addr())
}

// RoInitialize initializes the Windows Runtime (WinRT) on the current OS thread.
// RPC_E_CHANGED_MODE (0x80010106) and S_FALSE (1) are treated as benign success.
func RoInitialize(initType uint32) error {
	r1, _, _ := syscall.SyscallN(procRoInitialize.Addr(), uintptr(initType))
	hr := HRESULT(int32(r1))
	if hr.Failed() && uint32(hr) != 0x80010106 {
		return hr.Err("RoInitialize")
	}
	return nil
}

// LockThreadMTA pins the calling goroutine to its OS thread via runtime.LockOSThread,
// ensures COM is initialized in the Multi-Threaded Apartment (COINIT_MULTITHREADED),
// and returns an unlock function to be deferred by the caller.
func LockThreadMTA() (func(), error) {
	runtime.LockOSThread()
	if err := CoInitializeEx(CoInitMultithreaded); err != nil {
		runtime.UnlockOSThread()
		return func() {}, err
	}
	return runtime.UnlockOSThread, nil
}

// CoCreateInstance creates a single uninitialized object of the class associated with `clsid`.
func CoCreateInstance(clsid *GUID, outer uintptr, clsContext uint32, iid *GUID) (Ptr, error) {
	if clsid == nil || iid == nil {
		return Ptr{}, E_POINTER.Err("CoCreateInstance")
	}
	var instance uintptr
	r1, _, _ := syscall.SyscallN(
		procCoCreateInstance.Addr(),
		uintptr(unsafe.Pointer(clsid)),
		outer,
		uintptr(clsContext),
		uintptr(unsafe.Pointer(iid)),
		uintptr(unsafe.Pointer(&instance)),
	)
	if err := CheckHR("CoCreateInstance", r1); err != nil {
		return Ptr{}, err
	}
	return Ptr{raw: instance}, nil
}

// CoSetProxyBlanket configures RPC authentication and impersonation security on a COM proxy
// (required for out-of-process `CLSCTX_LOCAL_SERVER` COM objects such as WMI, VSS, Task Scheduler, and SandboxUDK).
func CoSetProxyBlanket(
	proxy uintptr,
	authnSvc, authzSvc uint32,
	serverPrincName uintptr,
	authnLevel, impLevel uint32,
	authInfo uintptr,
	capabilities uint32,
) error {
	if proxy == 0 {
		return E_POINTER.Err("CoSetProxyBlanket")
	}
	r1, _, _ := syscall.SyscallN(
		procCoSetProxyBlanket.Addr(),
		proxy,
		uintptr(authnSvc),
		uintptr(authzSvc),
		serverPrincName,
		uintptr(authnLevel),
		uintptr(impLevel),
		authInfo,
		uintptr(capabilities),
	)
	return CheckHR("CoSetProxyBlanket", r1)
}

// VariantClear releases any resources (`BSTR`, `SAFEARRAY`, `IUnknown`) owned by `v`
// and resets `v.VT` to `VTEmpty` via `oleaut32!VariantClear`.
func VariantClear(v *VARIANT) error {
	if v == nil {
		return nil
	}
	r1, _, _ := syscall.SyscallN(procVariantClear.Addr(), uintptr(unsafe.Pointer(v)))
	return CheckHR("VariantClear", r1)
}

// SafeArrayCreateVector allocates a 1-dimensional contiguous `SAFEARRAY` of `vt` elements.
func SafeArrayCreateVector(vt VARTYPE, lbound int32, count uint32) SAFEARRAY {
	r1, _, _ := syscall.SyscallN(
		procSafeArrayCreateVector.Addr(),
		uintptr(vt),
		uintptr(uint32(lbound)),
		uintptr(count),
	)
	return SAFEARRAY(r1)
}

// SafeArrayDestroy releases a `SAFEARRAY` and all elements it owns via `oleaut32!SafeArrayDestroy`.
func SafeArrayDestroy(psa SAFEARRAY) error {
	if psa == 0 {
		return nil
	}
	r1, _, _ := syscall.SyscallN(procSafeArrayDestroy.Addr(), uintptr(psa))
	return CheckHR("SafeArrayDestroy", r1)
}

// SafeArrayBounds returns the lower and upper bounds of dimension 1 of `psa`.
func SafeArrayBounds(psa SAFEARRAY) (int32, int32, error) {
	if psa == 0 {
		return 0, -1, E_POINTER.Err("SafeArrayBounds")
	}
	var lb, ub int32
	r1, _, _ := syscall.SyscallN(procSafeArrayGetLBound.Addr(), uintptr(psa), 1, uintptr(unsafe.Pointer(&lb)))
	if err := CheckHR("SafeArrayGetLBound", r1); err != nil {
		return 0, 0, err
	}
	r2, _, _ := syscall.SyscallN(procSafeArrayGetUBound.Addr(), uintptr(psa), 1, uintptr(unsafe.Pointer(&ub)))
	if err := CheckHR("SafeArrayGetUBound", r2); err != nil {
		return 0, 0, err
	}
	return lb, ub, nil
}

// AppendSafeArrayInt32 appends all 32-bit signed integers from a 1D `VT_I4`/`VT_INT` `SAFEARRAY` into `dst`
// without heap allocations when `dst` has sufficient capacity.
func AppendSafeArrayInt32(dst []int32, psa SAFEARRAY) ([]int32, error) {
	if psa == 0 {
		return dst, nil
	}
	lb, ub, err := SafeArrayBounds(psa)
	if err != nil {
		return dst, err
	}
	count := int(ub - lb + 1)
	if count <= 0 {
		return dst, nil
	}
	var rawData unsafe.Pointer
	r1, _, _ := syscall.SyscallN(procSafeArrayAccessData.Addr(), uintptr(psa), uintptr(unsafe.Pointer(&rawData)))
	if err := CheckHR("SafeArrayAccessData", r1); err != nil {
		return dst, err
	}
	defer func() {
		_, _, _ = syscall.SyscallN(procSafeArrayUnaccessData.Addr(), uintptr(psa))
	}()
	src := unsafe.Slice((*int32)(rawData), count)
	return append(dst, src...), nil
}

// RoGetActivationFactory retrieves the WinRT activation factory for `activatableClassID`
// implementing interface `iid` via `combase.dll!RoGetActivationFactory`.
func RoGetActivationFactory(activatableClassID string, iid *GUID) (Ptr, error) {
	if iid == nil {
		return Ptr{}, E_POINTER.Err("RoGetActivationFactory")
	}
	if err := RoInitialize(RoInitMultithreaded); err != nil {
		return Ptr{}, err
	}
	hs, err := NewHString(activatableClassID)
	if err != nil {
		return Ptr{}, err
	}
	defer DeleteHString(hs)

	var factory uintptr
	r1, _, _ := syscall.SyscallN(
		procRoGetActivationFactory.Addr(),
		uintptr(hs),
		uintptr(unsafe.Pointer(iid)),
		uintptr(unsafe.Pointer(&factory)),
	)
	if err := CheckHR("RoGetActivationFactory", r1); err != nil {
		return Ptr{}, err
	}
	return Ptr{raw: factory}, nil
}

// RoActivateInstance activates a WinRT runtime class by its full class name.
func RoActivateInstance(activatableClassID string) (Ptr, error) {
	if err := RoInitialize(RoInitMultithreaded); err != nil {
		return Ptr{}, err
	}
	hs, err := NewHString(activatableClassID)
	if err != nil {
		return Ptr{}, err
	}
	defer DeleteHString(hs)

	var instance uintptr
	r1, _, _ := syscall.SyscallN(
		procRoActivateInstance.Addr(),
		uintptr(hs),
		uintptr(unsafe.Pointer(&instance)),
	)
	if err := CheckHR("RoActivateInstance", r1); err != nil {
		return Ptr{}, err
	}
	return Ptr{raw: instance}, nil
}

// AllocBSTR allocates an OLE Automation BSTR from a Go UTF-8 string using `SysAllocStringLen`.
// For short strings (<= 120 UTF-16 code units), transcoding occurs on a stack buffer with zero Go heap allocations.
func AllocBSTR(s string) BSTR {
	if s == "" {
		r1, _, _ := syscall.SyscallN(procSysAllocStringLen.Addr(), 0, 0)
		return BSTR(r1)
	}
	var stackBuf [128]uint16
	u16 := encodeUTF16Into(stackBuf[:0], s)
	r1, _, _ := syscall.SyscallN(
		procSysAllocStringLen.Addr(),
		uintptr(unsafe.Pointer(&u16[0])),
		uintptr(len(u16)),
	)
	return BSTR(r1)
}

// FreeBSTR releases an OLE Automation BSTR via `SysFreeString`.
func FreeBSTR(b BSTR) {
	if b != 0 {
		_, _, _ = syscall.SyscallN(procSysFreeString.Addr(), uintptr(b))
	}
}

// bstrLen returns the number of UTF-16 code units in `b` by reading the 32-bit byte-length
// prefix immediately preceding the BSTR payload (`[b - 4] / 2`), per the MS-OAUT BSTR ABI.
func bstrLen(b BSTR) int {
	if b == 0 {
		return 0
	}
	byteLen := *(*uint32)(ptrFromUintptr(uintptr(b) - 4))
	return int(byteLen / 2)
}

// AppendBSTRUTF8 appends the UTF-8 contents of `b` to `dst` without freeing `b`
// and with zero heap allocations when `dst` has sufficient capacity.
func AppendBSTRUTF8(dst []byte, b BSTR) []byte {
	length := bstrLen(b)
	if length == 0 {
		return dst
	}
	u16 := unsafe.Slice((*uint16)(ptrFromUintptr(uintptr(b))), length)
	return appendUTF16AsUTF8(dst, u16)
}

// BSTRToString converts `b` into a Go string and immediately frees `b` via `SysFreeString`.
func BSTRToString(b BSTR) string {
	if b == 0 {
		return ""
	}
	defer FreeBSTR(b)
	length := bstrLen(b)
	if length == 0 {
		return ""
	}
	u16 := unsafe.Slice((*uint16)(ptrFromUintptr(uintptr(b))), length)
	return windows.UTF16ToString(u16)
}

// NewHString creates a WinRT immutable HSTRING from a Go string.
func NewHString(s string) (HSTRING, error) {
	if s == "" {
		return 0, nil
	}
	var stackBuf [128]uint16
	u16 := encodeUTF16Into(stackBuf[:0], s)
	var hs HSTRING
	r1, _, _ := syscall.SyscallN(
		procWindowsCreateString.Addr(),
		uintptr(unsafe.Pointer(&u16[0])),
		uintptr(len(u16)),
		uintptr(unsafe.Pointer(&hs)),
	)
	if err := CheckHR("WindowsCreateString", r1); err != nil {
		return 0, err
	}
	return hs, nil
}

// DeleteHString releases a WinRT HSTRING handle.
func DeleteHString(hs HSTRING) {
	if hs != 0 {
		_, _, _ = syscall.SyscallN(procWindowsDeleteString.Addr(), uintptr(hs))
	}
}

// AppendHStringUTF8 appends the UTF-8 representation of `hs` into `dst` without freeing `hs`.
func AppendHStringUTF8(dst []byte, hs HSTRING) []byte {
	if hs == 0 {
		return dst
	}
	var length uint32
	rawPtr, _, _ := syscall.SyscallN(
		procWindowsGetStringRawBuffer.Addr(),
		uintptr(hs),
		uintptr(unsafe.Pointer(&length)),
	)
	if rawPtr == 0 || length == 0 {
		return dst
	}
	u16 := unsafe.Slice((*uint16)(ptrFromUintptr(rawPtr)), int(length))
	return appendUTF16AsUTF8(dst, u16)
}

// CoTaskMemAlloc allocates `size` bytes from the COM task memory allocator (`ole32!CoTaskMemAlloc`).
func CoTaskMemAlloc(size uintptr) unsafe.Pointer {
	if size == 0 {
		return nil
	}
	r1, _, _ := syscall.SyscallN(procCoTaskMemAlloc.Addr(), size)
	return ptrFromUintptr(r1)
}

// CoTaskMemFree frees a memory block allocated via `CoTaskMemAlloc`.
func CoTaskMemFree(ptr unsafe.Pointer) {
	if ptr != nil {
		_, _, _ = syscall.SyscallN(procCoTaskMemFree.Addr(), uintptr(ptr))
	}
}

// AppendCoTaskUTF8 reads a NUL-terminated UTF-16 string allocated by COM (`CoTaskMemAlloc`),
// appends its UTF-8 bytes to `dst`, and unconditionally frees `ptr` via `CoTaskMemFree`.
func AppendCoTaskUTF8(dst []byte, ptr *uint16) []byte {
	if ptr == nil {
		return dst
	}
	defer CoTaskMemFree(unsafe.Pointer(ptr))
	const maxChars = 1 << 20
	u16 := unsafe.Slice(ptr, maxChars)
	for i, ch := range u16 {
		if ch == 0 {
			return appendUTF16AsUTF8(dst, u16[:i])
		}
	}
	return appendUTF16AsUTF8(dst, u16)
}

// DrainCoTaskUTF16 converts a NUL-terminated UTF-16 string allocated by COM into a Go string
// and unconditionally frees the underlying buffer via `CoTaskMemFree`.
func DrainCoTaskUTF16(ptr *uint16) string {
	if ptr == nil {
		return ""
	}
	defer CoTaskMemFree(unsafe.Pointer(ptr))
	return windows.UTF16PtrToString(ptr)
}

func encodeUTF16Into(dst []uint16, s string) []uint16 {
	for _, r := range s {
		if r < 0x10000 {
			dst = append(dst, uint16(r))
		} else {
			r1, r2 := utf16.EncodeRune(r)
			dst = append(dst, uint16(r1), uint16(r2))
		}
	}
	return dst
}
