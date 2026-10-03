# ADR-0003: Four-Tier Memory Allocation Contracts

- **Status**: Accepted
- **Date**: 2026-10-03
- **Author / Deciders**: Lemon4ksan
- **Target Packages**: Foundation-wide (all packages)

## 1. Context & Problem Statement

Uncontrolled heap allocation is the primary driver of latency jitter, GC pause spikes, and cache churn in high-throughput Go systems. The standard library frequently accepts allocations in return for ergonomics (e.g., `fmt.Sprintf`, `strings.Split`, `io.ReadAll`, returning interfaces).

While the original mandate for `foundation` was "zero-allocation", an unnuanced "100% zero-alloc everywhere" rule is technically unsound:
- Data structures (trees, hash tables, deques, heaps) must allocate backing arrays during initialization or capacity growth.
- Connection managers and async task executors inherently allocate metadata structures when establishing new connections.
- Offline tools, CLI flag parsers, and test assertions do not benefit from zero-allocation constraints.

Without precise categorization, developers and AI agents either write contorted code or violate the guideline without accountability.

## 2. Decision Drivers

- Provide rigorous, measurable, and testable allocation bounds.
- Distinguish hot-path streaming from initialization and cold utility paths.
- Enforce allocation budgets via automated tests (`testing.AllocsPerRun`).

## 3. Considered Options

1. **Option 1: Blanket Zero-Alloc Mandate**: Require 0 B/op on all functions in the repository without exception. (Unrealistic for dynamic structures and CLI).
2. **Option 2: Binary Split (Hot vs Cold)**: Loose binary distinction with undefined boundaries.
3. **Option 3: Four-Tier Allocation Taxonomy**: Explicit tiers declared per-package in `doc.go` with automated verification.

## 4. Decision Outcome

**Chosen Option**: **Option 3: Four-Tier Allocation Taxonomy**.

### 4.1. The Four Allocation Tiers

| Tier | Contract | Target Subsystems | Verification Requirement |
| :--- | :--- | :--- | :--- |
| **`zero`** | Strictly **0 B/op, 0 allocs/op** on every single invocation. Callers supply destination memory (`Append*`, `dst []byte`). | `silicon/*`, `encoding/*`, byte parsers, `borrow` | `testing.AllocsPerRun(100, fn) == 0` |
| **`amortized`** | Allocations permitted only on construction and capacity expansion. Strictly **0 allocs/op** in steady-state operations. | `structures/*`, `silicon/pool`, `bufkit`, `codec/*` | `AllocsPerRun == 0` after warmup |
| **`bounded(N)`** | Upper bound of at most $N$ heap allocations per completed transaction/session (exact $N$ documented in `doc.go`). | `async/task`, `net/quic` (per-connection setup) | `AllocsPerRun <= N` |
| **`unconstrained`** | No allocation limits. Prioritizes developer clarity, safety, and cold execution simplicity. | `argkit`, `fskit`, `cmd/c2plan9`, `testing/*` | Goroutine leak verification |

### 4.2. Invariants & Rules

- **Ceiling Principle**: The package-level tier declared in `doc.go` represents the *loose ceiling*. Individual exported functions within the package may be strictly zero-allocation even if the package is `amortized` or `bounded`.
- **Monotonicity**: Packages in the `zero` tier **cannot** import packages with `amortized` or `bounded` contracts.

## 5. Verification & Testing

Every package declaring `zero` or `amortized` contracts must provide unit tests using `testing.AllocsPerRun` to assert that hot-path functions perform zero allocations. Benchmarks run in CI with `-benchmem` confirm zero unexpected heap allocations.
