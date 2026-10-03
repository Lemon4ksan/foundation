# ADR-0007: Baseline Statement Coverage Ratchet

- **Status**: Accepted
- **Date**: 2026-10-03
- **Author / Deciders**: Lemon4ksan
- **Target Packages**: Testing framework, CI workflows, all packages

## 1. Context & Problem Statement

The repository guidelines previously demanded "100% test coverage" across all packages. In practice, audit analysis revealed:
1. Significant portions of ported algorithms (e.g., LZ4, Zstandard, hardware SIMD branches, socket error handling) sat between 30% and 85% statement coverage.
2. Blanket "100%" rules with zero automated CI enforcement lead to cynical workarounds: either developers ignore the rule, or AI agents write meaningless dummy tests to game the metric without testing edge cases.
3. Completely blocking all forward progress until every legacy package reaches 100% is economically unviable and paralyzes maintenance.

## 2. Decision Drivers

- Enforce a strict "no regression" policy programmatically in CI.
- Convert test debt reduction into actionable, measurable Pull Requests.
- Require high coverage standards for all newly introduced packages without halting ongoing work.

## 3. Considered Options

1. **Option 1: Blanket Global Threshold (e.g., Module average >= 80%)**: Allows high-coverage packages to mask severe regressions in critical packages.
2. **Option 2: Strict 100% Immediate Gate**: Fails CI immediately for 50+ packages, blocking all development.
3. **Option 3: Per-Package Statement Coverage Ratchet**: Record the exact statement coverage of every package in a baseline file (`scripts/coverage_baseline.txt`). In CI, assert that no package drops below its baseline, and require >= 90% for newly added packages.

## 4. Decision Outcome

**Chosen Option**: **Option 3: Per-Package Statement Coverage Ratchet**.

### 4.1. Implementation Mechanics

1. **Baseline Tracking**: `scripts/coverage_baseline.txt` stores the exact statement coverage of every package.
2. **Automated Verification**: `scripts/coverage_ratchet/main.go` parses `coverage.out` generated during CI test execution.
   - If any existing package regresses by >0.05%, the check fails with a clear diff.
   - If a new package is introduced without baseline entry, it must achieve $\ge 90.0\%$ statement coverage.
   - If a Pull Request improves coverage, the author/agent updates the baseline, locking in the progress permanently.

## 5. Verification & Testing

Included in `make check` and the CI pipeline (`.github/workflows/ci.yml`). Any decrease in coverage immediately breaks the build.
