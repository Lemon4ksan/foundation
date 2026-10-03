# ADR-0002: Layering and dependency isolation

Status: Accepted
Date: 2026-10-03
Deciders: Lemon4ksan

## Context
As the repository grew, we risked uncontrolled dependency entanglement. Lower-level primitives could inadvertently pull in heavy high-level packages, and convenience abstractions could leak into zero-allocation packages, causing circular coupling and preventing targeted compilation.

## Decision
We group packages into layers and enforce acyclic, unidirectional dependencies.

- Layer 0: `silicon/*`
- Layer 1: `borrow`, `structures/*`, `refkit`, `bufkit`
- Layer 2: `sync/*`, `async/*`
- Layer 3: `codec/*`, `crypto/*`, `encoding/*`, `text/*`, `iokit`
- Layer 4: `net/*`
- Layer 5: `types/*`, `timekit`, `argkit`, `fskit`, `pathkit`, `tuikit`, `testing/*`, `cmd/*`
- External: `generic` (outside the graph: public for outside consumers, never imported by foundation packages).

Four laws govern imports:
1. Imports only point down the layer stack.
2. Internal packages do not import `generic`.
3. `silicon/*` packages import no other foundation package.
4. Allocation monotonicity: a package may not import one with a weaker allocation tier.

These are enforced today by `TestArchitecture_LayersAndIsolation` in `root_test.go`. The allocation monotonicity law is planned.

## Consequences
Dependency entanglement is impossible. The build fails immediately on violation. We pay by occasionally needing to duplicate trivial logic or extract it to a lower layer.
