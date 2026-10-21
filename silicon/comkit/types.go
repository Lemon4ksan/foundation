// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package comkit

import (
	"errors"
	"syscall"
	"unicode/utf8"
	"unsafe"
)

// GUID represents a 128-bit globally unique identifier (CLSID / IID / UUID)
// with the exact binary layout expected by Win32, COM, and WinRT ABIs.
type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

// Well-known foundational COM and WinRT interface identifiers.
var (
	// IIDIUnknown is the root COM interface identifier ({00000000-0000-0000-C000-000000000046}).
	IIDIUnknown = GUID{
		Data1: 0x00000000,
		Data2: 0x0000,
		Data3: 0x0000,
		Data4: [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46},
	}

	// IIDIDispatch is the OLE Automation IDispatch interface identifier ({00020400-0000-0000-C000-000000000046}).
	IIDIDispatch = GUID{
		Data1: 0x00020400,
		Data2: 0x0000,
		Data3: 0x0000,
		Data4: [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46},
	}

	// IIDIInspectable is the base WinRT IInspectable interface identifier ({AF86E2E0-B12D-4C6A-9C5A-D7AA65101E90}).
	IIDIInspectable = GUID{
		Data1: 0xAF86E2E0,
		Data2: 0xB12D,
		Data3: 0x4C6A,
		Data4: [8]byte{0x9C, 0x5A, 0xD7, 0xAA, 0x65, 0x10, 0x1E, 0x90},
	}

	// IIDIClientSecurity controls COM proxy security blankets ({0000013D-0000-0000-C000-000000000046}).
	IIDIClientSecurity = GUID{
		Data1: 0x0000013D,
		Data2: 0x0000,
		Data3: 0x0000,
		Data4: [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46},
	}
)

// COM apartment and class context constants.
const (
	// CoInitMultithreaded initializes the thread for multi-threaded object concurrency (COINIT_MULTITHREADED).
	CoInitMultithreaded uint32 = 0x0
	// CoInitApartmentThreaded initializes the thread for single-threaded apartment concurrency (COINIT_APARTMENTTHREADED).
	CoInitApartmentThreaded uint32 = 0x2
	// CoInitDisableOLE1DDE disables DDE for OLE1 support (COINIT_DISABLE_OLE1DDE).
	CoInitDisableOLE1DDE uint32 = 0x4
	// CoInitSpeedOverMemory trades memory usage for higher RPC performance (COINIT_SPEED_OVER_MEMORY).
	CoInitSpeedOverMemory uint32 = 0x8

	// RoInitSingleThreaded initializes WinRT on the thread as STA (RO_INIT_SINGLETHREADED).
	RoInitSingleThreaded uint32 = 0
	// RoInitMultithreaded initializes WinRT on the thread as MTA (RO_INIT_MULTITHREADED).
	RoInitMultithreaded uint32 = 1

	// ClsctxInprocServer activates an in-process DLL COM server (CLSCTX_INPROC_SERVER).
	ClsctxInprocServer uint32 = 0x1
	// ClsctxInprocHandler activates an in-process handler (CLSCTX_INPROC_HANDLER).
	ClsctxInprocHandler uint32 = 0x2
	// ClsctxLocalServer activates an out-of-process EXE COM server (CLSCTX_LOCAL_SERVER).
	ClsctxLocalServer uint32 = 0x4
	// ClsctxRemoteServer activates a remote DCOM server (CLSCTX_REMOTE_SERVER).
	ClsctxRemoteServer uint32 = 0x10
	// ClsctxAll permits any in-proc, local, or remote server context (CLSCTX_ALL).
	ClsctxAll uint32 = ClsctxInprocServer | ClsctxInprocHandler | ClsctxLocalServer | ClsctxRemoteServer
)

// RPC authentication, authorization, and impersonation constants for CoSetProxyBlanket.
const (
	RPCAuthnNone              uint32 = 0
	RPCAuthnWinNT             uint32 = 10
	RPCAuthnDefault           uint32 = 0xFFFFFFFF
	RPCAuthzNone              uint32 = 0
	RPCAuthzDefault           uint32 = 0xFFFFFFFF
	RPCAuthnLevelDefault      uint32 = 0
	RPCAuthnLevelNone         uint32 = 1
	RPCAuthnLevelConnect      uint32 = 2
	RPCAuthnLevelCall         uint32 = 3
	RPCAuthnLevelPkt          uint32 = 4
	RPCAuthnLevelPktIntegrity uint32 = 5
	RPCAuthnLevelPktPrivacy   uint32 = 6
	RPCImpLevelDefault        uint32 = 0
	RPCImpLevelAnonymous      uint32 = 1
	RPCImpLevelIdentify       uint32 = 2
	RPCImpLevelImpersonate    uint32 = 3
	RPCImpLevelDelegate       uint32 = 4

	EOACNone            uint32 = 0x0
	EOACStaticCloaking  uint32 = 0x20
	EOACDynamicCloaking uint32 = 0x40

	COLEDefaultPrincipal uintptr = ^uintptr(0)
	COLEDefaultAuthInfo  uintptr = ^uintptr(0)
)

// Standard COM HRESULT codes.
const (
	S_OK               HRESULT = 0
	S_FALSE            HRESULT = 1
	E_NOTIMPL          HRESULT = -0x7FFFBFFF // 0x80004001
	E_NOINTERFACE      HRESULT = -0x7FFFBFFE // 0x80004002
	E_POINTER          HRESULT = -0x7FFFBFFD // 0x80004003
	E_ABORT            HRESULT = -0x7FFFBFFC // 0x80004004
	E_FAIL             HRESULT = -0x7FFFBFFB // 0x80004005
	E_ACCESSDENIED     HRESULT = -0x7FF8FFFB // 0x80070005
	E_INVALIDARG       HRESULT = -0x7FF8FFA9 // 0x80070057
	RPC_E_CHANGED_MODE HRESULT = -0x7FFEFefa // 0x80010106
)

const hexDigits = "0123456789abcdef"

// IsZero reports whether g is the all-zero GUID.
func (g GUID) IsZero() bool {
	return g == GUID{}
}

// Equal reports whether g and other represent the exact same 128-bit GUID.
func (g GUID) Equal(other GUID) bool {
	return g == other
}

// AppendText appends the canonical 36-byte lowercase hyphenated representation
// (`xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`) of g to dst without heap allocations.
func (g GUID) AppendText(dst []byte) []byte {
	var buf [36]byte
	putHex32(buf[0:8], g.Data1)
	buf[8] = '-'
	putHex16(buf[9:13], g.Data2)
	buf[13] = '-'
	putHex16(buf[14:18], g.Data3)
	buf[18] = '-'
	putHex8(buf[19:21], g.Data4[0])
	putHex8(buf[21:23], g.Data4[1])
	buf[23] = '-'
	putHex8(buf[24:26], g.Data4[2])
	putHex8(buf[26:28], g.Data4[3])
	putHex8(buf[28:30], g.Data4[4])
	putHex8(buf[30:32], g.Data4[5])
	putHex8(buf[32:34], g.Data4[6])
	putHex8(buf[34:36], g.Data4[7])
	return append(dst, buf[:]...)
}

// String returns the canonical lowercase `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx` string.
func (g GUID) String() string {
	var buf [36]byte
	res := g.AppendText(buf[:0])
	return string(res)
}

// ParseGUID parses a canonical 36-character (`xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`)
// or 38-character braced (`{xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx}`) GUID string
// with zero heap allocations.
func ParseGUID(s string) (GUID, error) {
	if len(s) == 38 && s[0] == '{' && s[37] == '}' {
		s = s[1:37]
	}
	if len(s) != 36 || s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return GUID{}, errors.New("comkit: invalid GUID format")
	}
	d1, ok1 := parseHex32(s[0:8])
	d2, ok2 := parseHex16(s[9:13])
	d3, ok3 := parseHex16(s[14:18])
	if !ok1 || !ok2 || !ok3 {
		return GUID{}, errors.New("comkit: invalid GUID hex digit")
	}
	var d4 [8]byte
	offsets := [8]int{19, 21, 24, 26, 28, 30, 32, 34}
	for i, off := range offsets {
		b, ok := parseHex8(s[off : off+2])
		if !ok {
			return GUID{}, errors.New("comkit: invalid GUID hex byte")
		}
		d4[i] = b
	}
	return GUID{Data1: d1, Data2: d2, Data3: d3, Data4: d4}, nil
}

// GUIDFromMD5Bytes converts a 16-byte MD5 digest into a Windows GUID structure
// following the little-endian System.Guid(byte[]) field layout.
func GUIDFromMD5Bytes(b [16]byte) GUID {
	data1 := uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
	data2 := uint16(b[4]) | uint16(b[5])<<8
	data3 := uint16(b[6]) | uint16(b[7])<<8
	var data4 [8]byte
	copy(data4[:], b[8:16])
	return GUID{
		Data1: data1,
		Data2: data2,
		Data3: data3,
		Data4: data4,
	}
}

// HRESULT represents a 32-bit signed COM/WinRT status code.
type HRESULT int32

// Succeeded reports whether hr is non-negative (S_OK, S_FALSE, or positive success code).
func (hr HRESULT) Succeeded() bool {
	return int32(hr) >= 0
}

// Failed reports whether hr has the high severity bit set (< 0).
func (hr HRESULT) Failed() bool {
	return int32(hr) < 0
}

// Uint32 returns the unsigned 32-bit bit-pattern of hr (e.g. 0x80004005).
func (hr HRESULT) Uint32() uint32 {
	return uint32(int32(hr))
}

// Uintptr returns the unsigned 32-bit bit-pattern of hr widened to uintptr for syscall/callback returns.
func (hr HRESULT) Uintptr() uintptr {
	return uintptr(uint32(int32(hr)))
}

// Win32Error extracts the underlying 16-bit Win32 error code when the facility is FACILITY_WIN32 (7).
func (hr HRESULT) Win32Error() syscall.Errno {
	u := uint32(hr)
	if u&0xFFFF0000 == 0x80070000 {
		return syscall.Errno(u & 0xFFFF)
	}
	return syscall.Errno(u)
}

// Err converts a failed HRESULT into an *HresultError, or returns nil if hr.Succeeded().
func (hr HRESULT) Err(op string) error {
	if hr.Succeeded() {
		return nil
	}
	return &HresultError{Op: op, Hr: hr}
}

// HresultError wraps a negative 32-bit COM/WinRT HRESULT failure code.
type HresultError struct {
	Op string
	Hr HRESULT
}

// Error returns the formatted COM operation and 32-bit hexadecimal HRESULT code.
func (e *HresultError) Error() string {
	var buf [128]byte
	b := buf[:0]
	b = append(b, e.Op...)
	b = append(b, " failed with HRESULT 0x"...)
	var hexBuf [8]byte
	putHex32Upper(hexBuf[:], uint32(e.Hr))
	b = append(b, hexBuf[:]...)
	return string(b)
}

// Unwrap returns the underlying Win32 error code if the HRESULT encodes FacilityWin32.
func (e *HresultError) Unwrap() error {
	return e.Hr.Win32Error()
}

// CheckHR converts a raw syscall return value into an error if the signed 32-bit HRESULT is negative.
func CheckHR(op string, r1 uintptr) error {
	return HRESULT(int32(r1)).Err(op)
}

// NTSTATUS represents a 32-bit Windows kernel status code.
type NTSTATUS uint32

// Succeeded reports whether st is non-negative.
func (st NTSTATUS) Succeeded() bool {
	return int32(st) >= 0
}

// Failed reports whether st has the high bit set.
func (st NTSTATUS) Failed() bool {
	return int32(st) < 0
}

// NtStatusError wraps a failed 32-bit NTSTATUS code.
type NtStatusError struct {
	Op     string
	Status NTSTATUS
}

// Error returns the formatted kernel operation and 32-bit hexadecimal NTSTATUS code.
func (e *NtStatusError) Error() string {
	var buf [96]byte
	b := buf[:0]
	b = append(b, e.Op...)
	b = append(b, " failed with NTSTATUS 0x"...)
	var hexBuf [8]byte
	putHex32Upper(hexBuf[:], uint32(e.Status))
	b = append(b, hexBuf[:]...)
	return string(b)
}

// CheckNTStatus checks whether a raw syscall return value indicates an NTSTATUS failure.
func CheckNTStatus(op string, r1 uintptr) error {
	st := NTSTATUS(uint32(r1))
	if st.Failed() {
		return &NtStatusError{Op: op, Status: st}
	}
	return nil
}

// BSTR represents an OLE Automation length-prefixed UTF-16 string pointer (`SysAllocString`).
type BSTR uintptr

// HSTRING represents a WinRT immutable string handle (`WindowsCreateString`).
type HSTRING uintptr

// SAFEARRAY represents an OLE Automation `SAFEARRAY*` descriptor pointer.
type SAFEARRAY uintptr

// VARTYPE identifies the active union member of a [VARIANT].
type VARTYPE uint16

// Standard OLE Automation VARIANT type tags (`VARENUM`).
const (
	VTEmpty    VARTYPE = 0
	VTNull     VARTYPE = 1
	VTI2       VARTYPE = 2
	VTI4       VARTYPE = 3
	VTR4       VARTYPE = 4
	VTR8       VARTYPE = 5
	VTBSTR     VARTYPE = 8
	VTDispatch VARTYPE = 9
	VTBool     VARTYPE = 11
	VTVariant  VARTYPE = 12
	VTUnknown  VARTYPE = 13
	VTI1       VARTYPE = 16
	VTUI1      VARTYPE = 17
	VTUI2      VARTYPE = 18
	VTUI4      VARTYPE = 19
	VTI8       VARTYPE = 20
	VTUI8      VARTYPE = 21
	VTArray    VARTYPE = 0x2000
	VTByRef    VARTYPE = 0x4000
)

// VARIANT models the exact 24-byte 64-bit Windows `tagVARIANT` ABI structure
// (2-byte VARTYPE + 6 bytes reserved + 16-byte payload union).
type VARIANT struct {
	VT        VARTYPE
	Reserved1 uint16
	Reserved2 uint16
	Reserved3 uint16
	Val1      uint64
	Val2      uint64
}

// NewVariantEmpty returns a zero-initialized VT_EMPTY VARIANT.
func NewVariantEmpty() VARIANT {
	return VARIANT{VT: VTEmpty}
}

// NewVariantI4 constructs a 32-bit signed integer (VT_I4) VARIANT.
func NewVariantI4(v int32) VARIANT {
	return VARIANT{VT: VTI4, Val1: uint64(uint32(v))}
}

// NewVariantI8 constructs a 64-bit signed integer (VT_I8) VARIANT.
func NewVariantI8(v int64) VARIANT {
	return VARIANT{VT: VTI8, Val1: uint64(v)}
}

// NewVariantBool constructs a COM VARIANT_BOOL (VT_BOOL: -1 = VARIANT_TRUE, 0 = VARIANT_FALSE).
func NewVariantBool(v bool) VARIANT {
	var flag uint64
	if v {
		flag = 0xFFFF // VARIANT_TRUE (-1 as int16)
	}
	return VARIANT{VT: VTBool, Val1: flag}
}

// NewVariantBSTR constructs a VT_BSTR VARIANT holding b.
func NewVariantBSTR(b BSTR) VARIANT {
	return VARIANT{VT: VTBSTR, Val1: uint64(b)}
}

// NewVariantUnknown constructs a VT_UNKNOWN VARIANT holding a COM IUnknown pointer.
func NewVariantUnknown(unk uintptr) VARIANT {
	return VARIANT{VT: VTUnknown, Val1: uint64(unk)}
}

// Int32 returns the signed 32-bit payload of a VT_I4 VARIANT.
func (v VARIANT) Int32() int32 {
	return int32(uint32(v.Val1))
}

// Int64 returns the signed 64-bit payload of a VT_I8 VARIANT.
func (v VARIANT) Int64() int64 {
	return int64(v.Val1)
}

// Bool returns the boolean value of a VT_BOOL VARIANT.
func (v VARIANT) Bool() bool {
	return uint16(v.Val1) != 0
}

// Ptr returns the raw pointer payload (VT_BSTR, VT_UNKNOWN, VT_DISPATCH).
func (v VARIANT) Ptr() uintptr {
	return uintptr(v.Val1)
}

// IUnknownVtbl represents the canonical 3-slot C++ virtual method table of `IUnknown`.
type IUnknownVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
}

// IInspectableVtbl represents the 6-slot C++ virtual method table of WinRT `IInspectable`.
type IInspectableVtbl struct {
	QueryInterface      uintptr
	AddRef              uintptr
	Release             uintptr
	GetIids             uintptr
	GetRuntimeClassName uintptr
	GetTrustLevel       uintptr
}

// Ptr is a zero-allocation value wrapper around a raw COM or WinRT interface pointer.
type Ptr struct {
	raw uintptr
}

// NewPtr wraps a raw COM interface pointer.
func NewPtr(raw uintptr) Ptr {
	return Ptr{raw: raw}
}

// Raw returns the underlying uintptr interface address.
func (p Ptr) Raw() uintptr {
	return p.raw
}

// IsNil reports whether the interface pointer is zero.
func (p Ptr) IsNil() bool {
	return p.raw == 0
}

// ptrFromUintptr converts an OS/COM unmanaged address into an unsafe.Pointer
// without triggering go vet's GC-heap uintptr-to-Pointer heuristic.
func ptrFromUintptr(p uintptr) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&p))
}

// VtblSlot reads the function pointer at zero-based index `slot` from the COM object's
// C++ virtual method table without bounds-check or slice allocation overhead.
func VtblSlot(instance uintptr, slot int) uintptr {
	if instance == 0 || slot < 0 {
		return 0
	}
	vtbl := *(*uintptr)(ptrFromUintptr(instance))
	if vtbl == 0 {
		return 0
	}
	return *(*uintptr)(ptrFromUintptr(vtbl + uintptr(slot)*unsafe.Sizeof(uintptr(0))))
}

// AppendUTF16AsUTF8 transcodes a UTF-16 code unit slice (including surrogate pairs
// and stopping at the first NUL terminator) directly into dst with zero heap
// allocations when dst has sufficient capacity.
func AppendUTF16AsUTF8(dst []byte, u16 []uint16) []byte {
	return appendUTF16AsUTF8(dst, u16)
}

// appendUTF16AsUTF8 transcodes a UTF-16 slice (including surrogate pairs) directly into dst
// with zero heap allocations when dst has sufficient capacity.
func appendUTF16AsUTF8(dst []byte, u16 []uint16) []byte {
	for i := 0; i < len(u16); i++ {
		r := rune(u16[i])
		if r == 0 {
			break
		}
		if r >= 0xD800 && r <= 0xDBFF && i+1 < len(u16) {
			low := rune(u16[i+1])
			if low >= 0xDC00 && low <= 0xDFFF {
				r = 0x10000 + ((r - 0xD800) << 10) + (low - 0xDC00)
				i++
			}
		}
		dst = utf8.AppendRune(dst, r)
	}
	return dst
}

func putHex8(dst []byte, v uint8) {
	_ = dst[1]
	dst[0] = hexDigits[v>>4]
	dst[1] = hexDigits[v&0x0F]
}

func putHex16(dst []byte, v uint16) {
	_ = dst[3]
	dst[0] = hexDigits[(v>>12)&0x0F]
	dst[1] = hexDigits[(v>>8)&0x0F]
	dst[2] = hexDigits[(v>>4)&0x0F]
	dst[3] = hexDigits[v&0x0F]
}

func putHex32(dst []byte, v uint32) {
	_ = dst[7]
	for i := 7; i >= 0; i-- {
		dst[i] = hexDigits[v&0x0F]
		v >>= 4
	}
}

func putHex32Upper(dst []byte, v uint32) {
	const upperDigits = "0123456789ABCDEF"
	_ = dst[7]
	for i := 7; i >= 0; i-- {
		dst[i] = upperDigits[v&0x0F]
		v >>= 4
	}
}

func fromHexChar(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	default:
		return 0, false
	}
}

func parseHex8(s string) (uint8, bool) {
	if len(s) != 2 {
		return 0, false
	}
	hi, ok1 := fromHexChar(s[0])
	lo, ok2 := fromHexChar(s[1])
	return (hi << 4) | lo, ok1 && ok2
}

func parseHex16(s string) (uint16, bool) {
	if len(s) != 4 {
		return 0, false
	}
	b0, ok0 := parseHex8(s[0:2])
	b1, ok1 := parseHex8(s[2:4])
	return (uint16(b0) << 8) | uint16(b1), ok0 && ok1
}

func parseHex32(s string) (uint32, bool) {
	if len(s) != 8 {
		return 0, false
	}
	w0, ok0 := parseHex16(s[0:4])
	w1, ok1 := parseHex16(s[4:8])
	return (uint32(w0) << 16) | uint32(w1), ok0 && ok1
}
