// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package memkit

import (
	"structs"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/cpu"
)

// CacheLinePad is an architecture-aware cache-line padding barrier embedded
// between concurrently mutated struct fields (such as producer/consumer cursors
// or per-core shards) to eliminate MESI cache-line ping-pong (false sharing)
// across CPU cores.
//
// Its size is resolved at compile time per target GOARCH via
// [cpu.CacheLinePad] (64 bytes on amd64/arm64, 128 on ppc64, 256 on s390x).
type CacheLinePad = cpu.CacheLinePad

// CacheLineSize is the target architecture's CPU cache line size in bytes,
// evaluated at compile time from [CacheLinePad].
const CacheLineSize = int(unsafe.Sizeof(CacheLinePad{}))

// NoCopy may be embedded as the first field of a struct that must not be copied
// by value after first use.
//
// It implements the sync.Locker interface on its pointer receiver (*NoCopy) as
// a compile-time no-op so that "go vet -copylocks" statically flags any value
// copy (assignment, pass-by-value, or return-by-value) without adding any
// memory or runtime overhead.
//
// Always place NoCopy as the first field of a struct; placing a zero-sized
// field at the end of a non-empty struct causes the Go compiler to append
// trailing alignment padding.
//
// See https://golang.org/issues/8005#issuecomment-190753527 for details.
type NoCopy struct{}

// Lock is a no-op used by the -copylocks static analyzer in go vet.
func (*NoCopy) Lock() {}

// Unlock is a no-op used by the -copylocks static analyzer in go vet.
func (*NoCopy) Unlock() {}

// NoCompare may be embedded as the first field of a struct to disallow
// equality comparisons (== and !=) and prevent the struct from being used
// as a map key at compile time.
//
// Because a zero-length array of functions ([0]func()) has size 0 and
// pointer alignment, it adds zero bytes before pointer-aligned fields while
// exploiting the Go language specification rule that function types are
// incomparable.
type NoCompare [0]func()

// Align64 may be embedded as the first field of a struct (or immediately
// before a 64-bit field) to guarantee 8-byte memory alignment across both
// 64-bit and 32-bit architectures with zero size overhead.
//
// On 32-bit architectures (386, arm, mips), unaligned 64-bit atomic operations
// panic at runtime. Because [0]atomic.Int64 has size 0 and alignment 8,
// embedding Align64 forces 64-bit alignment without adding padding bytes.
type Align64 [0]atomic.Int64

// HostLayout marks a struct as conforming to the host platform's C ABI memory
// layout expectations. It is a type alias for the standard library's
// [structs.HostLayout] marker.
type HostLayout = structs.HostLayout

// Eface represents the internal two-word runtime memory layout of an empty
// interface (any / interface{}) in the Go ABI (runtime.eface).
type Eface struct {
	// Type points to the runtime._type descriptor of the boxed concrete value.
	Type unsafe.Pointer
	// Data points to the underlying value (or holds a pointer-shaped value directly).
	Data unsafe.Pointer
}

// Iface represents the internal two-word runtime memory layout of a non-empty
// interface with methods in the Go ABI (runtime.iface).
type Iface struct {
	// Itab points to the runtime.itab dispatch table pairing the interface and concrete type.
	Itab unsafe.Pointer
	// Data points to the underlying concrete value.
	Data unsafe.Pointer
}

// EfaceOf reinterprets a pointer to an empty interface (*any) as a pointer to
// its two-word [Eface] runtime header without allocation.
//
//go:nosplit
func EfaceOf(v *any) *Eface {
	return (*Eface)(unsafe.Pointer(v))
}

// IfaceOf reinterprets a pointer to an interface variable (*T) as a pointer to
// its two-word [Iface] runtime header without allocation.
//
//go:nosplit
func IfaceOf[T any](v *T) *Iface {
	return (*Iface)(unsafe.Pointer(v))
}

// IsNilInterface reports whether v is either an untyped nil interface or an
// interface boxing a typed nil pointer, channel, map, or function.
//
// In Go, an interface holding a typed nil pointer (for example, (*MyError)(nil)
// returned as an error) satisfies v != nil because its runtime type descriptor
// word (Eface.Type) is non-nil even though its data word (Eface.Data) is nil.
// IsNilInterface checks both words in O(1) time with zero allocations and
// without invoking reflect.ValueOf.
//
//go:nosplit
func IsNilInterface(v any) bool {
	if v == nil {
		return true
	}
	return (*Eface)(unsafe.Pointer(&v)).Data == nil
}

// NoEscape hides a pointer from the Go compiler's escape analysis, preventing
// the pointed-to value from escaping to the heap when passed to functions or
// syscalls guaranteed not to retain the pointer beyond the call's lifetime.
//
// It compiles down to a no-op XOR-with-zero on the pointer's uintptr
// representation, severing the compiler's data-flow provenance graph at zero
// runtime cost (modeled after runtime.noescape).
//
// SAFETY: The caller MUST guarantee that the returned pointer does not outlive
// the current stack frame and is never stored into a heap object while the
// goroutine stack may grow or relocate.
//
//go:nosplit
func NoEscape[T any](p *T) *T {
	x := uintptr(unsafe.Pointer(p))
	return (*T)(*(*unsafe.Pointer)(unsafe.Pointer(&x)))
}

// NoEscapePointer hides an unsafe.Pointer from the Go compiler's escape analysis.
//
// SAFETY: See [NoEscape] for lifetime requirements.
//
//go:nosplit
func NoEscapePointer(p unsafe.Pointer) unsafe.Pointer {
	x := uintptr(p)
	return *(*unsafe.Pointer)(unsafe.Pointer(&x))
}

// NoEscapeBytes hides the backing array pointer of a byte slice from the Go
// compiler's escape analysis, allowing stack-allocated scratch buffers passed
// to Read/Write/parser functions to remain on the stack.
//
// SAFETY: See [NoEscape] for lifetime requirements.
//
//go:nosplit
func NoEscapeBytes(b []byte) []byte {
	if len(b) == 0 {
		return b
	}
	ptr := (*byte)(NoEscapePointer(unsafe.Pointer(unsafe.SliceData(b))))
	return unsafe.Slice(ptr, cap(b))[:len(b)]
}

// NoEscapeString hides the backing byte pointer of a string from the Go
// compiler's escape analysis.
//
// SAFETY: See [NoEscape] for lifetime requirements.
//
//go:nosplit
func NoEscapeString(s string) string {
	if len(s) == 0 {
		return ""
	}
	ptr := (*byte)(NoEscapePointer(unsafe.Pointer(unsafe.StringData(s))))
	return unsafe.String(ptr, len(s))
}

// UintptrToPointer converts a non-GC kernel or hardware memory address (such as
// VirtualAlloc, mmap, io_uring shared rings, or COM vtable addresses) into an
// unsafe.Pointer via pointer punning without triggering go vet's unsafeptr
// false-positive warning.
//
// SAFETY: addr MUST refer to OS-managed or static memory outside the Go
// garbage collector's heap, as the GC does not track uintptr values across
// stack growth or compaction points.
//
//go:nosplit
func UintptrToPointer(addr uintptr) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&addr))
}
