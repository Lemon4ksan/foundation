# ADR-0002: Layering and dependency isolation

Status: Accepted  
Date: 2026-10-03  
Deciders: Lemon4ksan  

## Context

As a foundational systems substrate grows to encompass dozens of packages across memory management, concurrency, cryptographic primitives, and network protocols, it faces a lethal risk: **uncontrolled dependency entanglement**.

Without rigid structural boundaries:
1. Low-level memory primitives (e.g. CPU detection or ring buffers) inadvertently import high-level packages (e.g. formatting or logging), inflating binary sizes and preventing standalone compilation.
2. Convenience abstractions (such as generic monads `Optional[T]` or `Result[T]`) leak into hot substrate loops, introducing interface boxing, heap escapes, and unnecessary indirection into microsecond-critical code.
3. Cyclic imports and fragile initialization orders (`init()` races) proliferate, creating maintenance paralysis.

## Decision

We establish an acyclic, strictly unidirectional 6-layer architecture for all packages in `foundation`, enforced programmatically via automated architectural tests in [`root_test.go`](../../root_test.go).

### The Layer Hierarchy

```text
Layer 5: Tooling & Utilities      [types/*, timekit, fskit, pathkit, testing/*, scripts/*]
   ▲
Layer 4: Wire & Transport         [net/* (net/ip, net/dns, net/urlkit, net/tls)]
   ▲
Layer 3: Codecs & Cryptography    [codec/*, crypto/*, encoding/*, text/*, iokit]
   ▲
Layer 2: Concurrency & Scheduling [sync/*, async/*]
   ▲
Layer 1: Memory & Structures      [borrow, structures/* (bitset, deque, disjointset, linkedlist, minheap, ringbuffer, trie), refkit, bufkit]
   ▲
Layer 0: Silicon Substrate        [silicon/* (bytesconv, clock, comkit, cpukit, endian, hexkit, offheap, pool, randkit, ringbuf, simd, sysnet)]
-----------------------------------------------------------------------------------------
Isolated Boundary: generic/       (Public high-level monads & slice helpers for consumers;
                                   NEVER imported by any package inside foundation)
```

### The Four Invariant Laws of Isolation

1. **Downward Import Rule:** Imports must strictly flow downward in the layer stack (`Layer N` may only import `Layer M` where `M < N`). Packages within the same layer may import siblings only if no cyclic dependency exists.
2. **Generic Isolation Invariant:** Internal packages within `foundation` must never import `github.com/lemon4ksan/foundation/generic`. Low-level APIs return standard Go tuples `(T, bool)` or `(T, error)`. Callers who desire monadic chains or functional transformations wrap results at their own callsite.
3. **Silicon Autonomy Invariant:** Primitives in `silicon/*` (Layer 0) must never import any other package from `foundation`. They depend solely on Go standard library primitives and direct assembly/syscalls.
4. **Allocation Monotonicity Law:** A package with a strict allocation budget ([ADR-0003](0003-allocation-budgets.md)) may not import a package with a weaker allocation tier on its hot path.

### Enforcement

The rules are validated on every build and CI run by `TestArchitecture_LayersAndIsolation` in `root_test.go`. The test inspects the package dependency graph via `go list -json ./...` and immediately fails the build if any upward import, `generic` leak, or cross-layer violation is introduced.

## Consequences

### What gets easier:
- **Zero Circular Dependencies:** The dependency graph is guaranteed to be a Directed Acyclic Graph (DAG).
- **Targeted Modular Linking:** Consuming projects requiring only Layer 0 (`silicon/simd`) or Layer 1 (`bufkit.RingBuffer`) can import them without pulling network stacks, cryptographic routines, or reflection parsers.
- **Architectural Regression Prevention:** Contributor PRs and AI-generated code cannot secretly compromise layer boundaries; violations fail tests instantly.

### What gets harder / what we pay:
- When a low-level package needs a utility routine available in a higher layer, the logic must either be factored downward into a lower layer or implemented locally within the package's allocation budget.
- Refactoring layer boundaries requires updating `getPackageLayer` in `root_test.go` and documenting rationale in an ADR.

## Alternatives Considered

- **Flat Package Structure:** Keeping all primitives at root or allowing unrestricted imports. Rejected: quickly deteriorates into an untestable web of circular dependencies.
- **Go `internal/` Package Visibility:** Using Go's built-in `internal/` directory mechanism. Rejected as insufficient: `internal/` only prevents external repository imports, but provides zero enforcement of layering within the repository itself.
