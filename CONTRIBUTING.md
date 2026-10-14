# Contributing to foundation

Thank you for considering a contribution! `foundation` is Go's standard library without compromises: a silicon-grade foundation for mission-critical systems engineering. We welcome performance improvements, bug fixes, and well-designed primitives that adhere to our strict architectural principles.

## Ground Rules

- **Zero Unintended Allocations.** Low-level packages must respect their declared allocation budgets ([ADR-0003](docs/adr/0003-allocation-budgets.md)). Never hide a heap allocation behind `any` or reflection on a hot path.
- **Layering & Isolation.** Imports strictly flow downward ([ADR-0002](docs/adr/0002-layering-and-isolation.md)). Primitives in `silicon/*` import nothing else from foundation, and nothing imports `generic`.
- **No L7 Protocols or Business Logic.** HTTP, gRPC, and database drivers belong in consuming projects (`mach`, `sein`, `dawn`), not here.
- **Canonical Documentation.** `doc.go` and `example_test.go` are the single source of truth for package documentation ([ADR-0008](docs/adr/0008-doc-go-canonical-truth.md)).
- **Coverage Ratchet.** Test coverage must never decrease. New primitives must be backed by `-race` unit tests, and parsers/decoders must include fuzz targets.

## Getting Started

```bash
git clone https://github.com/lemon4ksan/foundation
cd foundation
go mod download
make test   # run unit tests
make race   # run race detector
make lint   # run golangci-lint
```

## Workflow

1. **Fork** the repository and create a feature branch from `main`.
2. Write code following repository standards and ADR specifications.
3. Verify that `make check` (lint, race tests, architecture tests) passes with 0 errors.
4. Submit a Pull Request targeting `main` using the provided PR template.

## Commit Style

Use [Conventional Commits](https://www.conventionalcommits.org/):

```text
feat(arena): add concurrent slab allocator with cache-line alignment
fix(atomic): eliminate potential torn read in 64-bit counter on ARM32
perf(simd): vectorize byte search kernel with AVX-512 VBMI2
docs(doc.go): document lexical borrow scope lifetime invariants
```

## Tests & Quality

| Command | Purpose |
|---|---|
| `make test` | Fast unit tests |
| `make race` | Tests with Go race detector |
| `make bench` | Silicon throughput and allocation microbenchmarks |
| `make fuzz` | Continuous wire parser and decoder fuzzing |
| `make lint` | Run `golangci-lint` verification |
| `make format` | Format Go code and enforce BSD-3-Clause license headers |

## Architectural Decisions

If your proposed change introduces a new package, alters a layer boundary, or impacts allocation budgets, please discuss it first by opening an issue or drafting an [ADR](docs/adr/).
