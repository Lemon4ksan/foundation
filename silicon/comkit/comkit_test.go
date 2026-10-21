// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package comkit_test

import (
	"crypto/md5"
	"errors"
	"runtime"
	"syscall"
	"testing"
	"unsafe"

	"github.com/lemon4ksan/foundation/silicon/comkit"
)

func TestGUID_FormatAndParse(t *testing.T) {
	if !(comkit.GUID{}).IsZero() {
		t.Fatal("expected zero GUID to report IsZero() == true")
	}
	if comkit.IIDIUnknown.IsZero() {
		t.Fatal("expected IIDIUnknown to be non-zero")
	}
	if !comkit.IIDIUnknown.Equal(comkit.IIDIUnknown) {
		t.Fatal("expected IIDIUnknown.Equal(IIDIUnknown) == true")
	}
	if comkit.IIDIUnknown.Equal(comkit.IIDIInspectable) {
		t.Fatal("expected IIDIUnknown != IIDIInspectable")
	}

	want := "af86e2e0-b12d-4c6a-9c5a-d7aa65101e90"
	got := comkit.IIDIInspectable.String()
	if got != want {
		t.Fatalf("IIDIInspectable.String() = %q, want %q", got, want)
	}

	parsed, err := comkit.ParseGUID(want)
	if err != nil {
		t.Fatalf("ParseGUID(%q) failed: %v", want, err)
	}
	if !parsed.Equal(comkit.IIDIInspectable) {
		t.Fatalf("parsed GUID %v != %v", parsed, comkit.IIDIInspectable)
	}

	parsedBraced, err := comkit.ParseGUID("{AF86E2E0-B12D-4C6A-9C5A-D7AA65101E90}")
	if err != nil {
		t.Fatalf("ParseGUID braced uppercase failed: %v", err)
	}
	if !parsedBraced.Equal(comkit.IIDIInspectable) {
		t.Fatalf("parsedBraced GUID %v != %v", parsedBraced, comkit.IIDIInspectable)
	}

	invalidCases := []string{
		"",
		"too-short",
		"af86e2e0_b12d-4c6a-9c5a-d7aa65101e90",
		"zz86e2e0-b12d-4c6a-9c5a-d7aa65101e90",
		"af86e2e0-b12d-4c6a-9c5a-d7aa65101e9z",
	}
	for _, tc := range invalidCases {
		if _, err := comkit.ParseGUID(tc); err == nil {
			t.Errorf("ParseGUID(%q) succeeded, expected error", tc)
		}
	}
}

func TestGUID_MD5Derivation(t *testing.T) {
	sum := md5.Sum([]byte("S-1-5-21-test-container"))
	g := comkit.GUIDFromMD5Bytes(sum)
	if g.IsZero() {
		t.Fatal("expected non-zero GUID from MD5 digest")
	}
	reparsed, err := comkit.ParseGUID(g.String())
	if err != nil || !reparsed.Equal(g) {
		t.Fatalf("roundtrip MD5 GUID failed: %v, err=%v", reparsed, err)
	}
}

func TestGUID_ZeroAllocs(t *testing.T) {
	var dst [64]byte
	g := comkit.IIDIInspectable
	s := "{AF86E2E0-B12D-4C6A-9C5A-D7AA65101E90}"

	allocsAppend := testing.AllocsPerRun(1000, func() {
		res := g.AppendText(dst[:0])
		if len(res) != 36 {
			t.Fatal("unexpected length")
		}
	})
	if allocsAppend != 0 {
		t.Fatalf("GUID.AppendText allocated %v times, want 0", allocsAppend)
	}

	allocsParse := testing.AllocsPerRun(1000, func() {
		_, err := comkit.ParseGUID(s)
		if err != nil {
			t.Fatal(err)
		}
	})
	if allocsParse != 0 {
		t.Fatalf("ParseGUID allocated %v times, want 0", allocsParse)
	}
}

func TestHRESULT_And_NTSTATUS(t *testing.T) {
	if !comkit.S_OK.Succeeded() || comkit.S_OK.Failed() {
		t.Fatal("S_OK must succeed")
	}
	if !comkit.S_FALSE.Succeeded() || comkit.S_FALSE.Failed() {
		t.Fatal("S_FALSE must succeed")
	}
	if comkit.S_OK.Err("noop") != nil {
		t.Fatal("S_OK.Err must be nil")
	}
	if comkit.CheckHR("noop", 0) != nil {
		t.Fatal("CheckHR(0) must be nil")
	}

	if !comkit.E_ACCESSDENIED.Failed() || comkit.E_ACCESSDENIED.Succeeded() {
		t.Fatal("E_ACCESSDENIED must fail")
	}
	if comkit.E_ACCESSDENIED.Uint32() != 0x80070005 {
		t.Fatalf("E_ACCESSDENIED.Uint32() = 0x%X, want 0x80070005", comkit.E_ACCESSDENIED.Uint32())
	}
	if comkit.E_ACCESSDENIED.Win32Error() != syscall.Errno(5) {
		t.Fatalf("E_ACCESSDENIED.Win32Error() = %v, want 5", comkit.E_ACCESSDENIED.Win32Error())
	}
	if comkit.E_NOINTERFACE.Win32Error() != syscall.Errno(0x80004002) {
		t.Fatalf("E_NOINTERFACE.Win32Error() = 0x%X, want 0x80004002", uint32(comkit.E_NOINTERFACE.Win32Error()))
	}

	err := comkit.CheckHR("QueryInterface", comkit.E_ACCESSDENIED.Uintptr())
	if err == nil {
		t.Fatal("expected error for E_ACCESSDENIED")
	}
	var hrErr *comkit.HresultError
	if !errors.As(err, &hrErr) || hrErr.Hr != comkit.E_ACCESSDENIED {
		t.Fatalf("expected *HresultError with E_ACCESSDENIED, got %v", err)
	}
	if !errors.Is(err, syscall.Errno(5)) {
		t.Fatalf("expected errors.Is(err, syscall.Errno(5)) to be true, got %v", err)
	}
	if got := err.Error(); got != "QueryInterface failed with HRESULT 0x80070005" {
		t.Fatalf("unexpected error string: %q", got)
	}

	// NTSTATUS checks
	if comkit.CheckNTStatus("NtSubscribeWnfStateChange", 0) != nil {
		t.Fatal("expected nil for NTSTATUS 0")
	}
	stSuccess := comkit.NTSTATUS(0)
	if !stSuccess.Succeeded() || stSuccess.Failed() {
		t.Fatal("NTSTATUS(0) must succeed")
	}
	ntErr := comkit.CheckNTStatus("NtSubscribeWnfStateChange", 0xC0000022)
	if ntErr == nil {
		t.Fatal("expected error for NTSTATUS 0xC0000022")
	}
	if got := ntErr.Error(); got != "NtSubscribeWnfStateChange failed with NTSTATUS 0xC0000022" {
		t.Fatalf("unexpected NTSTATUS error string: %q", got)
	}
}

func TestVARIANT_LayoutAndAccessors(t *testing.T) {
	if sz := unsafe.Sizeof(comkit.VARIANT{}); sz != 24 {
		t.Fatalf("sizeof(VARIANT) = %d, want 24 bytes (64-bit Windows ABI)", sz)
	}

	vEmpty := comkit.NewVariantEmpty()
	if vEmpty.VT != comkit.VTEmpty {
		t.Fatalf("expected VTEmpty, got %v", vEmpty.VT)
	}

	vI4 := comkit.NewVariantI4(-42)
	if vI4.VT != comkit.VTI4 || vI4.Int32() != -42 {
		t.Fatalf("NewVariantI4(-42) = (%v, %d)", vI4.VT, vI4.Int32())
	}

	vI8 := comkit.NewVariantI8(-9876543210)
	if vI8.VT != comkit.VTI8 || vI8.Int64() != -9876543210 {
		t.Fatalf("NewVariantI8 = (%v, %d)", vI8.VT, vI8.Int64())
	}

	vTrue := comkit.NewVariantBool(true)
	if vTrue.VT != comkit.VTBool || !vTrue.Bool() || vTrue.Val1 != 0xFFFF {
		t.Fatalf("NewVariantBool(true) = %+v", vTrue)
	}

	vFalse := comkit.NewVariantBool(false)
	if vFalse.VT != comkit.VTBool || vFalse.Bool() {
		t.Fatalf("NewVariantBool(false) = %+v", vFalse)
	}

	vBstr := comkit.NewVariantBSTR(comkit.BSTR(0x12345678))
	if vBstr.VT != comkit.VTBSTR || vBstr.Ptr() != 0x12345678 {
		t.Fatalf("NewVariantBSTR = %+v", vBstr)
	}

	vUnk := comkit.NewVariantUnknown(0xABCDEF00)
	if vUnk.VT != comkit.VTUnknown || vUnk.Ptr() != 0xABCDEF00 {
		t.Fatalf("NewVariantUnknown = %+v", vUnk)
	}
}

func TestVtblSlot_And_Ptr(t *testing.T) {
	pNil := comkit.NewPtr(0)
	if !pNil.IsNil() || pNil.Raw() != 0 {
		t.Fatal("expected nil Ptr")
	}
	if comkit.VtblSlot(0, 0) != 0 {
		t.Fatal("expected 0 slot for nil instance")
	}
	if comkit.VtblSlot(0x1000, -1) != 0 {
		t.Fatal("expected 0 slot for negative index")
	}

	var zeroVtbl uintptr
	if comkit.VtblSlot(uintptr(unsafe.Pointer(&zeroVtbl)), 0) != 0 {
		t.Fatal("expected 0 when vtable pointer is zero")
	}

	fakeSlots := [4]uintptr{0x1111, 0x2222, 0x3333, 0x4444}
	vtblPtr := uintptr(unsafe.Pointer(&fakeSlots[0]))
	instance := uintptr(unsafe.Pointer(&vtblPtr))

	for i, want := range fakeSlots {
		if got := comkit.VtblSlot(instance, i); got != want {
			t.Fatalf("VtblSlot(%d) = 0x%X, want 0x%X", i, got, want)
		}
	}

	allocs := testing.AllocsPerRun(1000, func() {
		_ = comkit.VtblSlot(instance, 2)
	})
	runtime.KeepAlive(&fakeSlots)
	runtime.KeepAlive(&vtblPtr)
	if allocs != 0 {
		t.Fatalf("VtblSlot allocated %v times, want 0", allocs)
	}

	// Test AppendUTF16AsUTF8 with ASCII, surrogate pair (U+1F680), and NUL terminator.
	u16 := []uint16{'H', 'i', ' ', 0xD83D, 0xDE80, 0, 'X'}
	utf8Out := comkit.AppendUTF16AsUTF8(nil, u16)
	if string(utf8Out) != "Hi 🚀" {
		t.Fatalf("AppendUTF16AsUTF8 = %q, want %q", string(utf8Out), "Hi 🚀")
	}
}
