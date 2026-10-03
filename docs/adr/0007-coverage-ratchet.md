# ADR-0007: Baseline statement coverage ratchet

Status: Accepted
Date: 2026-10-03
Deciders: Lemon4ksan

## Context
A blanket "100% coverage" rule without automated enforcement leads to dummy tests or developers ignoring the rule entirely. Blocking work until every legacy package reaches 100% paralyzes development.

## Decision
We enforce a strict "no regression" policy programmatically in CI. We record the exact statement coverage of every package in `scripts/coverage_baseline.txt`. The `scripts/coverage_ratchet` tool parses `coverage.out` during CI runs.

If an existing package regresses, the build fails. If a new package is added, it must achieve at least 90.0% coverage. When a PR improves coverage, the author updates the baseline, locking in the progress permanently.

## Consequences
We stop coverage decay without blocking ongoing feature work. We pay by having to update the baseline file whenever we improve tests.
