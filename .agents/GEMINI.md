---
name: foundation-guidelines
description: Strict architecture, performance, and engineering guidelines for AI agents working on foundation
trigger: always_on
# Foundation AI Engineering Guidelines

When reading, writing, modifying, or refactoring code in `foundation`, AI agents **MUST** strictly adhere to the following principles:

## 0. Vision & Core Philosophy (Stdlib v2.0)
- `foundation` is **Go Standard Library v2.0**. It re-evaluates standard library compromises (Go 1 compatibility promise, lowest common hardware denominator, single-size API ergonomics, heap allocation tolerance) to provide zero-overhead, hardware-accelerated systems primitives.
- Before making significant architectural changes or adding packages, consult `docs/VISION.md` and `docs/adr/`.
- `foundation` serves five primary consumers: `aoni` (game engine), `g-man` (gateway/proxy), `sein` (storage/WAL), `dawn` (framing/wire), and `seal` (crypto enclave). All packages must be domain-agnostic and required by at least one consumer.

## 1. Explicit Non-Goals & Boundaries
- **No L7 Protocols**: HTTP/1.1, HTTP/2, HTTP/3, HPACK, QPACK, WebSocket, Steam Network Protocol, Game RPC belong in `mach` or the respective consumer repositories. Do NOT add L7 protocol logic to `foundation`.
- **No Business Logic**: No game simulation state, no application models, no ORMs/databases.
- **No Convenience at the Cost of Allocations**: Never introduce helper functions or wrappers that hide heap allocations (`any`, reflection, unneeded interface boxes) on hot paths.

## 2. Layering Architecture & Import Laws
Imports are strictly monotonic and checked automatically via `root_test.go`:
- **Layer 0 (Hardware Substrate)**: `silicon/*`
- **Layer 1 (Memory & Structures)**: `borrow`, `structures/*`, `refkit`, `bufkit`
- **Layer 2 (Concurrency)**: `sync/*`, `async/*`
- **Layer 3 (Codecs & Cryptography)**: `codec/*`, `crypto/*`, `encoding/*`, `text/*`, `iokit`
- **Layer 4 (Wire & Transport)**: `net/*`
- **Layer 5 (Tooling & Utilities)**: `types/*`, `timekit`, `argkit`, `fskit`, `pathkit`, `tuikit`, `testing/*`, `cmd/*`
- **External Only**: `generic` is reserved exclusively for external consumer applications. Internal `foundation` packages **MUST NOT** import `generic`.
- **Silicon Autonomy**: Packages inside `silicon/*` are standalone primitives and **MUST NOT** import each other or any other `foundation` package.
- **Layer Law**: A package in layer $L_A$ may only import packages from layers $L_B \le L_A$.

## 3. Allocation Budgets & Contracts
Every package belongs to one of four allocation tiers, declared in `doc.go` and verified in tests:
1. `zero`: 0 B/op, 0 allocs/op on every call. Memory is supplied by the caller via destination slices (`Append*`).
2. `amortized`: Allocations permitted only during initial construction or growth. Zero allocs/op in steady-state.
3. `bounded(N)`: Strictly bounded to at most $N$ allocations per completed operation.
4. `unconstrained`: Cold paths, CLI tooling, test fixtures.

## 4. Testing & Quality Standards
- **Coverage Ratchet**: Statement coverage must never decrease relative to `scripts/coverage_baseline.txt`.
- **Concurrency Safety**: 100% thread-safe. Running `go test -race` must produce zero race warnings under all circumstances.
- **Fuzzing**: Mandatory for all decoders, parsers, wire format unpackers, and cryptographic primitives.
- **No Hacks or Dummy Code**: It is strictly forbidden to add fake tests, empty stubs, or trivial assertions to artificially inflate coverage.
- **Definition of Done**: `make check` (lint, test, race, arch, coverage-ratchet) must pass completely cleanly.

## 5. Canonical Package Documentation (`doc.go`)
- `doc.go` is the single source of truth for each package API.
- Each package must declare its standard library relationship in `doc.go`:
  ```go
  // Stdlib counterpart: <stdlib pkg or "none — fills gap: ...">
  // Rejected compromise: <compromise rejected>
  // Accepted cost: <tradeoff paid>
  // Allocations: zero | amortized | bounded(N/op) | unconstrained
  ```
- Package-level Markdown guides inside `docs/` are legacy and must be migrated into `doc.go` and runnable `example_*_test.go` files.

## 6. Agent Protocol
- Respect task scopes: do not modify packages outside the task's stated target packages.
- When introducing a new package or changing layer boundaries, write an ADR in `docs/adr/`.
- If requirements are ambiguous or require crossing established layer boundaries, **STOP and ask the user**.
