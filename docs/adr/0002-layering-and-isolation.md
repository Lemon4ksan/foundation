# ADR-0002: Layering Hierarchy and Dependency Isolation

- **Status**: Accepted
- **Date**: 2026-10-03
- **Author / Deciders**: Lemon4ksan
- **Target Packages**: Root architectural model and all packages

## 1. Context & Problem Statement

As `foundation` expanded to over 100 packages across networking, cryptography, concurrency, and hardware vectorization, there was a risk of uncontrolled dependency entanglement. Without explicit architectural boundaries:
- Lower-level primitives could inadvertently pull in heavy high-level packages (e.g., pulling a JSON parser or regex engine into a fundamental cryptographic structure).
- Packages written for convenience (such as `generic`) introduce reflection or heap escaping, contaminating zero-allocation packages.
- Cross-dependencies between hardware subpackages (`silicon/*`) cause circular coupling and prevent targeted compilation.

## 2. Decision Drivers

- Enforce strict acyclic, unidirectional dependencies.
- Prevent "convenience creep": convenience abstractions must never leak into zero-overhead substrates.
- Ensure automated, deterministic enforcement in CI so that layer violations immediately fail builds.

## 3. Considered Options

1. **Option 1: Complete Package Isolation**: Every single package is completely standalone, duplicating common logic in private `internal/` trees.
2. **Option 2: Unconstrained Intra-Module Graph**: Any package in `github.com/lemon4ksan/foundation` can import any other package as long as there are no direct cycles.
3. **Option 3: Six-Layer Stratification with Substrate Autonomy and Generic Isolation**: Group all packages into layers L0 through L5 with four programmatic laws.

## 4. Decision Outcome

**Chosen Option**: **Option 3**.

### 4.1. Layer Definitions

- **Layer 0 (Hardware Substrate)**: `silicon/*` (vector primitives, CPU feature detection, high-resolution clocks, off-heap memory, cache-padded pools).
- **Layer 1 (Memory & Structures)**: `borrow`, `structures/*`, `refkit`, `bufkit`.
- **Layer 2 (Concurrency & Synchronization)**: `sync/*`, `async/*`.
- **Layer 3 (Codecs, Cryptography & Serialization)**: `codec/*`, `crypto/*`, `encoding/*`, `text/*`, `iokit`.
- **Layer 4 (Wire Protocols & Transport)**: `net/*`, `net/quic`, `net/proxy`, `net/ip`, `net/tls`, `net/urlkit`.
- **Layer 5 (Tooling & High-Level Utilities)**: `types/*`, `timekit`, `argkit`, `fskit`, `pathkit`, `tuikit`, `testing/*`, `cmd/*`.
- **External Only**: `generic` (provided solely for external consumers; forbidden internally).

### 4.2. Dependency Laws

1. **Monotonicity**: Package in layer $L_A$ may only import packages from layers $L_B \le L_A$.
2. **Generic Isolation**: Internal packages **must not** import `generic`.
3. **Silicon Autonomy**: Packages in `silicon/*` **must not** import each other or any other foundation package.
4. **Allocation Monotonicity**: Packages with stricter allocation tiers (`zero`) cannot import packages with looser tiers (`amortized`, `bounded`).

## 5. Verification & Testing

Enforced in `root_test.go` via `TestArchitecture_LayersAndIsolation`, which executes `go list -json ./...` and verifies every import edge against these laws. Any violation causes immediate test failure. Temporary transition exceptions are tracked in whitelists that can only decrease (ratchet).
