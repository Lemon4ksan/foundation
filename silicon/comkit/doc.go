// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package comkit provides a zero-allocation Component Object Model (COM) and Windows
// Runtime (WinRT) ABI substrate for direct C++ virtual method table dispatch,
// reference-counted interface pointers, and unmanaged string/task memory.
//
// Interacting with Windows system interfaces (such as DirectX/DXGI, UI Automation,
// OneCore container managers, and WinRT activation factories) via naive variadic
// syscall wrappers or late-bound IDispatch reflection introduces heap escapes on every
// method call. When a variadic slice ([]uintptr) is constructed or appended to prepend
// the implicit C++ `this` pointer, the Go compiler spills the slice to the heap,
// triggering garbage collection pressure inside high-frequency frame capture and
// accessibility tree traversal loops.
//
// This package eliminates heap escapes by providing fixed-arity vtable dispatchers
// ([VtblCall0] through [VtblCall9]), value-typed COM interface handles ([Ptr]), exact
// 24-byte x64 [VARIANT] structures, and caller-buffered UTF-8 decoders for OLE [BSTR],
// WinRT [HSTRING], and `CoTaskMemAlloc` UTF-16 buffers.
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: direct Windows COM vtable and WinRT ABI
// invocation without cgo or variadic slice heap escapes.
//
// Rejected compromise: Variadic `uintptr` slice wrappers that escape to the Go heap on
// every virtual method call, and third-party `IDispatch` reflection libraries that
// allocate per parameter.
//
// Accepted cost: Callers must specify exact C++ vtable slot indices and manage
// reference counts (`Release`), `BSTR` lifecycles (`FreeBSTR`), and `HSTRING` handles
// (`DeleteHString`) explicitly.
//
// Allocations: zero
package comkit
