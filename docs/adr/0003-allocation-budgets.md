# ADR-0003: Memory allocation tiers and runtime allocation budgets

Status: Accepted  
Date: 2026-10-03  
Deciders: Lemon4ksan  

## Context

In high-throughput, latency-critical Go systems, garbage collection (GC) pauses, mark-assist cycles, and heap churn represent the primary source of tail-latency spikes (p99/p99.9). Even with modern concurrent collectors, heap allocations impose steep microarchitectural penalties:
1. **GC Workload and Mark Assist:** Every heap object must be tracked, scanned, and swept. Under heavy traffic, goroutines performing allocations are forced by the runtime into "mark assist", stealing CPU cycles directly from request processing.
2. **Memory Subsystem Thrashing:** Heap allocation requires traversal of per-P `mcache` span lists, atomic contention on `mcentral`, and frequent pointer chasing that evicts CPU L1/L2 caches.
3. **Escapes and Hidden Allocations:** Subtle Go compiler escape analysis rules (e.g. passing a concrete type to an `any` parameter, closure variable capture, or slice resizing) routinely trigger silent heap escapes on hot paths.

At the same time, a dogmatic, blanket "zero-allocation everywhere" mandate across an entire repository is technically unsound:
- Dynamic data structures (trees, hash maps, ring buffers) must allocate during initial sizing and capacity expansion.
- Startup configuration loaders, offline code generators, and test runners derive negligible benefit from zero-allocation constraints, while paying heavily in API complexity.

Without explicit, formally classified allocation boundaries, systems inevitably slide toward two failure modes: either uncontrolled allocation drift on hot paths or unusable, over-engineered APIs across offline utilities.

## Decision

We establish four immutable memory allocation tiers for all packages in `foundation`. Every package declares its primary allocation tier in its canonical `doc.go` header ([ADR-0008](0008-doc-go-canonical-truth.md)).

### The Four Allocation Tiers

```text
┌─────────────────┬─────────────────────────────────────────────────────────────────┐
│ Tier            │ Invariant Contract                                              │
├─────────────────┼─────────────────────────────────────────────────────────────────┤
│ zero            │ 0 B/op and 0 allocs/op in steady-state operation.               │
│ amortized       │ 0 allocs/op during active throughput; allocations permitted     │
│                 │ strictly during construction and capacity growth.               │
│ bounded(N/op)   │ Strict upper bound of at most N heap allocations per call.      │
│ unconstrained   │ No allocation restrictions. Permitted for offline tools/tests.  │
└─────────────────┴─────────────────────────────────────────────────────────────────┘
```

1. **`zero`**:
   - **Contract:** Exactly 0 bytes and 0 allocations per operation in steady state.
   - **Target Packages:** Low-level silicon kernels, cryptographic primitives, binary serializers/deserializers, text scanners, bitwise manipulations, and ring queues (`silicon/*`, `encoding/bin`, `encoding/base64`, `crypto/*`, `bufkit`, `borrow`).
   - **Mechanics:** APIs must operate via caller-provided destination buffers (`dst []byte`), fixed-size arrays, lexical stack arenas, or generational lease handles (`borrow.Slot[T]`). Functions must never return freshly allocated slices or box values into interfaces.
2. **`amortized`**:
   - **Contract:** Zero allocations during ongoing steady-state execution. Allocations are strictly confined to initial allocation/construction (`New...`), buffer acquisition, and dynamic capacity doubling when limits are exceeded.
   - **Target Packages:** Vectored I/O queues (`iokit.BufferQueue`), memory pools (`silicon/pool`), dynamic data structures (`structures/*`), and deduplication filters (`async/dedup`).
   - **Mechanics:** Once initialized with sufficient capacity, subsequent read, write, scan, and iteration operations must remain strictly 0 allocs/op.
3. **`bounded(N/op)`**:
   - **Contract:** An invariant mathematical ceiling of at most $N$ heap allocations per operation.
   - **Target Packages:** Complex wire protocol parsers, metadata decoders, and structured logging (`net/dns/svcb`, `async/logkit`, `net/urlkit`).
   - **Mechanics:** Used where allocating a small, bounded wrapper or slice descriptor is algorithmically required, but unbounded recursion or per-field heap allocations remain strictly prohibited.
4. **`unconstrained`**:
   - **Contract:** Free use of standard heap allocation patterns.
   - **Target Packages:** Offline code generators (`c2plan9` wrappers), testing frameworks (`testing/*`), filesystem utilities (`fskit`), and development scripts (`scripts/*`).

### Allocation Monotonicity Law

In accordance with [ADR-0002](0002-layering-and-isolation.md), allocation budgets are monotonic: a package declared with a stricter tier may not import or delegate hot-path operations to a package with a looser tier.

### Automated Verification

Allocation guarantees are not treated as passive guidelines; they are programmatically verified in the test suite:
- Unit tests for `zero` and `amortized` routines employ `testing.AllocsPerRun(1000, fn)` asserting that allocations equal `0`.
- Continuous benchmark runs (`go test -bench=. -benchmem ./...`) track allocation regression in CI.
- The `doc.go` declaration is parsed and validated by `TestArchitecture_PublicPackagesDocGo` in [`root_test.go`](../../root_test.go).

## Consequences

### What gets easier:
- **Zero GC Jitter:** Hot execution paths (hashing, binary framing, ring queues, borrow leases) produce zero GC work, guaranteeing predictable microsecond p99 latencies.
- **Architectural Clarity:** Consumers instantly know how to use an API: `zero` requires destination buffers; `amortized` requires sizing up front; `unconstrained` can be used freely without manual buffer management.
- **Regression Prevention:** Hidden escapes introduced by compiler updates or code changes are caught instantly by allocation benchmark tests.

### What gets harder / what we pay:
- APIs in the `zero` tier require caller-managed buffers (`AppendX(dst []byte, ...)` instead of `X() []byte`).
- Writing and reviewing code requires vigilant escape analysis checking (`go build -gcflags="-m"`).
- Developers must maintain allocation regression benchmarks for every exported hot path.

## Alternatives Considered

- **Blanket "Zero Allocation Everywhere" Rule:** Rejected. Unrealistic and counter-productive for startup configuration, filesystem walks, and CLI tooling. It leads developers to circumvent the rule via ugly unsafe hacks or dummy wrappers.
- **Informal Prose Guidelines:** Rejected. Without formal tier names, explicit `doc.go` tags, and automated `testing.AllocsPerRun` assertions, allocation creep inevitably pollutes high-performance codebases over time.
- **Experimental Go Arena Allocators:** Evaluated and rejected. Go arenas (`arena.NewArena`) remain experimental, lack stability guarantees, and are subject to runtime deprecation or breaking changes.
