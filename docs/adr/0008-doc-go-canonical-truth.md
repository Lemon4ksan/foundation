# ADR-0008: Canonical Package Documentation via `doc.go`

- **Status**: Accepted
- **Date**: 2026-10-03
- **Author / Deciders**: Lemon4ksan
- **Target Packages**: Documentation tree (`docs/`), all packages

## 1. Context & Problem Statement

Prior repository structure maintained duplicate documentation channels:
- 40+ standalone Markdown files in `docs/<package>/<PACKAGE>.md`.
- `doc.go` files in a subset of packages (43 of 120 packages).
- Outdated links and inconsistent examples that diverged when APIs were refactored.

Maintaining documentation outside Go source code leads to inevitable rot: external Markdown files are not checked by the Go compiler or godoc, while AI agents frequently update one location and leave the other obsolete.

## 2. Decision Drivers

- Establish a Single Source of Truth for all API documentation.
- Leverage standard Go tooling (`godoc`, `pkg.go.dev`, compiler type-checking of `example_*_test.go`).
- Ensure all architecture guides (`docs/`) focus exclusively on repository-wide architecture, vision, ADRs, and engineering processes.

## 3. Considered Options

1. **Option 1: Maintain Both `doc.go` and `docs/*/*.md`**: Double the maintenance overhead.
2. **Option 2: Standalone `README.md` in Every Package**: Creates hundreds of Markdown files invisible to `pkg.go.dev` package overviews.
3. **Option 3: `doc.go` as Canonical Truth & Runnable Examples**: Place all package overview documentation in `doc.go` with executable `example_*_test.go` test suites.

## 4. Decision Outcome

**Chosen Option**: **Option 3: `doc.go` as Canonical Truth**.

### 4.1. Requirements for Package `doc.go`

Every package in `foundation` must feature a `doc.go` file containing:
1. High-level architectural narrative, thread-safety invariants, and lifecycle semantics.
2. The standard library v2.0 comparison block:
   ```go
   // Stdlib counterpart: <counterpart or "none — fills gap: ...">
   // Rejected compromise: <specific compromise rejected>
   // Accepted cost: <tradeoff accepted>
   // Allocations: zero | amortized | bounded(N/op) | unconstrained
   ```
3. Comprehensive, compilable `Example_` functions in `example_test.go` verifying idiomatic usage.

### 4.2. Role of the `docs/` Directory

`docs/` is reserved strictly for high-level repository governance:
- `docs/VISION.md`: Mission, boundaries, non-goals, and consumer map.
- `docs/ARCHITECTURE.md`: Low-level systems mechanics, SIMD pipelines, CPU cache alignment.
- `docs/PERFORMANCE.md`: Microbenchmark tables and hardware throughput logs.
- `docs/adr/`: Architecture Decision Records.

Legacy package-specific `.md` files in `docs/` are systematically migrated into `doc.go` and retired.

## 5. Verification & Testing

Architectural tests in `root_test.go` assert that every package contains a `doc.go` file with the required metadata block.
