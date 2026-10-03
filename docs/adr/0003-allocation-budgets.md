# ADR-0003: Memory allocation tiers

Status: Accepted
Date: 2026-10-03
Deciders: Lemon4ksan

## Context
Uncontrolled heap allocation causes latency jitter and GC spikes. An unnuanced "zero-allocation everywhere" rule is technically unsound: dynamic structures need to allocate on growth, and offline tools do not benefit from zero-allocation constraints.

## Decision
We classify packages into four strict memory allocation tiers. Every package declares its tier in `doc.go`.

- `zero`: 0 allocations per operation.
- `amortized`: Allocations permitted only on construction and capacity expansion.
- `bounded(N/op)`: Upper bound of N allocations per operation.
- `unconstrained`: No limits.

The declared tier is a ceiling; individual functions can be stricter. We link to this ADR instead of re-listing these definitions elsewhere.

## Consequences
Developers have clear, testable boundaries instead of vague guidelines. We pay by needing explicit destination buffers in `zero` APIs and tracking allocations in CI.
