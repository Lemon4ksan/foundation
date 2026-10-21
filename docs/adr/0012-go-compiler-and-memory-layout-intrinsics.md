# ADR-0012: Go compiler, memory-layout, and runtime ABI intrinsics in `silicon/memkit`

Status: Accepted  
Date: 2026-10-21  
Deciders: Lemon4ksan  

## Context

Go intentionally hides low-level struct layout guarantees, static-analyzer hooks, escape-analysis overrides, and runtime interface header representations inside unexported runtime or standard library packages (`sync/atomic.noCopy`, `sync/atomic.align64`, `runtime.noescape`, `runtime.eface`, `runtime.iface`).

Previously, packages across `foundation` and downstream projects either duplicated these idioms in unrelated domains or imported external packages purely for struct padding:
1. **Misplaced Static-Analysis Guards (`NoCopy`)**: `type NoCopy struct{}` (with `Lock()` and `Unlock()` pointer-receiver methods for `go vet -copylocks`) lived inside `silicon/bytesconv/copy.go`, an unrelated string/byte conversion package.
2. **Duplicated Runtime Interface Headers (`emptyInterface`)**: `encoding/json` defined a private `type emptyInterface struct { typ, word unsafe.Pointer }` to unpack `any` values without `reflect.ValueOf` allocations, while other packages lacked a zero-allocation `O(1)` helper to detect typed-`nil` pointers boxed inside interfaces (`v != nil` despite `Data == nil`).
3. **Scattered False-Sharing Padding (`CacheLinePad`)**: Higher-layer packages (`bufkit`, `net/urlkit`, `sync/spinlock`, `aoni/fast`) mixed `golang.org/x/sys/cpu.CacheLinePad`, hardcoded `const CacheLineSize = 64`, and manual `_ [60]byte` / `_ [32]byte` arrays.
4. **Missing Zero-Cost Layout & Escape Primitives**: Structs needing compile-time incomparability (`[0]func()`), guaranteed 64-bit atomic alignment on 32-bit architectures (`[0]atomic.Int64`), C-ABI host layout (`structs.HostLayout`), stack escape barriers (`uintptr(p) ^ 0`), or off-heap kernel pointer punning (`*(*unsafe.Pointer)(unsafe.Pointer(&addr))`) had no canonical home in Layer 0.

## Decision

We introduce **`silicon/memkit`** in **Layer 0 (Silicon Substrate)** ([ADR-0002](0002-layering-and-isolation.md)) as the single canonical home for zero-cost Go compiler, `go vet`, memory-layout, escape-analysis, and runtime ABI intrinsics:

### 1. Zero-Sized Compile-Time & Static-Analysis Struct Markers

| Marker | Underlying Type | Size / Align | Compiler / Analyzer Mechanism |
| :--- | :--- | :--- | :--- |
| `memkit.NoCopy` | `struct{}` with `(*NoCopy).Lock()` & `Unlock()` | `0 B` / `1 B` | Triggers `go vet -copylocks` whenever the enclosing struct is copied by value after first use. |
| `memkit.NoCompare` | `[0]func()` | `0 B` / `ptr` | Exploits the Go spec rule that `func` types are incomparable, rejecting `==`, `!=`, and `map` key usage at compile time. |
| `memkit.Align64` | `[0]atomic.Int64` | `0 B` / `8 B` | Forces 8-byte struct/field alignment across both 64-bit and 32-bit architectures (`386`, `arm`, `mips`), preventing unaligned 64-bit atomic panics. |
| `memkit.HostLayout` | `structs.HostLayout` | `0 B` / `1 B` | Instructs the Go compiler to lay out struct fields according to host platform C ABI rules. |

#### The Leading Zero-Sized Field Invariant
All zero-sized markers (`NoCopy`, `NoCompare`, `Align64`, `HostLayout`) **must** be placed as the **first** field(s) of a struct (or immediately before a non-zero-sized field). If a zero-sized field is placed as the **final** field of a non-empty struct, the Go compiler appends 1 byte of trailing padding (rounded up to the struct's alignment, adding up to 8 bytes) so taking the address of the trailing field cannot alias the next heap/stack allocation.

### 2. Hardware False-Sharing Isolation (`CacheLinePad` & `CacheLineSize`)
- `memkit.CacheLinePad` aliases `golang.org/x/sys/cpu.CacheLinePad` (`64 B` on `amd64`/`arm64`, `128 B` on `ppc64`, `256 B` on `s390x`) and `const CacheLineSize = int(unsafe.Sizeof(CacheLinePad{}))` derives the compile-time constant size from it, giving all higher-layer packages an architecture-aware L1D cache-line barrier from a single `silicon/memkit` import.

### 3. Runtime Interface ABI Introspection (`Eface`, `Iface`, `IsNilInterface`)
- `memkit.Eface` (`Type, Data unsafe.Pointer`) and `memkit.Iface` (`Itab, Data unsafe.Pointer`) expose the two-word Go runtime representation of empty (`any`) and method interfaces, alongside `EfaceOf` and `IfaceOf`.
- `memkit.IsNilInterface(v any) bool` solves Go's typed-`nil` interface gotcha in `O(1)` time with zero allocations by checking `v == nil || (*Eface)(unsafe.Pointer(&v)).Data == nil`.

### 4. Escape-Analysis Barriers & Off-Heap Pointer Punning
- `memkit.NoEscape[T]`, `NoEscapePointer`, `NoEscapeBytes`, and `NoEscapeString` sever the compiler's escape-analysis provenance graph via `uintptr(p) ^ 0` (`//go:nosplit`), keeping stack-allocated structs and scratch buffers on the goroutine stack when passed to non-retaining callees or OS syscalls.
- `memkit.UintptrToPointer(addr uintptr) unsafe.Pointer` performs pointer punning (`*(*unsafe.Pointer)(unsafe.Pointer(&addr))`) for OS kernel-allocated regions (`mmap`, `VirtualAlloc`, `io_uring` rings, COM vtables) outside the Go garbage collector's heap without triggering `go vet -unsafeptr` false positives.

## Consequences

### What gets easier:
- Every package in Layer 1 through Layer 5 (`bufkit`, `sync/spinlock`, `encoding/json`, `borrow`, `net/urlkit`) and downstream repositories (`aoni`, `mach`, `stasis`, `ctrl`, `seal`, `sein`, `dawn`) imports a single, self-documented Layer 0 package (`silicon/memkit`) for compiler and memory-layout primitives.
- `silicon/bytesconv` returns to its pure string/byte conversion focus (`NoCopy` is moved out of `bytesconv` into `memkit`).
- `encoding/json` uses `memkit.EfaceOf` instead of maintaining a private `emptyInterface` mirror.
- `spinlock.Lock` and `bufkit.Ring` embed `_ memkit.NoCopy` at offset 0 with zero size overhead, enabling `go vet -copylocks` protection.

### What gets harder / what we pay:
- Callers of `NoEscape*` and `UintptrToPointer` assume responsibility for pointer lifetime invariants: a stack pointer hidden via `NoEscape` must never be retained past the stack frame in which it was allocated, and `UintptrToPointer` must only be used for non-GC memory.

## Alternatives Considered

- **Keeping `NoCopy` in `silicon/bytesconv`**: Rejected. `bytesconv` is a string/byte slice conversion package; placing struct-copy static analysis markers there obscured discoverability.
- **Putting `NoCopy` and `CacheLinePad` in `silicon/cpukit`**: Rejected. `cpukit` queries hardware ISA extensions (`AVX2`, `BMI2`, `GFNI`, `NEON`) at runtime, whereas `NoCopy`, `NoCompare`, `Eface`, and `NoEscape` are Go compiler and runtime ABI constructs.
