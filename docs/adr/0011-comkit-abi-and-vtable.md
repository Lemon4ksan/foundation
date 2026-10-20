# ADR-0011: Zero-allocation COM and WinRT ABI substrate in `silicon/comkit`

Status: Accepted
Date: 2026-10-20
Deciders: Lemon4ksan

## Context

Downstream systems in our ecosystem—specifically the `stasis` Windows Sandbox hypervisor engine and `ctrl`'s upcoming `uiakit` Computer Use / UI Automation framework—interact heavily with native Windows Component Object Model (COM) and Windows Runtime (WinRT) binary interfaces (`IUnknown`, `IInspectable`, `IUIAutomation8`, `ID3D11Device`, `IDXGIOutputDuplication`, and OneCore container factories).

Existing approaches in the Go ecosystem suffer from three severe architectural flaws:
1. **Heap Escapes on Vtable Dispatch**: Naïve variadic wrappers (`func CallVtbl(obj uintptr, slot int, args ...uintptr)`) allocate a slice on the heap whenever more than 4–5 arguments are passed or when arguments escape via `append([]uintptr{obj}, args...)`. In 60 FPS DXGI frame loops or deep UIA subtree walks, this violates the `zero` allocation budget ([ADR-0003](0003-allocation-budgets.md)).
2. **Legacy `IDispatch` Overhead in Third-Party Libraries**: Community libraries such as `go-ole` rely on late-bound `IDispatch::Invoke` reflection, string-based member lookups, and unpooled `VARIANT` boxing, adding 10–50× latency over direct C++ vtable dispatch.
3. **Duplicated Unsafe Boilerplate**: Without a shared substrate in `foundation`, `stasis` and `ctrl` would duplicate `IUnknown` reference counting, `BSTR` (`SysAllocString` / `SysFreeString`), WinRT `HSTRING` (`WindowsCreateString` / `WindowsDeleteString`), `CoTaskMemFree` UTF-16 drainers, `VARIANT` 24-byte x64 ABI layouts, and OS thread apartment initialization (`CoInitializeEx` / `RoInitialize`).

## Decision

We introduce **`silicon/comkit`** in **Layer 0 (Silicon Substrate)** ([ADR-0002](0002-layering-and-isolation.md)) as the canonical, zero-allocation COM and WinRT ABI substrate:

1. **Placement in Layer 0 (`silicon/comkit`)**:
   - Conforming to the Silicon Autonomy Invariant ([ADR-0002](0002-layering-and-isolation.md)), `silicon/comkit` imports zero other `foundation` packages and depends solely on Go's standard library (`syscall`, `unsafe`, `runtime`) and `golang.org/x/sys/windows` (with portable non-Windows stubs so cross-platform packages compile cleanly).
   - Allocation tier: `zero` ([ADR-0003](0003-allocation-budgets.md)) on steady-state vtable invocation, reference counting, `VARIANT` construction, and caller-buffered UTF-16 string decoding.

2. **Arity-Specialized Zero-Escape Vtable Dispatch**:
   - Provide inlineable, arity-specialized dispatchers (`VtblCall0` through `VtblCall9`) that pass arguments directly in registers/stack to `syscall.SyscallN` without constructing intermediate `[]uintptr` slices on the heap.
   - Provide `Ptr[T]` / `Interface` value types wrapping raw COM interface pointers with `QueryInterface`, `AddRef`, `Release`, and `Vtbl(slot)`.

3. **Deterministic Memory & String Primitives**:
   - **OLE Automation `BSTR`**: `AllocBSTR`, `FreeBSTR`, `BSTRToString`, and zero-allocation `AppendBSTRUTF8(dst []byte, bstr uintptr)`.
   - **WinRT `HSTRING`**: `NewHString`, `DeleteHString`, and `RoGetActivationFactory` / `RoActivateInstance`.
   - **COM Task Allocator (`CoTaskMemFree`)**: `DrainCoTaskUTF16` and `AppendCoTaskUTF8` for safely consuming `LPWSTR` buffers returned by OneCore and Shell APIs.
   - **x64 `VARIANT` & `SAFEARRAY` ABI**: Exact 24-byte `VARIANT` layout (`vt uint16`, reserved words, 16-byte payload union) with `VariantClear`, plus 1D `SAFEARRAY` creation, bounds inspection, zero-allocation `AppendSafeArrayInt32` (for UIA `RuntimeId` and WMI arrays), and `SafeArrayDestroy`.
   - **Out-of-Process RPC Security (`CoSetProxyBlanket`)**: Direct `ole32!CoSetProxyBlanket` configuration for `CLSCTX_LOCAL_SERVER` proxies (WMI, VSS, Task Scheduler, `SandboxUDK`).
   - **Apartment Thread Guard (`ScopeMTA` / `ScopeSTA`)**: Deterministic `runtime.LockOSThread()` + `CoInitializeEx` / `RoInitialize` lifecycle guards that tolerate `RPC_E_CHANGED_MODE` when a thread's apartment is already established.

4. **Strict Separation from Higher-Level Domain APIs**:
   - Domain-specific COM interfaces do **not** belong in `foundation/silicon/comkit`:
     - Cross-process UI Automation (`CUIAutomation8`, `FindAllBuildCache`), Set-of-Marks vision, and desktop automation belong in **`ctrl/uiakit`**.
     - Hypervisor-internal `SandboxUDK.dll` (`IManagedWindowsVM`) and OneCore CLEM bindings remain in **`stasis`**.

## Consequences

- What gets easier:
  - Both `ctrl/uiakit` and `stasis` share a single, battle-tested, zero-allocation COM/WinRT ABI foundation without duplicating `unsafe.Pointer` vtable arithmetic or `BSTR`/`HSTRING` memory management.
  - Fixed-arity `VtblCall0..9` eliminates heap allocations during high-frequency COM calls (DXGI GPU texture mapping, UIA cache traversal).
  - Cross-platform builds (`GOOS=linux`, `GOOS=darwin`) compile against `silicon/comkit` without `#ifdef` clutter in consumer type definitions.
- What gets harder / what we pay:
  - Callers invoking `VtblCallN` are responsible for matching exact C++ vtable slot indices and `uintptr` argument widths, as Go cannot verify C++ virtual method signatures at compile time.
  - Callers must still pair `AddRef` / `CoCreateInstance` with `defer ptr.Release()` and `AllocBSTR` with `defer FreeBSTR(b)`.

## Alternatives considered

- **Placing all COM primitives inside `ctrl`**: Rejected. `ctrl` is the CLI/TUI/MCP application surface; low-level ABI pointer arithmetic, `CoTaskMem`, and `SyscallN` vtable dispatch are Layer 0 OS substrate concerns that belong in `foundation/silicon`.
- **Using `github.com/go-ole/go-ole`**: Rejected. `go-ole` targets `IDispatch` scripting, allocates on every call, lacks WinRT `HSTRING` / `RoGetActivationFactory` support, and violates our zero-dependency and `zero` allocation tier mandates ([ADR-0003](0003-allocation-budgets.md)).
