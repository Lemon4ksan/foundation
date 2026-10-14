# ADR-0001: Foundation as Go standard library v2.0

Status: Accepted  
Date: 2026-10-03  
Deciders: Lemon4ksan  

## Context

Go's standard library (`stdlib`) is celebrated for its stability, backwards compatibility, and simplicity. However, these very strengths impose severe architectural compromises for mission-critical, latency-sensitive systems:

1. **Lowest-Common-Denominator Portability:** Standard library packages must run identically on embedded 32-bit MIPS systems, WASM runtimes, and high-end server hardware. Consequently, algorithms cannot freely leverage modern CPU instruction sets (AVX2, AVX-512, ARM NEON, GFNI) without complex runtime feature probes that often degrade into un-vectorized scalar loops.
2. **Allocation & Interface Boxing Invariants:** Go 1 compatibility guarantees freeze flawed API designs. Common interfaces accept `any` / `interface{}` or return newly allocated heap slices (e.g. `strings.Split`, `time.Now().Format`, `json.Unmarshal`), triggering inescapable GC allocations and pointer chasing on request hot paths.
3. **Absence of Memory Ownership Models:** Standard synchronization primitives (`sync.Pool`) offer zero defense against use-after-free, buffer races, or slice aliasing once a buffer is returned to a pool, leading to catastrophic data corruption in concurrent pipelines.

For systems demanding hundreds of thousands of operations per second per CPU core with sub-microsecond latency, standard packages become an insurmountable bottleneck.

## Decision

We design and maintain **`foundation`** as what the Go standard library would look like without these portability and legacy compromises — an uncompromising systems substrate ("stdlib v2.0") for the author's high-throughput ecosystem.

We establish the following foundational principles:

1. **Mechanical Sympathy First:** Hardware realities dictate software architecture. Data structures and algorithms are cache-line aligned (64 bytes), branch-misprediction optimized, and SIMD-vectorized for Tier-1 architectures (`amd64` and `arm64`).
2. **Explicit Allocation Tiers:** APIs favor caller-provided destination buffers, lexical stack arenas, and generational borrow scopes. Every package declares an explicit allocation tier ([ADR-0003](0003-allocation-budgets.md)) ensuring zero allocations on hot paths.
3. **Zero Reflection & Zero Interface Boxing:** Request fast paths never use `reflect` or box primitives into `any`. Type safety and zero-cost abstraction are achieved via Go 1.27+ generics and precompiled struct layout descriptors.
4. **Autonomous API Evolution:** We maintain full autonomy over package contracts. When standard library signatures (e.g. `time.Time`, `net.Buffers`, `sync.Map`) force allocations or hide concurrency hazards, `foundation` defines idiomatic, allocation-free alternatives ([ADR-0008](0008-doc-go-canonical-truth.md)).

## Consequences

### What gets easier:
- **Deterministic Latency & High Throughput:** Core subsystems (buffers, ring queues, UUIDs, varints, text casing, crypto primitives) operate with 0 B/op and 0 allocs/op, completely removing GC pressure.
- **Cross-Project Synergy:** Downstream network reactors (`mach`, `sein`), file format engines (`seal`), and TUI frameworks (`ctrl`) share a unified, ultra-reliable foundation.
- **Architectural Clarity:** Strict layered isolation ([ADR-0002](0002-layering-and-isolation.md)) prevents package spaghetti.

### What gets harder / what we pay:
- APIs require explicit memory management discipline (passing destination slices, respecting borrow scopes).
- We require modern Go compilers (Go 1.27+) and drop support for obsolete architectures.
- Breaking API refinements can occur prior to v1.0, requiring coordinated updates across downstream consumers.

## Alternatives Considered

- **Scattered Ad-Hoc Utilities:** Maintaining duplicated helper packages inside each consuming repo (`sein`, `aoni`, `seal`). Rejected: leads to divergence, duplicate debugging, and fragmented optimization work.
- **Forking the Go Compiler & Standard Library:** Creating a custom Go fork with modified stdlib packages. Rejected: untenable maintenance burden and incompatibility with standard tooling (`go build`, `golangci-lint`, IDEs).
- **Third-Party Dependency Grab Bag:** Relying on disparate open-source libraries. Rejected: introduces supply chain vulnerabilities, bloated transitive dependency graphs, and inconsistent allocation behavior.
