// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package memkit provides zero-cost Go compiler, static-analyzer, memory-layout,
// escape-analysis, and runtime ABI intrinsics for systems programming.
//
// Go intentionally hides low-level struct layout guarantees, escape-analysis
// overrides, and interface header representations behind internal runtime packages
// (such as runtime/internal/sys and sync/atomic.noCopy). Package memkit consolidates
// these canonical zero-overhead idioms in Layer 0 so higher-level packages do not
// scatter ad-hoc compiler tricks across unrelated modules.
//
// # Zero-Sized Field Ordering Rule
//
// When embedding zero-sized marker types ([NoCopy], [NoCompare], [Align64]), always
// place them as the first field(s) of a struct:
//
//	type GuardedState struct {
//	    _ memkit.NoCopy
//	    _ memkit.NoCompare
//	    _ memkit.Align64
//	    counter uint64
//	}
//
// If a zero-sized field is placed as the final field of a non-empty struct, the Go
// compiler appends 1 byte of trailing padding (rounded up to the struct's alignment,
// adding up to 8 bytes) so taking the address of the trailing field never points past
// the end of the allocation.
//
// Stdlib counterpart: sync/atomic (unexported noCopy, align64), structs.HostLayout,
// runtime (unexported noescape, eface, iface), and golang.org/x/sys/cpu.CacheLinePad.
//
// Rejected compromise: Keeping compiler and vet markers private inside standard
// library internals or scattering duplicate definitions (such as NoCopy in bytesconv
// and emptyInterface in encoding/json) across unrelated packages.
//
// Accepted cost: Functions such as [NoEscape] and [UintptrToPointer] bypass compiler
// escape tracking and go vet pointer provenance checks; callers must uphold lifetime
// invariants manually so stack pointers never outlive their frame.
//
// Allocations: zero.
package memkit
