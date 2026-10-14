# ADR-0007: Per-package statement coverage ratchet and monotonic quality enforcement

Status: Accepted  
Date: 2026-10-03  
Deciders: Lemon4ksan  

## Context

Maintaining test coverage across a large systems substrate presents two opposing engineering pitfalls:

1. **The Fallacy of Blanket Targets:** Mandating a single global threshold (e.g. "repository coverage must exceed 80%") provides false confidence. Under a global threshold, low-level critical packages (such as `crypto/*`, `borrow`, or `silicon/simd`) can quietly suffer regressions from 98% down to 70% during refactors, masked by harmless test expansions in superficial utility packages.
2. **The Paralysis of Blanket 100% Mandates:** Mandating 100% line coverage upfront blocks ongoing feature development, encourages developers to write low-value "vanity tests" that assert trivially obvious getter/setter paths, and produces brittle mock setups that slow refactoring.
3. **The Silent Erosion of Quality:** Without automated enforcement, code coverage deteriorates monotonically. Engineers refactor code, delete outdated tests without replacing them, or add untested branches to handle edge cases, gradually degrading the reliability of the substrate.
4. **Third-Party SaaS Flakiness:** Delegating coverage gating to external SaaS platforms (such as Codecov or Coveralls) introduces brittle external network dependencies into CI, requires API token management, and exposes internal repository topology.

## Decision

We establish an automated, zero-dependency, per-package **monotonic statement coverage ratchet** integrated directly into our CI pipeline via [`scripts/coverage_ratchet`](../../scripts/coverage_ratchet/main.go) and version-controlled baseline table [`scripts/coverage_baseline.txt`](../../scripts/coverage_baseline.txt).

### Ratchet Architecture and Rules

```text
       PR Commit / CI Trigger
                 │
                 ▼
    go test -coverprofile=coverage.out ./...
                 │
                 ▼
    go run ./scripts/coverage_ratchet -check
                 │
        ┌────────┴────────┐
        ▼                 ▼
 Existing Package?     New Package?
        │                 │
  Coverage < Base?    Coverage < 90%?
     ├── YES ──► FAIL    ├── YES ──► FAIL
     └── NO  ──► PASS    └── NO  ──► PASS
```

1. **Per-Package Baseline Ledger:**
   The exact statement coverage percentage of every package is tracked in `scripts/coverage_baseline.txt`. This file is committed to git, establishing a transparent historical record of package quality.
2. **Zero-Regression Rule (Strict Monotonicity):**
   In CI (`.github/workflows/ci.yml`), `coverage_ratchet` compares `coverage.out` against `coverage_baseline.txt`.
   - If statement coverage for any existing package drops by more than **0.05%** (a small margin to account for floating-point rounding), the CI build fails with exit code 1.
   - Regressions cannot be merged into `main`.
3. **High-Bar Entry for New Packages:**
   Any newly introduced package not present in the baseline must achieve a minimum of **90.0% statement coverage** on its first appearance (`-min-new-coverage=90.0`). Incomplete or untested packages are rejected at inception.
4. **Ratchet Lock-In on Improvements:**
   When a pull request refactors or expands tests and increases package coverage, the author runs:
   ```bash
   go run ./scripts/coverage_ratchet -update
   ```
   This command updates `scripts/coverage_baseline.txt`. Once merged to `main`, the new higher coverage percentage becomes the permanent floor. Coverage can only move in one direction: upward.

## Consequences

### What gets easier:
- **Permanent Reliability Gains:** Code coverage is a one-way ratchet. Once a package reaches 95% or 100%, it can never silently regress.
- **Granular Accountability:** Regressions are pinpointed down to the exact package and percentage delta in CI output (`[REGRESSION] pkg dropped from 95.2% to 92.1% (delta: -3.1%)`).
- **Zero External SaaS Dependencies:** The ratchet is executed locally and in CI using pure Go standard library tools (`go test -coverprofile` and `bufio`), eliminating third-party API keys and external downtime.

### What gets harder / what we pay:
- Refactoring internal logic that changes statement counts may require authors to run `coverage_ratchet -update` if coverage drops below the tolerance margin, forcing conscious review of test thoroughness.
- Authors must write comprehensive tests upfront when introducing any new package to clear the 90.0% entry gate.

## Alternatives Considered

- **Global Repository-Wide Coverage Gate:** Rejected. Hides regressions in core subsystems behind high-volume test cases in secondary packages.
- **Third-Party Coverage SaaS (Codecov/Coveralls):** Rejected. Adds external network latency, sporadic rate-limiting, and supply chain exposure.
- **Manual PR Review of Test Coverage:** Rejected. Human reviewers cannot accurately calculate statement coverage percentages across complex diffs without automated tooling.
