# Foundation Vision & Charter

## 1. Executive Summary & Thesis (ADR-0001)

**`foundation` is Go standard library v2.0.**

The official Go standard library (`stdlib`) is engineered for broad web and service workloads, prioritizing utmost simplicity, universal portability across obscure CPU architectures, and unconditional backwards compatibility (the Go 1 promise). While ideal for business application development, these priorities mandate structural compromises unacceptable for high-density networking, real-time engines, and compute-intensive runtimes:

| Standard Library Compromise | `foundation` Architectural Stance | Accepted Tradeoff / Cost |
| :--- | :--- | :--- |
| **Go 1 Compatibility Promise** (APIs frozen in perpetuity) | Rapid iterative evolution within `v0.x`; breaking changes permitted when recorded in CHANGELOG | Consumers pin release tags and upgrade intentionally |
| **Universal Portability** (lowest common hardware denominator) | Mechanical sympathy: direct SIMD vectorization (AVX2/BMI2, NEON), `c2plan9`, and `simd/archsimd` intrinsics (Go 1.27+) | Tier-1 platforms prioritized (Linux/Windows/macOS on amd64/arm64) |
| **Single Universal API** ("one size fits all") | Zero-allocation push iterators (`iter.Seq`, `iter.Seq2`), specialized buffer-passing signatures (`Append*`), unrolled scanning | Slightly larger package surface area tailored to specific performance profiles |
| **Heap Tolerance** (allocations accepted for ergonomics and safety) | Zero-allocation contracts (`zero-alloc`), off-heap memory, arenas, explicit generational affine ownership (`borrow.Box`) | Callers pass destination buffers; responsibility for lifetime management |
| **Extreme Conservatism** (years-long deliberation through `x/exp`) | Immediate adoption of cutting-edge Go runtime capabilities (Go 1.27+, SIMD intrinsics, value generics) | Compiler version prerequisite (Go 1.27+) |

## 2. Primary Consumers

`foundation` serves as the unified systems substrate for five interconnected consumer projects, engineered to open-source enterprise library standards:

```
                  +----------------------------------------------+
                  |                  foundation                  |
                  |           (Go Standard Library 2.0)          |
                  +----------------------------------------------+
                         ^        ^         ^        ^        ^
                         |        |         |        |        |
         +---------------+        |         |        |        +---------------+
         |                        |         |        |                        |
     [ aoni ]                 [ g-man ]  [ sein ]  [ dawn ]                [ seal ]
 (Game Engine Core /      (Network Gateway / (Storage / (Protocol & Wire / (Security & Cryptographic
  Deterministic Loop)      High-Concurrency)   WAL / I/O) Framing Engine)   Enclave Runtime)
```

### Subsystem Mapping

| Consumer | Primary Domain | Core `foundation` Dependencies |
| :--- | :--- | :--- |
| **`aoni`** | Deterministic game engine, frame loops, math and physics substrate | `silicon/*` (SIMD, cpukit, clock), `structures/*` (bitset, minheap, deque), `borrow`, `timekit` |
| **`g-man`** | High-throughput distributed proxy, socket broker, connection router | `net/*` (quic, proxy, ip), `async/*` (pool, scheduler, pipeline, fsm), `sync/*` (limiter, breaker) |
| **`sein`** | Storage engine, write-ahead log (WAL), block I/O and streaming | `iokit`, `bufkit`, `fskit`, `codec/compress/*` (zstd, lz4, fse), `silicon/ringbuf` |
| **`dawn`** | Protocol processing, serialization, binary framing and decoding | `codec/json`, `encoding/*` (varint, bin, base64), `silicon/hexkit`, `silicon/trie` |
| **`seal`** | Cryptographic enclave, secure key management, identity runtime | `crypto/*` (aead, kdf, sign), `types/secret`, `types/uuid` |

## 3. Project Boundaries & Inclusion Criteria

### What Belongs in `foundation`:
1. **Domain-Neutral Systems Primitives**: Memory, concurrency, data structures, binary codecs, cryptography, low-level transport (L4 & QUIC).
2. **Real Consumer Demand**: Every package must be actively required by at least one consumer project. No speculative or purely academic code.
3. **Strict Domain Agnosticism**: No package in `foundation` may contain knowledge of game engine entities, business databases, or application logic.

### Explicit Non-Goals (What foundation Rejects):
- ❌ **Layer 7 Application Protocols**: HTTP/1.1, HTTP/2, HTTP/3, HPACK, QPACK, WebSocket, Steam Network Protocol, Game RPC. These belong in specialized transport repositories (e.g., `mach`) or within individual consumers.
- ❌ **UI Frameworks & Visual Rendering**: `tuikit` is constrained strictly to diagnostic and developer terminal utilities.
- ❌ **ORMs, Databases, and Migration Systems**.
- ❌ **Convenience at the Expense of Heap Pressure**: Never introduce helper functions that hide heap slices or cast to `any` on performance-critical paths.

## 4. Allocation Budgets & Contracts

Every package must state its allocation contract in `doc.go` and guarantee it through `testing.AllocsPerRun` tests:

| Tier | Allocation Contract | Target Packages | Test Verification |
| :--- | :--- | :--- | :--- |
| **`zero`** | **0 B/op, 0 allocs/op** on every single invocation. Callers supply pre-allocated memory via destination slices (`dst []byte`). | `silicon/*`, `encoding/*`, parsers, `borrow` | `AllocsPerRun == 0` |
| **`amortized`** | Allocations permitted **only** during initialization or capacity scaling. Strictly **0 allocs/op** in steady state. | `structures/*`, `silicon/pool`, `bufkit`, `codec/*` | `AllocsPerRun == 0` after warmup |
| **`bounded(N)`** | Strictly bounded to at most $N$ allocations per completed transaction (exact $N$ documented in `doc.go`). | `async/task`, `net/quic` (per session setup) | `AllocsPerRun <= N` |
| **`unconstrained`** | Cold execution paths: CLI flag parsing, system initialization, unit testing helpers, offline compilers (`c2plan9`). | `argkit`, `fskit`, `cmd/c2plan9`, `testing/*` | Goroutine leak checks |

## 5. Layering Architecture & Dependency Laws

The module dependency graph is strictly acyclic, monotonic with respect to allocation budgets, and stratified across six horizontal layers:

```
+-----------------------------------------------------------------------------------+
| Layer 5: Tooling & Utilities                                                      |
| (types/*, timekit, argkit, fskit, pathkit, tuikit, testing/*, cmd/*)              |
+-----------------------------------------------------------------------------------+
                                          |
                                          v
+-----------------------------------------------------------------------------------+
| Layer 4: Wire Protocols & Transport                                               |
| (net/*, net/quic, net/proxy, net/ip, net/tls, net/urlkit)                        |
+-----------------------------------------------------------------------------------+
                                          |
                                          v
+-----------------------------------------------------------------------------------+
| Layer 3: Codecs, Serialization & Cryptography                                     |
| (codec/*, crypto/*, encoding/*, text/*, iokit)                                    |
+-----------------------------------------------------------------------------------+
                                          |
                                          v
+-----------------------------------------------------------------------------------+
| Layer 2: Concurrency & Synchronization                                            |
| (sync/*, async/*)                                                                 |
+-----------------------------------------------------------------------------------+
                                          |
                                          v
+-----------------------------------------------------------------------------------+
| Layer 1: Memory & Substrate Structures                                            |
| (borrow, structures/*, refkit, bufkit)                                            |
+-----------------------------------------------------------------------------------+
                                          |
                                          v
+-----------------------------------------------------------------------------------+
| Layer 0: Hardware Substrate                                                       |
| (silicon/*: cpukit, clock, simd, bytesconv, hexkit, offheap, pool, etc.)          |
+-----------------------------------------------------------------------------------+
```

### Dependency Laws (Enforced programmatically in `root_test.go`):
1. **Layer Monotonicity**: A package in Layer $L_A$ may only import packages from Layers $L_B \le L_A$.
2. **Allocation Monotonicity**: A package with a `zero` allocation contract **cannot** import a package with `amortized` or `bounded` contracts.
3. **Generic Isolation**: The `generic` package is reserved strictly for external consumers. Internal `foundation` packages **must not** import `generic`.
4. **Silicon Substrate Autonomy**: Packages within `silicon/*` are standalone primitives and **must not** import each other or any other `foundation` package.

## 6. Canonical Documentation Standard

Each package must contain a `doc.go` file with the standard header articulating its rationale over standard library equivalents:

```go
// Package pool provides a dynamic auto-scaling worker pool.
//
// Stdlib counterpart: sync.Pool / sync.WaitGroup
// Rejected compromise: Standard sync.WaitGroup does not limit concurrency or manage worker lifetimes;
// channels allocate per task when unbounded.
// Accepted cost: Background monitor goroutine for idle timeout eviction; Mutex overhead on queue spikes.
// Allocations: amortized
package pool
```

API documentation resides entirely within Go source files (`doc.go` and `example_*_test.go`). The `docs/` tree is strictly dedicated to architectural guides (`VISION.md`, `ARCHITECTURE.md`, `PERFORMANCE.md`), style guidelines, and Architecture Decision Records (`docs/adr/`).
